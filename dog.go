package keeper

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/w6xian/keeper/config"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/keeper/utils/services"

	"github.com/w6xian/sloth/v2"
	"github.com/w6xian/sloth/v2/message"
	"github.com/w6xian/sloth/v2/option"
	"github.com/w6xian/sloth/v2/types"
	"go.uber.org/zap"
)

type Dog struct {
	ctx        context.Context
	logger     *zap.Logger
	addr       string
	wsPath     string
	clientRpc  *sloth.ServerRpc
	clientConn *sloth.Connect
	Name       string
	Watcher    IWatcher
}

func NewDog(ctx context.Context, addr, wsPath string, options ...DogOption) *Dog {

	loggerConfig := logger.Config{
		Level:      config.GlobalConfig.Log.Level,
		Filename:   config.GlobalConfig.Log.Filename,
		MaxSize:    config.GlobalConfig.Log.MaxSize,
		MaxBackups: config.GlobalConfig.Log.MaxBackups,
		MaxAge:     config.GlobalConfig.Log.MaxAge,
		Compress:   config.GlobalConfig.Log.Compress,
	}

	if err := logger.InitLogger(loggerConfig); err != nil {
		log.Fatalf("Failed to init logger: %v", err)
	}

	logger.GetLogger().Info("Dog started", zap.Int("pid", os.Getpid()))

	d := &Dog{
		logger:  logger.GetLogger(),
		addr:    addr,
		wsPath:  wsPath,
		Name:    "dog",
		Watcher: nil,
	}

	for _, opt := range options {
		opt(d)
	}
	d.ctx = ctx

	// Client logic container (ServerRpc handles client-side logic for outgoing requests)
	// Get service methods
	client := sloth.DefaultClient()
	d.clientRpc = client
	clientConn := sloth.ClientConn(client)
	d.clientConn = clientConn
	// Start WebSocket Client in a goroutine
	if d.Watcher != nil {
		d.clientConn.Register("dog", d.Watcher, d.Name)
	}
	d.clientRpc.Call(ctx, "command.KeepAlive", 200)

	return d
}

func (d *Dog) InitService() {
	services.InitCache(d.clientRpc)
	services.InitCommand(d.clientRpc)
	services.InitLog(d.clientRpc)
	services.InitRegistry(d.clientRpc)
	services.InitScript(d.clientRpc)
}

func (d *Dog) KeepAlive() error {
	wait := make(chan struct{})
	defer close(wait)
	handler := &Handler{server: d.clientRpc}
	handler.OnConnected(func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error {
		// Wait for connection to be established
		// --- Registry Logic ---
		instanceID := fmt.Sprintf("%s-%d", d.Name, os.Getpid())
		serviceName := fmt.Sprintf("%s-service", d.Name)
		regReq := registry.RegisterRequest{
			Instance: registry.ServiceInstance{
				ID:   instanceID,
				Name: serviceName,
				Host: "127.0.0.1",
				Port: 0, // Fake port for now
				Tags: []string{"v1", "test"},
			},
		}
		regRespBytes, err := d.clientRpc.Call(d.ctx, "registry.Register", regReq)
		if err != nil {
			fmt.Printf("[%s] Register failed: %v\n", d.Name, err)
		} else {
			fmt.Printf("[%s] Register success: %s\n", d.Name, string(regRespBytes))
		}
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-d.ctx.Done():
					return
				case <-ticker.C:
					services.Heartbeat(d.ctx, registry.HeartbeatRequest{
						ServiceName: serviceName,
						InstanceID:  instanceID,
					})
				}
			}
		}()
		wait <- struct{}{}
		return nil
	})
	// Dial
	go func() {
		d.clientConn.Dial(d.ctx, "ws", d.addr,
			option.WithAddress(d.addr),
			option.WithUriPath(d.wsPath),
			option.WithClientHandleMessage(handler),
		)
	}()
	<-wait
	return nil
}

func (d *Dog) Stop() error {
	status, err := d.clientRpc.Call(d.ctx, "command.Exit", 200)
	if err != nil {
		fmt.Printf("[%s] Exit failed: %v\n", d.Name, err)
	} else {
		fmt.Printf("[%s] Exit success: %s\n", d.Name, string(status))
	}
	return nil
}

func (d *Dog) Call(ctx context.Context, mtd string, args ...any) (interface{}, error) {
	return d.clientRpc.Call(ctx, mtd, args...)
}

// CallWithHeader calls a service method with a custom header.
func (d *Dog) CallWithHeader(ctx context.Context, header message.Header, method string, args ...any) (interface{}, error) {
	return d.clientRpc.CallWithHeader(ctx, header, method, args...)
}
