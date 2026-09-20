# Keeper

Keeper 是一个使用 Go 编写的轻量级守护/进程管理框架：通过 **TCP RPC** 将“父进程（Door）”与“子进程（Dog/业务进程）”解耦，实现进程拉起、远程控制、日志、注册发现与脚本执行等能力，并支持注册为系统服务（开机自启）。

## 特性

- Door/Dog 双进程模型：Door 负责监听与 RPC 服务注册，Dog 负责连接与业务执行
- **TCP RPC**：基于 `github.com/w6xian/sloth` **v4**，Door 与 Dog 之间走 TCP 传输（`sloth.TCP`）
- 日志：`zap` + `lumberjack` 文件滚动，RPC 暴露 `log.*` 接口
- 注册中心（内存版）：Register/Deregister/Heartbeat/Discovery，支持 TTL 过期剔除（`Close()` 可停止剔除协程）
- Dog 连接治理：断线自动重连（指数退避）+ 重连后自动回绑 RPC 客户端、首次注册重试、按周期心跳、Stop 时优雅注销
- 脚本执行：通过 RPC 提供 Lua 脚本执行（`script.Run`/`script.LoadFile`），基于 `github.com/w6xian/gua`
- 系统服务安装/卸载：
  - Windows：`sc create/start/stop/delete`
  - Linux：systemd unit + `systemctl enable --now`

## 快速开始（示例程序）

### 运行

```bash
go run ./example
```

示例程序默认行为：

- 启动 Door（默认 `127.0.0.1:8965`，可用 `--port` 覆盖，TCP 监听）
- 拉起 `app` 子命令作为子进程（Dog），并通过 TCP 连接 Door
- Dog 首次连上后会向注册中心发起 `registry.Register`，失败时自动重试
- 连接建立后按固定周期发送 `Heartbeat`（默认 5s）；连接断开则停止心跳并触发重连
- `Stop()`/`Close()` 会停止心跳、注销实例，并等待相关 goroutine 退出

### 构建

```bash
go build -o keeper ./example
```

Windows：

```bash
go build -o keeper.exe ./example
```

如果仓库包含 `vendor/` 目录，Go 会默认使用 vendor 依赖；若依赖缺失，可执行：

```bash
go mod tidy
go mod vendor
```

## Dog 选项

```go
dog := keeper.NewDog(ctx, "127.0.0.1:8965", "/ws",
    keeper.WithDogName("app"),
    keeper.WithDogAutoReconnect(true),                        // 默认开启；一次性命令行建议关闭
    keeper.WithDogDialTimeout(3*time.Second),                 // 等待连接就绪的超时
    keeper.WithDogHeartbeatInterval(5*time.Second),           // 心跳周期，应小于注册中心 TTL
    keeper.WithDogReconnectBackoff(500*time.Millisecond, 15*time.Second), // 重连退避
)
```

## 注册为系统服务（开机自启）

示例程序内置了 `install/uninstall` 子命令用于安装/卸载系统服务。

```bash
./keeper install
./keeper uninstall
```

说明：

- Windows 安装服务通常需要管理员权限
- Windows 下服务名在示例中由 `example/cmd/install.go` 的 `server_name` 变量决定
- Windows 下双击 `keeper.exe` 时，若服务已运行则直接退出；若服务未运行则优先尝试启动服务

## 配置

配置使用 `viper`，默认读取当前工作目录的 `config.yaml`（文件不存在时使用默认值），并支持环境变量覆盖（例如 `LOG_LEVEL` 对应 `log.level`）。

示例 `config.yaml`：

```yaml
log:
  level: info
  filename: ./logs/keeper.log
  max_size: 100
  max_backups: 3
  max_age: 28
  compress: true
```

服务编排配置（`conf.toml`）中的 `[[Service]]` 段支持 `name/after/start/reload/stop` 以及重启治理相关字段（`restart_limit`、`restart_delay`、`restart_backoff_max`、`restart_backoff_step`、`stop_timeout`）；`after` 支持 `a,b` 或 `a;b` 多依赖，按拓扑顺序启动。

## 已知限制

- 注册中心是**内存版**，进程重启后实例信息丢失，也不做多节点同步
- `utils/services` 的 cache bucket 是进程级全局的，并发使用多个 bucket 需要调用方自己串行化
- Dog 的重连由“连接断开事件”触发，不依赖 TCP 层心跳；若网络处于半开状态（对端不可达但本地未感知），恢复依赖下一次写失败

## 代码入口

- 示例程序入口：`example/main.go`
- CLI 与默认运行逻辑：`example/cmd/root.go`
- 服务安装/卸载：`example/cmd/install.go`
- Door（服务端/父进程）：`door.go`
- Dog（客户端/子进程）：`dog.go`
- 连接事件钩子：`c_handler.go`
- 服务编排与重启治理：`runner.go`
- RPC 服务实现：`service/`（command/log/registry/script/cache）
