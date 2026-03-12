package console

// Log 输出一般日志信息
func Log(args ...interface{}) {
	std.Log(args...)
}

func (c *Console) Log(args ...interface{}) {
	if c.disabled {
		return
	}
	c.writeSafe(c.output, colorReset, args...)
}

// Info 输出信息性日志
func Info(args ...interface{}) {
	std.Info(args...)
}

func (c *Console) Info(args ...interface{}) {
	if c.disabled {
		return
	}
	c.writeSafe(c.output, colorBlue, append([]interface{}{"ℹ️"}, args...)...)
}

// Warn 输出警告信息
func Warn(args ...interface{}) {
	std.Warn(args...)
}

func (c *Console) Warn(args ...interface{}) {
	if c.disabled {
		return
	}
	c.writeSafe(c.output, colorYellow, append([]interface{}{"⚠️"}, args...)...)
}

// Error 输出错误信息
func Error(args ...interface{}) {
	std.Error(args...)
}

func (c *Console) Error(args ...interface{}) {
	if c.disabled {
		return
	}
	c.writeSafe(c.errorOutput, colorRed, append([]interface{}{"❌"}, args...)...)
}

// Debug 输出调试信息
func Debug(args ...interface{}) {
	std.Debug(args...)
}

func (c *Console) Debug(args ...interface{}) {
	if c.disabled {
		return
	}
	c.writeSafe(c.output, colorGray, append([]interface{}{"🔍"}, args...)...)
}

// Logf 输出格式化日志信息
func Logf(format string, args ...interface{}) {
	std.Logf(format, args...)
}

func (c *Console) Logf(format string, args ...interface{}) {
	if c.disabled {
		return
	}
	c.writefSafe(c.output, colorReset, format, args...)
}

// Infof 输出格式化信息
func Infof(format string, args ...interface{}) {
	std.Infof(format, args...)
}

func (c *Console) Infof(format string, args ...interface{}) {
	if c.disabled {
		return
	}
	c.writefSafe(c.output, colorBlue, "ℹ️ "+format, args...)
}

// Warnf 输出格式化警告
func Warnf(format string, args ...interface{}) {
	std.Warnf(format, args...)
}

func (c *Console) Warnf(format string, args ...interface{}) {
	if c.disabled {
		return
	}
	c.writefSafe(c.output, colorYellow, "⚠️ "+format, args...)
}

// Errorf 输出格式化错误
func Errorf(format string, args ...interface{}) {
	std.Errorf(format, args...)
}

func (c *Console) Errorf(format string, args ...interface{}) {
	if c.disabled {
		return
	}
	c.writefSafe(c.errorOutput, colorRed, "❌ "+format, args...)
}

// Debugf 输出格式化调试信息
func Debugf(format string, args ...interface{}) {
	std.Debugf(format, args...)
}

func (c *Console) Debugf(format string, args ...interface{}) {
	if c.disabled {
		return
	}
	c.writefSafe(c.output, colorGray, "🔍 "+format, args...)
}
