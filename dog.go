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

	"github.com/w6xian/sloth/v2"
	"github.com/w6xian/sloth/v2/message"
	"github.com/w6xian/sloth/v2/option"
	"github.com/w6xian/sloth/v2/types"
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
	var readyOnce sync.Once
	handler := &Handler{server: d.clientRpc}
	handler.OnConnected(func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error {
		d.mu.Lock()
		instanceID := d.instanceID
		serviceName := d.serviceName
		d.connected = true
		if d.heartbeatCancel != nil {
			d.heartbeatCancel()
		}
		heartbeatCtx, heartbeatCancel := context.WithCancel(d.ctx)
		d.heartbeatCancel = heartbeatCancel
		d.mu.Unlock()

		regReq := registry.RegisterRequest{
			Instance: registry.ServiceInstance{
				ID:   instanceID,
				Name: serviceName,
				Host: "127.0.0.1",
				Port: 0, // Fake port for now
				Tags: []string{"v1", "test"},
			},
		}

		var regRespBytes []byte
		var err error
		for attempt := 1; attempt <= 5; attempt++ {
			regRespBytes, err = d.clientRpc.Call(ctx, "registry.Register", regReq)
			if err == nil {
				break
			}
			d.logger.Warn("registry register failed, will retry",
				zap.String("dog", d.Name),
				zap.String("service", serviceName),
				zap.String("instanceID", instanceID),
				zap.Int("attempt", attempt),
				zap.Error(err),
			)
			select {
			case <-ctx.Done():
				readyOnce.Do(func() { ready <- ctx.Err() })
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}
		if err != nil {
			readyOnce.Do(func() { ready <- fmt.Errorf("register failed after retries: %w", err) })
			return err
		}

		d.mu.Lock()
		if heartbeatCtx.Err() != nil {
			d.mu.Unlock()
			readyOnce.Do(func() { ready <- context.Canceled })
			return context.Canceled
		}
		d.mu.Unlock()
		d.logger.Info("registry register success",
			zap.String("dog", d.Name),
			zap.String("service", serviceName),
			zap.String("instanceID", instanceID),
			zap.ByteString("response", regRespBytes),
		)

		d.heartbeatWG.Add(1)
		go func() {
			defer d.heartbeatWG.Done()
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-heartbeatCtx.Done():
					return
				case <-ticker.C:
					if err := services.Heartbeat(heartbeatCtx, registry.HeartbeatRequest{
						ServiceName: serviceName,
						InstanceID:  instanceID,
					}); err != nil {
						d.logger.Warn("registry heartbeat failed",
							zap.String("dog", d.Name),
							zap.String("service", serviceName),
							zap.String("instanceID", instanceID),
							zap.Error(err),
						)
					}
				}
			}
		}()
		readyOnce.Do(func() { ready <- nil })
		return nil
	})
	handler.OnClosed(func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error {
		d.stopHeartbeat()
		d.mu.Lock()
		d.connected = false
		d.mu.Unlock()
		d.logger.Warn("dog connection closed",
			zap.String("dog", d.Name),
			zap.String("service", d.serviceName),
			zap.String("instanceID", d.instanceID),
		)
		return nil
	})
	handler.OnErrored(func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo, err error) error {
		d.stopHeartbeat()
		d.mu.Lock()
		d.connected = false
		d.mu.Unlock()
		d.logger.Warn("dog connection error",
			zap.String("dog", d.Name),
			zap.String("service", d.serviceName),
			zap.String("instanceID", d.instanceID),
			zap.Error(err),
		)
		return nil
	})
	go func() {
		defer func() {
			if r := recover(); r != nil {
				readyOnce.Do(func() { ready <- fmt.Errorf("dial panic: %v", r) })
			}
		}()
		d.clientConn.Dial(d.ctx, "ws", d.addr,
			option.WithAddress(d.addr),
			option.WithUriPath(d.wsPath),
			option.WithClientHandleMessage(handler),
		)
	}()
	select {
	case err := <-ready:
		return err
	case <-d.ctx.Done():
		return d.ctx.Err()
	case <-time.After(10 * time.Second):
		return fmt.Errorf("wait for connection timeout")
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
		fmt.Printf("[%s] Exit failed: %v\n", d.Name, err)
	} else {
		fmt.Printf("[%s] Exit success: %s\n", d.Name, string(status))
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
