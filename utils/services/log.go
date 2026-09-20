package services

import (
	"context"

	"github.com/w6xian/sloth/v4"
)

var logClient rpcHolder

// InitLog 绑定 log 服务的 RPC 客户端。
func InitLog(cli *sloth.ServerRpc) *LogService {
	logClient.init(cli)
	return &LogService{}
}

type LogService struct{}

func Info(ctx context.Context, msg string) error {
	cli, err := logClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "log.Info", msg)
	return err
}

func Debug(ctx context.Context, msg string) error {
	cli, err := logClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "log.Debug", msg)
	return err
}

func Warn(ctx context.Context, msg string) error {
	cli, err := logClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "log.Warn", msg)
	return err
}

func Error(ctx context.Context, msg string) error {
	cli, err := logClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "log.Error", msg)
	return err
}

type LogRequest struct {
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

func Log(ctx context.Context, req LogRequest) error {
	cli, err := logClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "log.Log", req)
	return err
}
