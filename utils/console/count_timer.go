package console

import (
	"fmt"
	"time"
)

// Count 记录并输出标签被调用的次数
func Count(label string) {
	std.Count(label)
}

func (c *Console) Count(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if label == "" {
		label = "default"
	}

	c.counts[label]++
	c.writef(c.output, colorCyan, "%s: %d", label, c.counts[label])
}

// CountReset 重置指定标签的计数器
func CountReset(label string) {
	std.CountReset(label)
}

func (c *Console) CountReset(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if label == "" {
		label = "default"
	}

	if _, exists := c.counts[label]; exists {
		c.counts[label] = 0
		c.writef(c.output, colorCyan, "%s: 计数器已重置", label)
	} else {
		c.writef(c.errorOutput, colorRed, "%s: 计数器不存在", label)
	}
}

// Time 启动一个计时器
func Time(label string) {
	std.Time(label)
}

func (c *Console) Time(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if label == "" {
		label = "default"
	}

	if _, exists := c.timers[label]; exists {
		c.writef(c.errorOutput, colorRed, "%s: 计时器已存在", label)
		return
	}

	c.timers[label] = time.Now()
	c.writef(c.output, colorGreen, "%s: 计时器已启动", label)
}

// TimeEnd 结束计时器并输出耗时
func TimeEnd(label string) {
	std.TimeEnd(label)
}

func (c *Console) TimeEnd(label string) {
	if c.disabled {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if label == "" {
		label = "default"
	}

	startTime, exists := c.timers[label]
	if !exists {
		c.writef(c.errorOutput, colorRed, "%s: 计时器不存在", label)
		return
	}

	duration := time.Since(startTime)
	delete(c.timers, label)
	
	durationStr := duration.String()
	if duration.Milliseconds() < 1 {
		durationStr = fmt.Sprintf("%.3f μs", float64(duration.Nanoseconds())/1000)
	} else if duration.Milliseconds() < 1000 {
		durationStr = fmt.Sprintf("%.2f ms", float64(duration.Microseconds())/1000)
	}
	
	c.writef(c.output, colorGreen, "%s: %s", label, durationStr)
}

// TimeLog 输出计时器的当前耗时
func TimeLog(label string) {
	std.TimeLog(label)
}

func (c *Console) TimeLog(label string) {
	if c.disabled {
		return
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	if label == "" {
		label = "default"
	}

	startTime, exists := c.timers[label]
	if !exists {
		c.writef(c.errorOutput, colorRed, "%s: 计时器不存在", label)
		return
	}

	duration := time.Since(startTime)
	durationStr := duration.String()
	if duration.Milliseconds() < 1 {
		durationStr = fmt.Sprintf("%.3f μs", float64(duration.Nanoseconds())/1000)
	} else if duration.Milliseconds() < 1000 {
		durationStr = fmt.Sprintf("%.2f ms", float64(duration.Microseconds())/1000)
	}
	
	c.writef(c.output, colorGreen, "%s: %s", label, durationStr)
}

// TimeStamp 添加一个时间标记
func TimeStamp(label string) {
	std.TimeStamp(label)
}

func (c *Console) TimeStamp(label string) {
	if c.disabled {
		return
	}

	if label == "" {
		label = "default"
	}

	now := time.Now()
	timestamp := now.Format("2006-01-02T15:04:05.000Z07:00")
	
	c.writef(c.output, colorGreen, "%s: %s", label, timestamp)
}
