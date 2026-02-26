package daemon

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

var pidFile = "/casher.pid"

// Stop 停止 casher
func Stop() error {
	pid, err := readPID()
	if err != nil {
		return fmt.Errorf("未找到运行中的 casher")
	}
	if err := processKill(pid); err != nil {
		return fmt.Errorf("停止 casher 失败: %w", err)
	}
	os.Remove(pidFile)
	fmt.Println("casher 已停止")
	return nil
}

// Running 检查 casher 是否在运行
func Running() bool {
	pid, err := readPID()
	if err != nil {
		return false
	}
	return processRunning(pid)
}

// PID 返回当前运行的 PID
func PID() int {
	pid, _ := readPID()
	return pid
}

func readPID() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(data)))
}
