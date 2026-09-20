package keeper

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/shirou/gopsutil/process"
)

// pidFilePath returns the path to the PID file
func pidFilePath(pidFile string) string {
	return pidFile
}

// PIDManager handles PID file operations
type PIDManager struct {
	pidFile string
}

// NewPIDManager creates a new PIDManager instance
func NewPIDManager(pidFile string) *PIDManager {
	return &PIDManager{
		pidFile: pidFile,
	}
}

// NewPIDManagerFromConfig creates a new PIDManager instance from config
func NewPIDManagerFromConfig(pidFile string) *PIDManager {
	return NewPIDManager(pidFile)
}

// WritePID writes the current process ID to the PID file
func (p *PIDManager) WritePID() error {
	dir := filepath.Dir(p.pidFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create PID directory: %w", err)
	}
	// 文件已存在：先确认里面的 PID 是不是本程序还活着，是则拒绝重复启动，
	// 否则说明是上次崩溃留下的陈旧文件，清理掉继续。
	if _, err := os.Stat(p.pidFile); err == nil {
		file, err := os.Open(p.pidFile)
		if err != nil {
			return fmt.Errorf("failed to open PID file: %w", err)
		}
		// 用 defer 关：旧实现只在成功路径 Close，ReadAll 出错时会漏掉一个 fd。
		defer file.Close()

		pidBytes, err := io.ReadAll(file)
		if err != nil {
			return fmt.Errorf("failed to read PID file: %w", err)
		}
		pid := strings.TrimSpace(string(pidBytes))

		// 根据pid读进程
		pidInt, err := strconv.Atoi(pid)
		if err != nil {
			return fmt.Errorf("failed to convert PID to int---: %w", err)
		}
		pn, pErr := process.NewProcess(int32(pidInt))
		if pErr == nil {
			pName, _ := pn.Name()
			executablePath, _ := os.Executable()
			if strings.Contains(executablePath, pName) {
				return fmt.Errorf("PID file already exists: %s", p.pidFile)
			}
		}
		removeErr := os.Remove(p.pidFile)
		if removeErr != nil {
			return fmt.Errorf("failed to remove PID file: %w", removeErr)
		}
	}
	pid := os.Getpid()
	if err := os.WriteFile(p.pidFile, []byte(fmt.Sprintf("%d\n", pid)), 0644); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}
	return nil
}

// RemovePID removes the PID file
func (p *PIDManager) RemovePID() error {
	return os.Remove(p.pidFile)
}

// GetPIDFile returns the PID file path
func (p *PIDManager) GetPIDFile() string {
	return p.pidFile
}
