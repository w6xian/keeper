package console

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"time"
)

// Clear 清空控制台
func Clear() {
	std.Clear()
}

func (c *Console) Clear() {
	if c.disabled {
		return
	}

	// Windows 和 Unix 的清屏命令不同
	print("\033[H\033[2J")
}

// Context 保存当前的上下文（简化实现，仅输出日志）
func Context(name string, data interface{}) {
	std.Context(name, data)
}

func (c *Console) Context(name string, data interface{}) {
	if c.disabled {
		return
	}

	c.writef(c.output, colorCyan, "上下文 [%s]:", name)
	c.write(c.output, colorGray, data)
}

// Task 表示一个可追踪的任务
type Task struct {
	name      string
	startTime time.Time
	console   *Console
}

// CreateTask 创建一个可追踪的任务
func CreateTask(name string) *Task {
	return std.CreateTask(name)
}

func (c *Console) CreateTask(name string) *Task {
	if c.disabled {
		return &Task{name: name}
	}

	return &Task{
		name:      name,
		startTime: time.Now(),
		console:   c,
	}
}

// Run 运行任务
func (t *Task) Run(fn func()) {
	t.console.Infof("任务 '%s' 开始执行", t.name)
	fn()
	t.console.Infof("任务 '%s' 执行完成，耗时: %s", t.name, time.Since(t.startTime).String())
}

// Fail 标记任务失败
func (t *Task) Fail(reason string) {
	t.console.Errorf("任务 '%s' 失败: %s", t.name, reason)
}

// Success 标记任务成功
func (t *Task) Success(message string) {
	t.console.Infof("任务 '%s' 成功: %s", t.name, message)
}

// Memory 输出内存使用情况
func Memory() {
	std.Memory()
}

func (c *Console) Memory() {
	if c.disabled {
		return
	}

	// 获取内存统计
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.write(c.output, colorCyan, "内存使用情况:")
	c.writef(c.output, colorGray, "  已分配内存: %s", formatBytes(m.Alloc))
	c.writef(c.output, colorGray, "  总分配内存: %s", formatBytes(m.TotalAlloc))
	c.writef(c.output, colorGray, "  系统内存: %s", formatBytes(m.Sys))
	c.writef(c.output, colorGray, "  GC 次数: %d", m.NumGC)
	c.writef(c.output, colorGray, "  下一次 GC 目标: %s", formatBytes(m.NextGC))
	c.writef(c.output, colorGray, "  堆对象数量: %d", m.HeapObjects)
}

// MemoryJSON 返回内存统计的 JSON 格式
func MemoryJSON() string {
	return std.MemoryJSON()
}

func (c *Console) MemoryJSON() string {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	mem := MemoryStats{
		UsedJSHeapSize:  m.HeapInuse,
		TotalJSHeapSize: m.HeapAlloc,
		JSHeapSizeLimit: m.Sys,
	}

	data, err := json.MarshalIndent(mem, "", "  ")
	if err != nil {
		return "{}"
	}

	return string(data)
}

// UpdateMemory 更新内部内存统计
func (c *Console) updateMemory() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	c.memory = MemoryStats{
		UsedJSHeapSize:  m.HeapInuse,
		TotalJSHeapSize: m.HeapAlloc,
		JSHeapSizeLimit: m.Sys,
	}
}

// formatBytes 格式化字节数为人类可读格式
func formatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// Disable 禁用控制台输出
func Disable() {
	std.Disable()
}

func (c *Console) Disable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disabled = true
}

// Enable 启用控制台输出
func Enable() {
	std.Enable()
}

func (c *Console) Enable() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disabled = false
}

// IsDisabled 检查控制台是否被禁用
func IsDisabled() bool {
	return std.IsDisabled()
}

func (c *Console) IsDisabled() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.disabled
}

// SetOutput 设置输出目标
func SetOutput(w *os.File) {
	std.SetOutput(w)
}

func (c *Console) SetOutput(w *os.File) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.output = w
}

// GetStandard 获取标准控制台实例
func GetStandard() *Console {
	return std
}
