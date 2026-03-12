package console

import (
	"testing"
	"time"
)

func TestConsole(t *testing.T) {
	// 基础日志方法
	Log("这是一条普通日志")
	Info("这是一条信息")
	Warn("这是一条警告")
	Error("这是一条错误")
	Debug("这是一条调试信息")

	// 计数功能
	Count("点击次数")
	Count("点击次数")
	Count("点击次数")
	CountReset("点击次数")

	// 计时功能
	Time("操作1")
	time.Sleep(100 * time.Millisecond)
	TimeLog("操作1")
	time.Sleep(50 * time.Millisecond)
	TimeEnd("操作1")

	// 时间戳
	TimeStamp("重要事件1")
	TimeStamp("重要事件2")

	// 分组功能
	Group("用户数据")
	Info("用户ID: 123")
	Info("用户名: 张三")
	GroupEnd()

	GroupCollapsed("系统配置")
	Info("配置项1: 值1")
	Info("配置项2: 值2")
	GroupEnd()

	// 对象显示
	type User struct {
		ID   int
		Name string
		Age  int
	}

	user := User{ID: 1, Name: "李四", Age: 30}
	Dir(user)
	DirXML(user)

	// 表格显示
	users := []User{
		{ID: 1, Name: "张三", Age: 25},
		{ID: 2, Name: "李四", Age: 30},
		{ID: 3, Name: "王五", Age: 28},
	}
	Table(users)

	// 简单数组
	numbers := []int{1, 2, 3, 4, 5}
	Table(numbers)

	// Map 类型
	userMap := map[string]interface{}{
		"name": "赵六",
		"age":  35,
		"city": "北京",
	}
	Table(userMap)

	// 性能分析
	Profile("数据处理")
	time.Sleep(200 * time.Millisecond)
	ProfileEnd("数据处理")

	// 断言
	Assert(true, "这个断言会通过")
	Assert(false, "这个断言会失败")

	// 格式化断言
	Assertf(1+1 == 2, "%d + %d == %d", 1, 1, 2)

	// 堆栈跟踪
	Trace("当前位置的堆栈:")

	// 上下文
	Context("用户上下文", user)

	// 任务追踪
	task := CreateTask("数据导入")
	task.Run(func() {
		time.Sleep(100 * time.Millisecond)
		Info("导入数据中...")
		time.Sleep(100 * time.Millisecond)
	})
	task.Success("所有数据已成功导入")

	// 内存使用
	Memory()
	Log(MemoryJSON())

	// 清空控制台
	// Clear()
}
