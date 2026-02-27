//go:build windows

package service

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/w6xian/keeper/internal/pathx"
)

type Windows struct {
	Name        string
	DisplayName string
}

func (w *Windows) Install(binPath, token string) error {
	binArg := fmt.Sprintf(`"%s" --path="%s" --token=%s`, binPath, pathx.GetCurrentAbPath(), token)
	if err := exec.Command("sc", "create", w.Name, "binPath=", binArg, "start=", "auto", "displayname=", w.DisplayName).Run(); err != nil {
		return fmt.Errorf("创建服务失败: %w", err)
	}
	return exec.Command("sc", "start", w.Name).Run()
}

func (w *Windows) Uninstall() error {
	exec.Command("sc", "stop", w.Name).Run()
	return exec.Command("sc", "delete", w.Name).Run()
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
