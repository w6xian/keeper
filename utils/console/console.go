package console

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
)

var (
	std = New()
)

// Console 提供与 JavaScript console 类似的功能
type Console struct {
	mu           sync.RWMutex
	output       io.Writer
	errorOutput  io.Writer
	counts       map[string]int
	timers       map[string]time.Time
	profiles     map[string]time.Time
	groupLevel   int
	memory       MemoryStats
	disabled     bool
	enableColors bool
}

// MemoryStats 内存统计信息
type MemoryStats struct {
	UsedJSHeapSize  uint64 `json:"usedJSHeapSize"`
	TotalJSHeapSize uint64 `json:"totalJSHeapSize"`
	JSHeapSizeLimit uint64 `json:"jsHeapSizeLimit"`
}

// Option 配置选项
type Option func(*Console)

// WithOutput 设置输出目标
func WithOutput(w io.Writer) Option {
	return func(c *Console) {
		c.output = w
	}
}

// WithErrorOutput 设置错误输出目标
func WithErrorOutput(w io.Writer) Option {
	return func(c *Console) {
		c.errorOutput = w
	}
}

// WithColors 启用或禁用颜色
func WithColors(enable bool) Option {
	return func(c *Console) {
		c.enableColors = enable
	}
}

// New 创建新的 Console 实例
func New(opts ...Option) *Console {
	c := &Console{
		output:       os.Stdout,
		errorOutput:  os.Stderr,
		counts:       make(map[string]int),
		timers:       make(map[string]time.Time),
		profiles:     make(map[string]time.Time),
		enableColors: true,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// format 格式化输出内容
func (c *Console) format(args ...interface{}) string {
	return fmt.Sprint(args...)
}

// formatf 格式化输出内容
func (c *Console) formatf(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

// getColor 获取颜色字符串
func (c *Console) getColor(color string) string {
	if !c.enableColors {
		return ""
	}
	return color
}

// write 写入输出（不持有锁，调用者需要保证线程安全）
func (c *Console) write(w io.Writer, color string, args ...interface{}) {
	message := c.format(args...)
	if c.enableColors {
		message = c.getColor(color) + message + c.getColor(colorReset)
	}
	fmt.Fprintln(w, message)
}

// writef 写入格式化输出（不持有锁，调用者需要保证线程安全）
func (c *Console) writef(w io.Writer, color, format string, args ...interface{}) {
	message := c.formatf(format, args...)
	if c.enableColors {
		message = c.getColor(color) + message + c.getColor(colorReset)
	}
	fmt.Fprintln(w, message)
}

// writeWithColor 写入带颜色的消息（不持有锁，调用者需要保证线程安全）
func (c *Console) writeWithColor(w io.Writer, color string, message string) {
	if c.enableColors {
		message = c.getColor(color) + message + c.getColor(colorReset)
	}
	fmt.Fprintln(w, message)
}

// writeSafe 线程安全的写入（持有锁）
func (c *Console) writeSafe(w io.Writer, color string, args ...interface{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.write(w, color, args...)
}

// writefSafe 线程安全的格式化写入（持有锁）
func (c *Console) writefSafe(w io.Writer, color, format string, args ...interface{}) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	c.writef(w, color, format, args...)
}

// getCallerInfo 获取调用者信息
func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 1)
	if !ok {
		return ""
	}
	parts := strings.Split(file, "/")
	if len(parts) > 3 {
		file = strings.Join(parts[len(parts)-3:], "/")
	}
	return fmt.Sprintf("%s:%d", file, line)
}
