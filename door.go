package keeper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/w6xian/keeper/service"
	"github.com/w6xian/keeper/utils/fsm"

	"github.com/w6xian/sloth/v4"
	"github.com/w6xian/sloth/v4/option"
)

// ErrRunnerNotStarted runner 尚未启动（TryExecuteFromConfig 未调用或已结束）。
var ErrRunnerNotStarted = errors.New("keeper: runner not started")

type Door struct {
	ctx     context.Context
	svrConn *sloth.Connect
	addr    string
	wsPath  string // 仅 WebSocket 传输使用；TCP 传输下无意义（保留字段以免破坏调用方）
	wg      *sync.WaitGroup
	Name    string
	fsmStore fsm.IFSM

	runnerMu sync.Mutex
	runner   *keeperRunner

	childMu  sync.Mutex
	childCmd *exec.Cmd

	stopOnce sync.Once
	stopped  bool
}

func NewDoor(ctx context.Context, wg *sync.WaitGroup, options ...DoorOption) *Door {
	wg.Add(1)

	log.Printf("Door started %d", os.Getpid())

	d := &Door{
		ctx:  ctx,
		wg:   wg,
		Name: ".door",
	}
	for _, opt := range options {
		opt(d)
	}
	if d.ctx == nil {
		d.ctx = context.Background()
	}
	if d.addr == "" {
		d.addr = "127.0.0.1:8965"
	}
	d.wsPath = "/ws"

	// 服务端逻辑容器（ClientRpc 指"调用目标是客户端"，本进程扮演服务端）
	clientRpc := sloth.DefaultServer()
	d.svrConn = sloth.ServerConn(clientRpc)

	// Register RPC Service
	if err := d.svrConn.Register("command", service.NewCommand(wg, d), ""); err != nil {
		log.Printf("Failed to register Command RPC %v", err)
	}
	// Register Registry Service
	if err := d.svrConn.Register("registry", service.NewRegistryService(), ""); err != nil {
		log.Printf("Failed to register Registry RPC %v", err)
	}
	// Register Script Service
	if err := d.svrConn.Register("script", service.NewScriptService(), ""); err != nil {
		log.Printf("Failed to register Script RPC %v", err)
	}
	if d.fsmStore != nil {
		// Register Cache Service
		if err := d.svrConn.Register("cache", service.NewCache(d.fsmStore), ""); err != nil {
			log.Printf("Failed to register Cache RPC %v", err)
		}
	}

	return d
}

// Start 监听并阻塞服务（TCP 传输）。
//
// PID 文件写入失败直接返回错误：旧实现在这里 log.Fatalf + os.Exit，
// 库代码替调用方决定"进程该死"是越权的，调用方可能还有自己的清理逻辑。
func (d *Door) Start(opts ...option.ConnectOption) error {
	pidManager := NewPIDManager(pidFilePath(d.Name))
	if err := pidManager.WritePID(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	// TCP 不需要 HTTP 路由：WithRouter 会 http.Handle(path, router)，
	// 重复注册同一 pattern 会直接 panic，这里不再注入。
	if err := d.svrConn.Listen(d.ctx, sloth.TCP, d.addr, opts...); err != nil {
		return fmt.Errorf("failed to listen %s: %w", d.addr, err)
	}
	if err := d.svrConn.Serve(); err != nil {
		return fmt.Errorf("serve %s: %w", d.addr, err)
	}
	return nil
}

// Execute 拉起子进程（默认 keeper app），阻塞等待其结束。
//
// 失败不再 log.Fatalf（那会直接杀掉调用方进程），错误信息交给返回值。
func (d *Door) Execute(args ...string) string {
	addr, err := d.ExecuteE(args...)
	if err != nil {
		log.Printf("Door execute failed: %v", err)
	}
	return addr
}

// ExecuteE 同 Execute，但把错误交给调用方。
func (d *Door) ExecuteE(args ...string) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}
	cmdArgs := append([]string{}, args...)
	if len(cmdArgs) == 0 {
		cmdArgs = append(cmdArgs, "app")
	}
	// Append port and path arguments
	finalArgs := append(cmdArgs, "--port", d.addr, "--path", d.wsPath)
	cmd := exec.Command(exe, finalArgs...)
	d.childMu.Lock()
	d.childCmd = cmd
	d.childMu.Unlock()
	if os.Getenv("KEEPER_SERVICE") != "1" {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start child process: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return d.addr, fmt.Errorf("child process exited with error: %w", err)
	}
	return d.addr, nil
}

func (d *Door) TryExecuteFromConfig(c string) error {
	conf, err := initConfig(c)
	if err != nil {
		log.Printf("failed to init config: %v", err)
		return err
	}
	ordered, err := sortServices(conf.Services)
	if err != nil {
		log.Printf("failed to sort services: %v", err)
		return err
	}

	ctx, cancel := signal.NotifyContext(d.ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	runner := newKeeperRunner(ctx, ordered, d.addr, d.wsPath)
	d.setRunner(runner)
	defer d.setRunner(nil)
	if err := runner.run(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (d *Door) StopService(ctx context.Context, name string) error {
	runner, err := d.currentRunner()
	if err != nil {
		return err
	}
	return runner.StopService(name)
}

func (d *Door) StartService(ctx context.Context, name string) error {
	runner, err := d.currentRunner()
	if err != nil {
		return err
	}
	return runner.StartService(name)
}

func (d *Door) ReloadService(ctx context.Context, name string) error {
	runner, err := d.currentRunner()
	if err != nil {
		return err
	}
	return runner.ReloadService(name)
}

func (d *Door) currentRunner() (*keeperRunner, error) {
	d.runnerMu.Lock()
	runner := d.runner
	d.runnerMu.Unlock()
	if runner == nil {
		return nil, ErrRunnerNotStarted
	}
	return runner, nil
}

func (d *Door) setRunner(runner *keeperRunner) {
	d.runnerMu.Lock()
	d.runner = runner
	d.runnerMu.Unlock()
}

// Stop 停止 Door：关闭监听、终止子进程、删除 PID 文件。幂等，可重复调用。
func (d *Door) Stop() error {
	d.stopOnce.Do(func() {
		d.stopped = true
	})
	d.killChild()
	if d.svrConn != nil {
		// 关闭 listener 并等待 Serve 的 goroutine 退出（v4 的 Close 会等 serveWg）。
		if err := d.svrConn.Close(); err != nil {
			log.Printf("failed to close listener: %v", err)
		}
	}
	pidManager := NewPIDManager(pidFilePath(d.Name))
	if err := pidManager.RemovePID(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("failed to remove pid file %s: %v", pidManager.GetPIDFile(), err)
		return err
	}
	return nil
}

func (d *Door) killChild() {
	d.childMu.Lock()
	child := d.childCmd
	d.childCmd = nil
	d.childMu.Unlock()
	if child != nil && child.Process != nil {
		_ = child.Process.Kill()
	}
}
