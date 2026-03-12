package main

import (
	"time"

	"github.com/w6xian/keeper/utils/console"
)

func main() {
	console.Log("=== Console 包演示 ===\n")

	// 基础日志
	console.Log("1. 基础日志测试:")
	console.Info("   - 信息日志")
	console.Warn("   - 警告日志")
	console.Error("   - 错误日志")
	console.Debug("   - 调试日志")
	console.Log("")

	// 计数器
	console.Log("2. 计数器测试:")
	console.Count("操作计数")
	console.Count("操作计数")
	console.Count("操作计数")
	console.CountReset("操作计数")
	console.Log("")

	// 计时器
	console.Log("3. 计时器测试:")
	console.Time("测试操作")
	time.Sleep(100 * time.Millisecond)
	console.TimeEnd("测试操作")
	console.Log("")

	// 内存使用
	console.Log("4. 内存使用:")
	console.Memory()
	console.Log("")

	// 断言
	console.Log("5. 断言测试:")
	console.Assert(true, "这个断言会通过")
	console.Assert(false, "这个断言会失败")
	console.Log("")

	// 表格显示
	console.Log("6. 表格显示:")
	type User struct {
		ID   int
		Name string
		Age  int
	}
	users := []User{
		{ID: 1, Name: "张三", Age: 25},
		{ID: 2, Name: "李四", Age: 30},
		{ID: 3, Name: "王五", Age: 28},
	}
	console.Table(users)
	console.Log("")

	console.Log("=== 演示完成 ===")
}
