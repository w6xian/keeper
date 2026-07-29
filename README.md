# Keeper

Keeper 是一个使用 Go 编写的轻量级守护/进程管理框架：通过 WebSocket RPC 将“父进程（Door）”与“子进程（Dog/业务进程）”解耦，实现进程拉起、远程控制、日志、注册发现与脚本执行等能力，并支持注册为系统服务（开机自启）。

## 特性

- Door/Dog 双进程模型：Door 负责监听与 RPC 服务注册，Dog 负责连接与业务执行
- WebSocket RPC：基于 `github.com/w6xian/sloth`
- 日志：`zap` + `lumberjack` 文件滚动，RPC 暴露 `log.*` 接口
- 注册中心（内存版）：Register/Deregister/Heartbeat/Discovery，支持 TTL 过期剔除
- Dog 连接治理：内置自动重连、首次注册重试、连接断开后自动停止心跳，Stop 时优雅等待注销完成
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

- 启动 Door（随机端口监听 RPC）
- 拉起 `app` 子命令作为子进程（Dog），并通过 WebSocket 连接 Door
- Dog 首次连上后会向注册中心发起 `registry.Register`，失败时自动重试
- 连接建立后按固定周期发送 `Heartbeat`，连接关闭或停止时会自动取消心跳并执行 `Deregister`

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

## 代码入口

- 示例程序入口：`example/main.go`
- CLI 与默认运行逻辑：`example/cmd/root.go`
- 服务安装/卸载：`example/cmd/install.go`
- Door（服务端/父进程）：`door.go`
- Dog（客户端/子进程）：`dog.go`
- RPC 服务实现：`service/`（command/log/registry/script）
