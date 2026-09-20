package keeper

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func longSleepCmd() string {
	if runtime.GOOS == "windows" {
		return "Start-Sleep -Seconds 60"
	}
	return "sleep 60"
}

func TestSortServicesOrdersDependenciesFirst(t *testing.T) {
	services := []Service{
		{Name: "web", After: "db,cache"},
		{Name: "cache", After: "db"},
		{Name: "db"},
	}
	got, err := sortServices(services)
	if err != nil {
		t.Fatalf("sortServices() error = %v", err)
	}
	names := make([]string, 0, len(got))
	for _, s := range got {
		names = append(names, s.Name)
	}
	want := []string{"db", "cache", "web"}
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("order = %v, want %v", names, want)
	}
}

func TestSortServicesRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name     string
		services []Service
		wantErr  string
	}{
		{"cycle", []Service{{Name: "a", After: "b"}, {Name: "b", After: "a"}}, "cycle"},
		{"duplicate", []Service{{Name: "a"}, {Name: "a"}}, "duplicate"},
		{"unknown dep", []Service{{Name: "a", After: "ghost"}}, "unknown service"},
		{"empty name", []Service{{Name: " "}}, "empty"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sortServices(tt.services)
			if err == nil {
				t.Fatalf("sortServices() error = nil, want %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
			}
		})
	}
}

// TestStartServiceInjectedArgs 覆盖"--port/--path 没传进业务进程"的缺陷：
// 参数必须出现在 shell 要执行的命令串里（sh -c "cmd --port X --path Y"），
// 而不是作为 sh 自己的 argv（那样会被当成脚本的 $0/$1）。
func TestStartServiceInjectedArgs(t *testing.T) {
	full := buildStartCommand("keeper app", "127.0.0.1:8965", "/ws")
	want := "keeper app --port 127.0.0.1:8965 --path /ws"
	if full != want {
		t.Fatalf("buildStartCommand() = %q, want %q", full, want)
	}
	args := shellArgs(full)
	if len(args) != 2 {
		t.Fatalf("shell args = %v, want exactly 2 elements (flag + command)", args)
	}
	if args[1] != want {
		t.Fatalf("shell command = %q, want %q", args[1], want)
	}
}

// TestStartServiceExecutesFullCommand 端到端验证：shell 真的执行了带参数的整条命令。
func TestStartServiceExecutesFullCommand(t *testing.T) {
	marker := filepath.Join(t.TempDir(), "marker.txt")
	// 末尾的注释符是必须的：拼接进来的 --port/--path 会被 shell 当参数解析，
	// 而 echo / Set-Content 并不接受它们（真实场景里由业务命令自行解析）。
	// 这样测的是"整条命令串确实被 shell 执行"，而不是被测命令是否支持这些参数。
	var writeCmd string
	if runtime.GOOS == "windows" {
		writeCmd = fmt.Sprintf("Set-Content -Path '%s' -Value 'ok' #", marker)
	} else {
		writeCmd = fmt.Sprintf("echo ok > %s #", marker)
	}

	cmd, err := startService(context.Background(), Service{Name: "marker", Start: writeCmd}, "127.0.0.1:8965", "/ws")
	if err != nil {
		t.Fatalf("startService() error = %v", err)
	}
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case <-done:
	case <-time.After(30 * time.Second):
		_ = cmd.Process.Kill()
		t.Fatal("startService() did not finish in time")
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("marker file not created: %v", err)
	}
}

// TestRunnerStopThenStart 回归"stop 之后再也 start 不起来"：
// stopService 消费了 exit 却不清空 cmd，StartService 会误判"还在跑"直接返回。
func TestRunnerStopThenStart(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	runner := newKeeperRunner(ctx, []Service{{Name: "sleeper", Start: longSleepCmd(), StopTimeout: 1}}, "127.0.0.1:0", "/ws")
	if err := runner.StartService("sleeper"); err != nil {
		t.Fatalf("StartService() error = %v", err)
	}
	if err := runner.StopService("sleeper"); err != nil {
		t.Fatalf("StopService() error = %v", err)
	}
	if err := runner.StartService("sleeper"); err != nil {
		t.Fatalf("StartService() after stop error = %v", err)
	}

	rt := runner.services[0]
	rt.mu.Lock()
	cmd := rt.cmd
	rt.mu.Unlock()
	if cmd == nil || cmd.Process == nil {
		t.Fatal("service was not actually started again after stop")
	}
	if err := runner.StopService("sleeper"); err != nil {
		t.Fatalf("StopService() error = %v", err)
	}
}

func TestRunnerServiceNotFound(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	runner := newKeeperRunner(ctx, []Service{{Name: "a"}}, "127.0.0.1:0", "/ws")
	for _, fn := range []func(string) error{runner.StartService, runner.StopService, runner.ReloadService} {
		if err := fn("ghost"); err == nil {
			t.Fatal("expected error for unknown service")
		}
		if err := fn(""); err == nil {
			t.Fatal("expected error for empty service name")
		}
	}
}

// TestRunnerConcurrentControl 并发压 Start/Stop/Reload + 后台 run 循环，
// 用 -race 暴露重启计数与进程句柄上的数据竞争。
func TestRunnerConcurrentControl(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	runner := newKeeperRunner(ctx, []Service{{Name: "sleeper", Start: longSleepCmd(), StopTimeout: 1}}, "127.0.0.1:0", "/ws")

	runErr := make(chan error, 1)
	go func() { runErr <- runner.run() }()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				switch (i + j) % 3 {
				case 0:
					_ = runner.StartService("sleeper")
				case 1:
					_ = runner.StopService("sleeper")
				default:
					_ = runner.ReloadService("sleeper")
				}
			}
		}(i)
	}
	wg.Wait()

	cancel()
	select {
	case <-runErr:
	case <-time.After(60 * time.Second):
		t.Fatal("runner.run() did not return after cancel")
	}
}
