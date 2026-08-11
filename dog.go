package keeper

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/w6xian/keeper/config"
	"github.com/w6xian/keeper/logger"
	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/keeper/utils/services"

	"github.com/w6xian/sloth/v3"
	"github.com/w6xian/sloth/v3/message"
	"github.com/w6xian/sloth/v3/option"
	"go.uber.org/zap"
)

type Dog struct {
	ctx             context.Context
	logger          *zap.Logger
	addr            string
	wsPath          string
	clientRpc       *sloth.ServerRpc
	clientConn      *sloth.Connect
	Name            string
	Watcher         IWatcher
	mu              sync.Mutex
	instanceID      string
	serviceName     string
	connected       bool
	heartbeatCancel context.CancelFunc
	heartbeatWG     sync.WaitGroup
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
		logger:      logger.GetLogger(),
		addr:        addr,
		wsPath:      wsPath,
		Name:        "dog",
		Watcher:     nil,
		instanceID:  fmt.Sprintf("%s-%d", "dog", os.Getpid()),
		serviceName: "dog-service",
	}

	for _, opt := range options {
		opt(d)
	}
	d.ctx = ctx
	d.instanceID = fmt.Sprintf("%s-%d", d.Name, os.Getpid())
	d.serviceName = fmt.Sprintf("%s-service", d.Name)

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
	if d.ctx == nil {
		return fmt.Errorf("dog context is nil")
	}
	ready := make(chan error, 1)
	go func() {
		d.clientConn.Dial(d.ctx, "ws", d.addr,
			option.WithUriPath(d.wsPath),
			option.WithClientHandleMessage(&Handler{ready: ready, dog: d}),
		)
	}()
	timer := time.NewTimer(3 * time.Second)
	defer timer.Stop()
	select {
	case <-timer.C:
		return fmt.Errorf("dog dial timeout")
	case err := <-ready:
		return err
	}
}

func (d *Dog) Stop() error {
	d.mu.Lock()
	instanceID := d.instanceID
	serviceName := d.serviceName
	d.mu.Unlock()
	d.stopHeartbeat()
	d.heartbeatWG.Wait()

	if serviceName != "" && instanceID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := services.Deregister(ctx, registry.DeregisterRequest{
			ServiceName: serviceName,
			InstanceID:  instanceID,
		}); err != nil {
			d.logger.Warn("registry deregister failed",
				zap.String("dog", d.Name),
				zap.String("service", serviceName),
				zap.String("instanceID", instanceID),
				zap.Error(err),
			)
		} else {
			d.logger.Info("registry deregister success",
				zap.String("dog", d.Name),
				zap.String("service", serviceName),
				zap.String("instanceID", instanceID),
			)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	status, err := d.clientRpc.Call(ctx, "command.Exit", 200)
	if err != nil {
		log.Printf("[%s] Exit failed: %v\n", d.Name, err)
	} else {
		log.Printf("[%s] Exit success: %s\n", d.Name, string(status))
	}
	return nil
}

func (d *Dog) Close() error {
	d.mu.Lock()
	instanceID := d.instanceID
	serviceName := d.serviceName
	d.mu.Unlock()
	d.stopHeartbeat()
	d.heartbeatWG.Wait()

	if serviceName != "" && instanceID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := services.Deregister(ctx, registry.DeregisterRequest{
			ServiceName: serviceName,
			InstanceID:  instanceID,
		}); err != nil {
			d.logger.Warn("registry deregister failed",
				zap.String("dog", d.Name),
				zap.String("service", serviceName),
				zap.String("instanceID", instanceID),
				zap.Error(err),
			)
		} else {
			d.logger.Info("registry deregister success",
				zap.String("dog", d.Name),
				zap.String("service", serviceName),
				zap.String("instanceID", instanceID),
			)
		}
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

func (d *Dog) stopHeartbeat() {
	d.mu.Lock()
	cancel := d.heartbeatCancel
	d.heartbeatCancel = nil
	d.mu.Unlock()
	if cancel != nil {
		cancel()
	}
}
