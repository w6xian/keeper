# Console Package

基于 JavaScript console 库方法实现的 Go 语言控制台输出库。

## 功能特性

实现了所有 JavaScript console 库的方法：

### 基础日志方法
- `Log` - 输出一般日志信息
- `Info` - 输出信息性日志
- `Warn` - 输出警告信息
- `Error` - 输出错误信息
- `Debug` - 输出调试信息

### 计数功能
- `Count` - 记录并输出标签被调用的次数
- `CountReset` - 重置指定标签的计数器

### 时间相关
- `Time` - 启动一个计时器
- `TimeEnd` - 结束计时器并输出耗时
- `TimeLog` - 输出计时器的当前耗时
- `TimeStamp` - 添加一个时间标记

### 分组功能
- `Group` - 创建一个内联分组
- `GroupCollapsed` - 创建一个折叠的分组
- `GroupEnd` - 结束当前分组

### 对象显示
- `Dir` - 显示对象的交互式列表
- `DirXML` - 显示对象的 XML/HTML 表示
- `Table` - 以表格形式显示数组或对象

### 性能分析
- `Profile` - 启动 CPU 性能分析器
- `ProfileEnd` - 结束 CPU 性能分析并输出结果

### 断言和跟踪
- `Assert` - 断言，如果为 false 则输出错误信息和堆栈跟踪
- `Trace` - 输出堆栈跟踪

### 特殊功能
- `Clear` - 清空控制台
- `Context` - 保存当前的上下文
- `CreateTask` - 创建一个可追踪的任务
- `Memory` - 输出内存使用情况

## 安装

```bash
go get github.com/yourusername/IoT-printer/internal/console
```

## 使用示例

### 基础日志

```go
package main

import (
    "github.com/yourusername/IoT-printer/internal/console"
)

func main() {
    console.Log("这是一条普通日志")
    console.Info("这是一条信息")
    console.Warn("这是一条警告")
    console.Error("这是一条错误")
    console.Debug("这是一条调试信息")
}
```

### 计数功能

```go
console.Count("点击次数")  // 输出: 点击次数: 1
console.Count("点击次数")  // 输出: 点击次数: 2
console.Count("点击次数")  // 输出: 点击次数: 3
console.CountReset("点击次数")  // 输出: 点击次数: 计数器已重置
```

### 计时功能

```go
console.Time("操作1")
// ... 执行一些操作 ...
console.TimeLog("操作1")  // 输出当前耗时
// ... 继续执行 ...
console.TimeEnd("操作1")  // 输出总耗时
```

### 分组功能

```go
console.Group("用户数据")
console.Info("用户ID: 123")
console.Info("用户名: 张三")
console.GroupEnd()
```

### 对象显示

```go
type User struct {
    ID   int
    Name string
    Age  int
}

user := User{ID: 1, Name: "李四", Age: 30}
console.Dir(user)      // 显示对象详细信息
console.DirXML(user)   // 显示 XML/HTML 格式
```

### 表格显示

```go
users := []User{
    {ID: 1, Name: "张三", Age: 25},
    {ID: 2, Name: "李四", Age: 30},
    {ID: 3, Name: "王五", Age: 28},
}
console.Table(users)  // 以表格形式显示
```

### 性能分析

```go
console.Profile("数据处理")
// ... 执行数据处理 ...
console.ProfileEnd("数据处理")  // 输出性能分析结果
```

### 断言

```go
console.Assert(true, "这个断言会通过")
console.Assert(false, "这个断言会失败")
console.Assertf(1+1 == 2, "%d + %d == %d", 1, 1, 2)
```

### 堆栈跟踪

```go
console.Trace("当前位置的堆栈:")
```

### 任务追踪

```go
task := console.CreateTask("数据导入")
task.Run(func() {
    // 执行任务...
})
task.Success("所有数据已成功导入")
```

### 内存使用

```go
console.Memory()  // 输出内存使用情况
json := console.MemoryJSON()  // 获取 JSON 格式的内存统计
console.Log(json)
```

## 高级用法

### 自定义配置

```go
// 创建自定义 Console 实例
c := console.New(
    console.WithOutput(os.Stdout),
    console.WithErrorOutput(os.Stderr),
    console.WithColors(true),
)

// 使用自定义实例
c.Log("使用自定义实例")
c.Error("自定义错误输出")
```

### 禁用/启用输出

```go
console.Disable()  // 禁用所有输出
console.Enable()   // 重新启用输出

if console.IsDisabled() {
    fmt.Println("控制台已禁用")
}
```

### 设置输出目标

```go
file, _ := os.Create("output.log")
console.SetOutput(file)
console.Log("这条日志将写入文件")
```

## 格式化方法

所有日志方法都有对应的格式化版本：

```go
console.Logf("用户 %s 的年龄是 %d", "张三", 25)
console.Infof("已处理 %d 个文件", 100)
console.Warnf("警告: %s", "磁盘空间不足")
console.Errorf("错误: %v", err)
console.Debugf("调试信息: %x", data)
```

## 注意事项

1. 颜色输出在某些终端中可能不支持，可以通过 `WithColors(false)` 禁用
2. `Profile` 是简化实现，实际的性能分析建议使用 Go 的 `pprof` 包
3. `DirXML` 在 Go 中简化为 JSON 格式输出
4. `GroupCollapsed` 在终端中行为与 `Group` 相同
5. `Clear` 在某些环境中可能不起作用

## 许可证

MIT License
