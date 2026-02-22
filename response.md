toolName: open_folder
            
status: failed
          
请先打开文件夹以保存项目中的文件
          
你好！我是 Trae。我明白你的需求：你需要一个基于 Golang 的守护进程 `keeper`，使用 `cobra` 处理命令行参数，并使用 `sloth` 库通过 WebSocket 实现 RPC 通信。

按照你的要求，我先梳理详细的需求文档和设计方案。请你审核，确认无误后我再开始编写代码。

### 📌 项目需求与设计方案：Keeper 守护进程

#### 1. 项目概述
`keeper` 是一个守护进程管理工具。它通过命令行启动，并在后台作为主进程运行（或前台运行等待指令），对外提供基于 WebSocket 的 RPC 接口，允许外部通过 RPC 控制或查询该进程的状态。

#### 2. 技术栈
*   **编程语言**: Golang (Go 1.18+)
*   **命令行框架**: `github.com/spf13/cobra` (用于解析启动参数和子命令)
*   **RPC 通信**: `github.com/w6xian/sloth` (基于 WebSocket 的 RPC 协议)

#### 3. 功能模块设计

##### 3.1. 命令行入口 (CLI Entry)
*   **主命令**: `keeper`
*   **子命令**: `app`
    *   **功能**: 启动守护进程的核心逻辑。
    *   **参数**: 支持配置参数（如监听端口、配置文件路径等，初期可先硬编码或简单的 flag）。
    *   **示例**: `keeper app --port 8080`

##### 3.2. RPC 服务端 (RPC Server)
*   在 `keeper app` 启动后，初始化 `sloth` Server。
*   **通信方式**: WebSocket。
*   **监听**: 在指定端口监听 WebSocket 连接。
*   **功能**: 注册 RPC 服务，等待客户端（或其他进程）调用。
    *   *示例服务*: `Ping` (返回 Pong)，`Status` (返回当前进程状态)。

##### 3.3. 守护进程逻辑 (Daemon Core)
*   实现一个简单的生命周期管理。
*   支持优雅停止 (Graceful Shutdown)，监听系统信号 (SIGINT, SIGTERM)。

#### 4. 拟定项目目录结构

```text
keeper/
├── cmd/
│   └── keeper/
│       └── main.go        # 程序入口
├── internal/
│   ├── command/           # Cobra 命令定义 (root, app)
│   ├── server/            # RPC 服务端逻辑 (封装 sloth)
│   └── service/           # 具体的业务逻辑实现
├── go.mod
└── go.sum
```

#### 5. 交互流程
1.  用户在终端执行 `keeper app`。
2.  Cobra 解析命令，进入 `Run` 函数。
3.  程序启动 WebSocket RPC Server (Sloth)。
4.  程序阻塞运行，打印日志表明服务已启动。
5.  (可选扩展) 用户可以使用测试客户端连接该 WebSocket 进行 RPC 调用。

---

**请确认以上设计方案是否符合你的预期？**
如果有任何修改意见（例如需要具体的 RPC 方法定义，或者特殊的启动参数），请告诉我。确认后，我将立即为你创建项目并编写代码。