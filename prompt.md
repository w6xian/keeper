### 第一次
设计一个基于golang语言4的守护进程keeper，用于加参数后启动自己成为主进程如：keeper app

要求：1、基于websocket的RPC通信协议（直接用github.com/w6xian/sloth)

2、启动参数用github.com/spf13/cobra实现

请先写明需求后，确认后，再实施

### 第二次
完善守护keeper功能
##### 日志功能
* 日志功能，keeper app 中 通过 rpc 调用 keeper 日志接口，keeper 中实现日志接口，将日志打印到文件中(用zap日志库)
* 日志需按等级分类，如：info、debug、warn、error等
* 日志文件需按日期分类，如：2023-08-01.log
* 日志目录可设置总大小，如：100MB，超过后自动删除旧日志文件
##### 其他功能
* 其他功能，如：配置文件、环境变量等 （用viper库）
* 其他功能，如：信号量、锁等 （用sync库）
* 嵌入glua脚本引擎，用于执行自定义逻辑（用github.com/w6xian/glua）




