//go:build windows

package service

import (
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/w6xian/keeper/utils/pathx"
)

type Windows struct {
	Name        string
	DisplayName string
}

func (w *Windows) Install(binPath, token string) error {
	binArg := fmt.Sprintf(`"%s" --path="%s" --token=%s`, binPath, pathx.GetCurrentAbPath(), token)
	createCmd := func() error {
		return exec.Command("sc", "create", w.Name, "binPath=", binArg, "start=", "auto", "displayname=", w.DisplayName).Run()
	}
	if err := createCmd(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			switch ee.ExitCode() {
			case 1072, 1073:
				for i := 0; i < 10; i++ {
					exec.Command("sc", "stop", w.Name).Run()
					exec.Command("sc", "delete", w.Name).Run()
					time.Sleep(1200 * time.Millisecond)
					if retryErr := createCmd(); retryErr == nil {
						return exec.Command("sc", "start", w.Name).Run()
					}
				}
				return fmt.Errorf("创建服务失败: %w（服务可能处于“标记删除”状态，请稍后重试或重启系统）", err)
			}
		}
		return fmt.Errorf("创建服务失败: %w", err)
	}
	return exec.Command("sc", "start", w.Name).Run()
}

func (w *Windows) Uninstall() error {
	exec.Command("sc", "stop", w.Name).Run()
	if err := exec.Command("sc", "delete", w.Name).Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			if ee.ExitCode() == 1060 {
				return nil
			}
		}
		return err
	}
	return nil
}

func (w *Windows) Running() bool {
	out, err := exec.Command("sc", "query", w.Name).Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "RUNNING")
}

func New(name string, displayName string) Service {
	return &Windows{
		Name:        name,
		DisplayName: displayName,
	}
}
