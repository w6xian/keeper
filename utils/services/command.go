package services

import (
	"context"

	"github.com/w6xian/sloth/v4"
)

var commandClient rpcHolder

// InitCommand 绑定 command 服务的 RPC 客户端。
//
// 可重复调用：重连后传入新的 cli 即完成回绑；传 nil 表示"不改，只取当前"。
func InitCommand(cli *sloth.ServerRpc) *Command {
	commandClient.init(cli)
	return &Command{}
}

type Command struct{}

// Exit sends exit signal to keeper
func Exit(ctx context.Context, code int) ([]byte, error) {
	cli, err := commandClient.get()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, "command.Exit", code)
}

// KeepAlive sends keepalive signal
func KeepAlive(ctx context.Context, code int) ([]byte, error) {
	cli, err := commandClient.get()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, "command.KeepAlive", code)
}

func StopService(ctx context.Context, name string) ([]byte, error) {
	cli, err := commandClient.get()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, "command.StopService", name)
}

func StartService(ctx context.Context, name string) ([]byte, error) {
	cli, err := commandClient.get()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, "command.StartService", name)
}

func ReloadService(ctx context.Context, name string) ([]byte, error) {
	cli, err := commandClient.get()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, "command.ReloadService", name)
}
