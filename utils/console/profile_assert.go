package console

import (
	"fmt"
	"runtime"
	"time"
)

// Profile 启动 CPU 性能分析器
func Profile(label string) {
	std.Profile(label)
}

func (c *Console) Profile(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if label == "" {
		label = "default"
	}

	if _, exists := c.profiles[label]; exists {
		c.writef(c.errorOutput, colorRed, "%s: 性能分析器已在运行", label)
		return
	}

	c.profiles[label] = time.Now()
	c.writef(c.output, colorCyan, "%s: 性能分析器已启动", label)
}

// ProfileEnd 结束 CPU 性能分析并输出结果
func ProfileEnd(label string) {
	std.ProfileEnd(label)
}

func (c *Console) ProfileEnd(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if label == "" {
		label = "default"
	}

	startTime, exists := c.profiles[label]
	if !exists {
		c.writef(c.errorOutput, colorRed, "%s: 性能分析器未启动", label)
		return
	}

	duration := time.Since(startTime)
	delete(c.profiles, label)
	
	c.writef(c.output, colorCyan, "%s: 性能分析完成", label)
	c.writef(c.output, colorGray, "  分析时长: %s", duration.String())
	c.write(c.output, colorGray, "  注: 此为简化实现，实际性能分析需要使用 pprof 包")
}

// Assert 如果断言为 false，输出错误信息和堆栈跟踪
func Assert(condition bool, message ...interface{}) {
	std.Assert(condition, message...)
}

func (c *Console) Assert(condition bool, message ...interface{}) {
	if c.disabled {
		return
	}

	if !condition {
		msg := "Assertion failed"
		if len(message) > 0 {
			msg = c.format(message...)
		}

		// 获取调用者堆栈
		pc, file, line, ok := runtime.Caller(1)
		if ok {
			fn := runtime.FuncForPC(pc)
			if fn != nil {
				msg = fmt.Sprintf("%s\n    at %s:%d\n    in %s", msg, file, line, fn.Name())
			}
		}

		c.write(c.errorOutput, colorRed, msg)
	}
}

// Assertf 格式化断言
func Assertf(condition bool, format string, args ...interface{}) {
	std.Assertf(condition, format, args...)
}

func (c *Console) Assertf(condition bool, format string, args ...interface{}) {
	c.Assert(condition, c.formatf(format, args...))
}

// Trace 输出堆栈跟踪
func Trace(args ...interface{}) {
	std.Trace(args...)
}

func (c *Console) Trace(args ...interface{}) {
	if c.disabled {
		return
	}

	if len(args) > 0 {
		c.write(c.output, colorPurple, c.format(args...))
	}

	c.write(c.output, colorPurple, "堆栈跟踪:")

	// 获取堆栈跟踪
	pcs := make([]uintptr, 32)
	n := runtime.Callers(2, pcs)
	if n == 0 {
		return
	}

	pcs = pcs[:n]
	frames := runtime.CallersFrames(pcs)

	for {
		frame, more := frames.Next()
		c.writef(c.output, colorGray, "  at %s (%s:%d)", frame.Function, frame.File, frame.Line)

		if !more {
			break
		}
	}
}
