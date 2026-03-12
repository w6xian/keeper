package console

// Package console 提供了与 JavaScript console 库类似的控制台输出功能。
//
// 已实现的全部方法：
//   - 基础日志：Log, Info, Warn, Error, Debug
//   - 计数功能：Count, CountReset
//   - 时间相关：Time, TimeEnd, TimeLog, TimeStamp
//   - 分组功能：Group, GroupCollapsed, GroupEnd
//   - 对象显示：Dir, DirXML, Table
//   - 性能分析：Profile, ProfileEnd
//   - 断言跟踪：Assert, Trace
//   - 特殊功能：Clear, Context, CreateTask, Memory
//
// 使用示例：
//
//	console.Log("这是一条普通日志")
//	console.Info("这是一条信息")
//	console.Count("点击次数")
//	console.Time("操作1")
//	// ... 执行操作 ...
//	console.TimeEnd("操作1")
//
// 更多用法请参考 README.md
