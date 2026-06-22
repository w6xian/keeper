package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/w6xian/keeper/example/cmd"
	"github.com/w6xian/keeper/service"
	"github.com/w6xian/keeper/utils/pathx"
)

func main() {
	if len(os.Args) <= 1 && runtime.GOOS == "windows" {
		name := cmd.DefaultServiceName()
		svc := service.New(name, cmd.DefaultDisplayName())
		if svc.Running() {
			return
		}

		startOut, startErr := exec.Command("sc", "start", name).CombinedOutput()
		if startErr == nil {
			return
		}

		if ee, ok := startErr.(*exec.ExitError); ok && ee.ExitCode() == 1060 {
			binPath := pathx.GetCaller()
			if err := svc.Install(binPath, "12358"); err == nil {
				return
			}
		}

		if err := svc.Install(pathx.GetCaller(), "12358"); err == nil {
			return
		}

		writeLauncherLog(fmt.Sprintf("sc start failed: %v: %s", startErr, string(startOut)))
		return
	}

	if len(os.Args) <= 1 {
		os.Args = append(os.Args, "--token", "12358")
	}
	// 其他情况，直接执行命令
	cmd.Execute()
}

func writeLauncherLog(msg string) {
	base := os.Getenv("PROGRAMDATA")
	if base == "" {
		base = "."
	} else {
		base = filepath.Join(base, "keeper")
	}
	_ = os.MkdirAll(base, 0755)
	f, err := os.OpenFile(filepath.Join(base, "launcher.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "%s %s\n", time.Now().Format(time.RFC3339), msg)
}
