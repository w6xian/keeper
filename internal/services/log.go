package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/w6xian/sloth"
)

var (
	logOnce sync.Once
	logSvc  *LogService
)

func InitLog(cli *sloth.ServerRpc) *LogService {
	logOnce.Do(func() {
		logSvc = &LogService{cli: cli}
	})
	return logSvc
}

type LogService struct {
	cli *sloth.ServerRpc
}

func Info(ctx context.Context, msg string) error {
	newLog := InitLog(nil)
	if newLog.cli == nil {
		return fmt.Errorf("log client is nil")
	}
	_, err := newLog.cli.Call(ctx, "log.Info", msg)
	return err
}

func Debug(ctx context.Context, msg string) error {
	newLog := InitLog(nil)
	if newLog.cli == nil {
		return fmt.Errorf("log client is nil")
	}
	_, err := newLog.cli.Call(ctx, "log.Debug", msg)
	return err
}

func Warn(ctx context.Context, msg string) error {
	newLog := InitLog(nil)
	if newLog.cli == nil {
		return fmt.Errorf("log client is nil")
	}
	_, err := newLog.cli.Call(ctx, "log.Warn", msg)
	return err
}

func Error(ctx context.Context, msg string) error {
	newLog := InitLog(nil)
	if newLog.cli == nil {
		return fmt.Errorf("log client is nil")
	}
	_, err := newLog.cli.Call(ctx, "log.Error", msg)
	return err
}

type LogRequest struct {
	Level   string                 `json:"level"`
	Message string                 `json:"message"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

func Log(ctx context.Context, req LogRequest) error {
	newLog := InitLog(nil)
	if newLog.cli == nil {
		return fmt.Errorf("log client is nil")
	}
	_, err := newLog.cli.Call(ctx, "log.Log", req)
	return err
}
