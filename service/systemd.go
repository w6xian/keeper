//go:build linux

package service

import (
	"fmt"
	"os"
	"os/exec"
)

type Systemd struct {
	Name        string
	DisplayName string
}

func (s *Systemd) unitPath() string {
	return "/etc/systemd/system/" + s.Name + ".service"
}

func (s *Systemd) Install(binPath, token string) error {
	unit := fmt.Sprintf(`[Unit]
Description=%s
After=network.target

[Service]
ExecStart=%s --token %s
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
`, s.DisplayName, binPath, token)

	if err := os.WriteFile(s.unitPath(), []byte(unit), 0644); err != nil {
		return err
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}
	return exec.Command("systemctl", "enable", "--now", s.Name).Run()
}

func (s *Systemd) Uninstall() error {
	exec.Command("systemctl", "disable", "--now", s.Name).Run()
	return os.Remove(s.unitPath())
}

func (s *Systemd) Running() bool {
	return exec.Command("systemctl", "is-active", "--quiet", s.Name).Run() == nil
}

func New(name string, displayName string) Service {
	return &Systemd{
		Name:        name,
		DisplayName: displayName,
	}
}
