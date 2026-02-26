//go:build windows

package daemon

import (
	"os/exec"
	"strconv"
	"strings"
)

// processRunning 检查进程是否存活（Windows: tasklist）
func processRunning(pid int) bool {
	out, err := exec.Command("tasklist", "/FI", "PID eq "+strconv.Itoa(pid), "/NH").Output()
	if err != nil {
		return false
	}
	return !strings.Contains(string(out), "No tasks")
}

// processKill 优雅终止进程（Windows: taskkill 不带 /F 发送关闭信号）
func processKill(pid int) error {
	return exec.Command("taskkill", "/PID", strconv.Itoa(pid)).Run()
}
