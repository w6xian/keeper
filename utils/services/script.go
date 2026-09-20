package services

import (
	"context"

	"github.com/w6xian/sloth/v4"
)

var scriptClient rpcHolder

// InitScript 绑定 script 服务的 RPC 客户端。
func InitScript(cli *sloth.ServerRpc) *ScriptService {
	scriptClient.init(cli)
	return &ScriptService{}
}

type ScriptService struct{}

func Run(ctx context.Context, s string) (string, error) {
	cli, err := scriptClient.get()
	if err != nil {
		return "", err
	}
	resp, err := cli.Call(ctx, "script.Run", s)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func LoadFile(ctx context.Context, filename string) (string, error) {
	cli, err := scriptClient.get()
	if err != nil {
		return "", err
	}
	resp, err := cli.Call(ctx, "script.LoadFile", filename)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}
