package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/w6xian/sloth"
)

var (
	scriptOnce sync.Once
	script     *ScriptService
)

func InitScript(cli *sloth.ServerRpc) *ScriptService {
	scriptOnce.Do(func() {
		script = &ScriptService{cli: cli}
	})
	return script
}

type ScriptService struct {
	cli *sloth.ServerRpc
}

func Run(ctx context.Context, s string) (string, error) {
	newScript := InitScript(nil)
	if newScript.cli == nil {
		return "", fmt.Errorf("script client is nil")
	}
	resp, err := newScript.cli.Call(ctx, "script.Run", s)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}

func LoadFile(ctx context.Context, filename string) (string, error) {
	newScript := InitScript(nil)
	if newScript.cli == nil {
		return "", fmt.Errorf("script client is nil")
	}
	resp, err := newScript.cli.Call(ctx, "script.LoadFile", filename)
	if err != nil {
		return "", err
	}
	return string(resp), nil
}
