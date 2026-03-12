package console

import (
	"testing"
)

func TestBasicLog(t *testing.T) {
	Log("测试 Log 方法")
	Info("测试 Info 方法")
	Warn("测试 Warn 方法")
	Error("测试 Error 方法")
	Debug("测试 Debug 方法")
}

func TestCount(t *testing.T) {
	Count("test")
	Count("test")
	CountReset("test")
}

func TestAssert(t *testing.T) {
	Assert(true, "应该通过")
	// Assert(false, "会失败") // 这个会输出错误但不会导致测试失败
}

func TestMemory(t *testing.T) {
	Memory()
	json := MemoryJSON()
	if json == "" {
		t.Error("MemoryJSON 返回空字符串")
	}
}
