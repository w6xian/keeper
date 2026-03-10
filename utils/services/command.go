package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/w6xian/sloth"
)

var (
	commandOnce sync.Once
	command     *Command
)

func InitCommand(cli *sloth.ServerRpc) *Command {
	commandOnce.Do(func() {
		command = &Command{cli: cli}
	})
	return command
}

type Command struct {
	cli *sloth.ServerRpc
}

// Exit sends exit signal to keeper
func Exit(ctx context.Context, code int) ([]byte, error) {
	newCommand := InitCommand(nil)
	if newCommand.cli == nil {
		return nil, fmt.Errorf("command client is nil")
	}
	return newCommand.cli.Call(ctx, "command.Exit", code)
}

// KeepAlive sends keepalive signal
func KeepAlive(ctx context.Context, code int) ([]byte, error) {
	newCommand := InitCommand(nil)
	if newCommand.cli == nil {
		return nil, fmt.Errorf("command client is nil")
	}
	return newCommand.cli.Call(ctx, "command.KeepAlive", code)
}
