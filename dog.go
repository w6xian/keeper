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

	"github.com/w6xian/sloth"
	"github.com/w6xian/sloth/nrpc/wsocket"
	"go.uber.org/zap"
)

type Dog struct {
	logger     *zap.Logger
	addr       string
	wsPath     string
	clientRpc  *sloth.ServerRpc
	clientConn *sloth.Connect
	Name       string
	Watcher    IWatcher
}

func NewDog(addr, wsPath string, options ...DogOption) *Dog {
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

	// Client logic container (ServerRpc handles client-side logic for outgoing requests)
	d.clientRpc = sloth.DefaultClient()
	// Connection manager
	d.clientConn = sloth.ClientConn(d.clientRpc)
	if d.Watcher != nil {
		d.clientConn.Register("dog", d.Watcher, d.Name)
	}

	return d
}

func (d *Dog) KeepAlive() error {
	// Dial
	go d.clientConn.StartWebsocketClient(
		wsocket.WithClientUriPath(d.wsPath),
		wsocket.WithClientServerUri(d.addr),
	)
	time.Sleep(1 * time.Second)
	// --- Registry Logic ---
	instanceID := fmt.Sprintf("%s-%d", d.Name, os.Getpid())
	serviceName := fmt.Sprintf("%s-service", d.Name)
	// 1. Register
	fmt.Printf("[%s] Registering service...\n", d.Name)
	regReq := registry.RegisterRequest{
		Instance: registry.ServiceInstance{
			ID:   instanceID,
			Name: serviceName,
			Host: "127.0.0.1",
			Port: 0, // Fake port for now
			Tags: []string{"v1", "test"},
		},
	}
	regRespBytes, err := d.clientRpc.Call(context.Background(), "registry.Register", regReq)
	if err != nil {
		fmt.Printf("[App] Register failed: %v\n", err)
	} else {
		fmt.Printf("[App] Register success: %s\n", string(regRespBytes))
	}

	return nil
}

func (d *Dog) Stop() error {
	status, err := d.clientRpc.Call(context.Background(), "command.Exit", 200)
	if err != nil {
		fmt.Printf("[%s] Exit failed: %v\n", d.Name, err)
	} else {
		fmt.Printf("[%s] Exit success: %s\n", d.Name, string(status))
	}
	return nil
}
