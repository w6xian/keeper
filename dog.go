package keeper

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/keeper/utils/services"

	"github.com/w6xian/sloth/v4"
	"github.com/w6xian/sloth/v4/message"
	"github.com/w6xian/sloth/v4/option"
)

const (
	defaultDialTimeout      = 3 * time.Second
	defaultCallTimeout      = 3 * time.Second
	defaultHeartbeat        = 5 * time.Second
	defaultReconnectBase    = 500 * time.Millisecond
	defaultReconnectMax     = 15 * time.Second
	defaultRegisterAttempts = 3
	defaultRegisterInterval = 500 * time.Millisecond
)

// ErrDogClosed Dog 已停止（Stop/Close 之后不能再发起调用）。
var ErrDogClosed = errors.New("keeper: dog is closed")

type Dog struct {
	ctx    context.Context
	addr   string
	wsPath string // 仅 WebSocket 传输使用；TCP 传输下无意义（保留字段以免破坏调用方）

	mu         sync.RWMutex
	clientRpc  *sloth.ServerRpc
	clientConn *sloth.Connect

	Name        string
	Watcher     IWatcher
	instanceID  string
	serviceName string
	connected   bool

	dialTimeout       time.Duration
	callTimeout       time.Duration
	heartbeatInterval time.Duration
	autoReconnect     bool
	reconnectBase     time.Duration
	reconnectMax      time.Duration
	registerAttempts  int
	registerInterval  time.Duration

	heartbeatCancel context.CancelFunc
	heartbeatWG     sync.WaitGroup

	disconnectCh chan struct{}
	reconnectWG  sync.WaitGroup
	stopCh       chan struct{}
	stopOnce     sync.Once
	closed       atomic.Bool
}

func NewDog(ctx context.Context, addr, wsPath string, options ...DogOption) *Dog {
	log.Printf("Dog started %d", os.Getpid())

	d := &Dog{
		addr:              addr,
		wsPath:            wsPath,
		Name:              "dog",
		instanceID:        fmt.Sprintf("%s-%d", "dog", os.Getpid()),
		serviceName:       "dog-service",
		dialTimeout:       defaultDialTimeout,
		callTimeout:       defaultCallTimeout,
		heartbeatInterval: defaultHeartbeat,
		autoReconnect:     true,
		reconnectBase:     defaultReconnectBase,
		reconnectMax:      defaultReconnectMax,
		registerAttempts:  defaultRegisterAttempts,
		registerInterval:  defaultRegisterInterval,
		disconnectCh:      make(chan struct{}, 1),
		stopCh:            make(chan struct{}),
	}

	for _, opt := range options {
		opt(d)
	}

	if ctx == nil {
		// 旧的 NewDog 把 nil ctx 一路传到底层，KeepAlive 才报错；
		// 这里兜住，保证构造出来的 Dog 一定可用。
		ctx = context.Background()
	}
	d.ctx = ctx
	d.instanceID = fmt.Sprintf("%s-%d", d.Name, os.Getpid())
	d.serviceName = fmt.Sprintf("%s-service", d.Name)

	// 建立客户端侧的 RPC 容器（ServerRpc 是"打给服务端"的调用端）。
	d.newClient()

	return d
}

// newClient 建立一套全新的 RPC 容器与连接对象。
//
// 重连必须换掉整套对象：sloth 的 Dial 会拒绝"已经 dial 过"的 Connect
// （ServerRpc.Listen 非空即报错），旧对象无法二次拨号。
func (d *Dog) newClient() *sloth.ServerRpc {
	client := sloth.DefaultClient()
	conn := sloth.ClientConn(client)
	if d.Watcher != nil {
		if err := conn.Register("dog", d.Watcher, d.Name); err != nil {
			log.Printf("[%s] register watcher failed: %v\n", d.Name, err)
		}
	}
	d.mu.Lock()
	d.clientRpc = client
	d.clientConn = conn
	d.mu.Unlock()
	return client
}

// InitService 把当前 RPC 客户端绑定到 services 包（command/log/registry/script/cache）。
//
// 重连后由 Dog 内部自动回绑，调用方无需重复调用。
func (d *Dog) InitService() {
	cli := d.rpcOrNil()
	services.InitCache(cli)
	services.InitCommand(cli)
	services.InitLog(cli)
	services.InitRegistry(cli)
	services.InitScript(cli)
}

// KeepAlive 连接 Door 并等待连接就绪。
//
// 返回 nil 表示已连上；此后 Dog 会自行注册、发心跳，并在断线后按退避重连
// （可用 WithDogAutoReconnect(false) 关闭）。
func (d *Dog) KeepAlive() error {
	if d.closed.Load() {
		return ErrDogClosed
	}
	ready := make(chan error, 1)
	if err := d.dial(d.ctx, ready); err != nil {
		return fmt.Errorf("dog dial %s: %w", d.addr, err)
	}
	if err := d.waitReady(d.ctx, ready); err != nil {
		return err
	}
	if d.autoReconnect {
		d.reconnectWG.Add(1)
		go d.reconnectLoop()
	}
	return nil
}

// dial 发起一次拨号。ready 由 Handler.OnReady 通知。
func (d *Dog) dial(ctx context.Context, ready chan error) error {
	d.mu.RLock()
	conn := d.clientConn
	d.mu.RUnlock()
	if conn == nil {
		return errors.New("rpc client not initialized")
	}
	return conn.Dial(ctx, sloth.TCP, d.addr,
		option.WithTcpClientHandleMessage(&Handler{ready: ready, dog: d}),
	)
}

// waitReady 等 OnReady 回调；超时按失败处理（底层连接可能仍在建立，
// 调用方拿到错误后应当退出或重试）。
func (d *Dog) waitReady(ctx context.Context, ready chan error) error {
	timer := time.NewTimer(d.dialTimeout)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return fmt.Errorf("dog dial %s canceled: %w", d.addr, ctx.Err())
	case <-timer.C:
		return fmt.Errorf("dog dial %s timeout after %s", d.addr, d.dialTimeout)
	case err := <-ready:
		return err
	}
}

// reconnectLoop 断线重连：等 disconnect 信号 → 退避 → 重建容器并拨号。
func (d *Dog) reconnectLoop() {
	defer d.reconnectWG.Done()
	attempt := 0
	for {
		select {
		case <-d.stopCh:
			return
		case <-d.ctx.Done():
			return
		case <-d.disconnectCh:
		}

		for {
			if d.isStopped() || d.ctx.Err() != nil {
				return
			}
			attempt++
			wait := d.backoff(attempt)
			log.Printf("[%s] reconnecting in %s (attempt=%d)\n", d.Name, wait, attempt)
			if !sleepCtx(d.ctx, d.stopCh, wait) {
				return
			}
			ready := make(chan error, 1)
			// 旧容器已 dial 过，不能复用：重建并回绑 services。
			cli := d.newClient()
			services.Rebind(cli)
			if err := d.dial(d.ctx, ready); err != nil {
				log.Printf("[%s] reconnect failed: %v\n", d.Name, err)
				continue
			}
			if err := d.waitReady(d.ctx, ready); err != nil {
				log.Printf("[%s] reconnect failed: %v\n", d.Name, err)
				continue
			}
			log.Printf("[%s] reconnected to %s\n", d.Name, d.addr)
			attempt = 0
			break
		}
	}
}

// backoff 指数退避（base * 2^(attempt-1)），上限 reconnectMax。
func (d *Dog) backoff(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	wait := d.reconnectBase
	for i := 1; i < attempt; i++ {
		wait *= 2
		if wait >= d.reconnectMax {
			return d.reconnectMax
		}
	}
	if wait > d.reconnectMax {
		return d.reconnectMax
	}
	return wait
}

// markConnected 由 Handler.OnReady 调用：标记已连接并启动注册/心跳。
func (d *Dog) markConnected() {
	d.mu.Lock()
	d.connected = true
	d.mu.Unlock()
	d.startHeartbeat()
}

// markDisconnected 由 Handler.OnClose/OnError 调用：停心跳并通知重连。
func (d *Dog) markDisconnected(err error) {
	d.mu.Lock()
	wasConnected := d.connected
	d.connected = false
	d.mu.Unlock()

	d.stopHeartbeat()
	if err != nil {
		log.Printf("[%s] connection error: %v\n", d.Name, err)
	}
	if !wasConnected || d.isStopped() {
		return
	}
	select {
	case d.disconnectCh <- struct{}{}:
	default:
	}
}

// startHeartbeat 启动注册 + 心跳循环；先停掉上一轮，避免重连后双份心跳。
func (d *Dog) startHeartbeat() {
	if d.isStopped() || d.ctx.Err() != nil {
		return
	}
	d.stopHeartbeat()
	d.heartbeatWG.Wait()

	hbCtx, cancel := context.WithCancel(d.ctx)
	d.mu.Lock()
	d.heartbeatCancel = cancel
	d.mu.Unlock()

	d.heartbeatWG.Add(1)
	go d.runHeartbeat(hbCtx)
}

func (d *Dog) runHeartbeat(ctx context.Context) {
	defer d.heartbeatWG.Done()

	registered := false
	if err := d.register(ctx); err != nil {
		log.Printf("[%s] register failed: %v\n", d.Name, err)
	} else {
		registered = true
	}

	ticker := time.NewTicker(d.heartbeatInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !registered {
				if err := d.register(ctx); err != nil {
					log.Printf("[%s] register failed: %v\n", d.Name, err)
					continue
				}
				registered = true
			}
			callCtx, cancel := context.WithTimeout(ctx, d.callTimeout)
			err := services.Heartbeat(callCtx, registry.HeartbeatRequest{
				ServiceName: d.serviceName,
				InstanceID:  d.instanceID,
			})
			cancel()
			if err != nil {
				log.Printf("[%s] heartbeat failed: %v\n", d.Name, err)
				// 服务端可能已把实例剔除（TTL 过期），下一轮先重新注册。
				registered = false
			}
		}
	}
}

// register 带重试地向注册中心登记自己。
func (d *Dog) register(ctx context.Context) error {
	var lastErr error
	for i := 0; i < d.registerAttempts; i++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		callCtx, cancel := context.WithTimeout(ctx, d.callTimeout)
		_, err := services.Register(callCtx, registry.RegisterRequest{
			Instance: registry.ServiceInstance{
				ID:     d.instanceID,
				Name:   d.serviceName,
				Host:   d.addr,
				Tags:   []string{d.Name},
				Status: 1, // UP
			},
		})
		cancel()
		if err == nil {
			return nil
		}
		lastErr = err
		if !sleepCtx(ctx, d.stopCh, d.registerInterval) {
			return lastErr
		}
	}
	if lastErr == nil {
		lastErr = errors.New("register not attempted")
	}
	return lastErr
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

// Stop 停止 Dog：注销、通知 Door 退出、停掉重连与心跳。可重复调用。
func (d *Dog) Stop() error {
	return d.shutdown(true)
}

// Close 停止 Dog，但不通知 Door 退出（短命令行的用法）。可重复调用。
func (d *Dog) Close() error {
	return d.shutdown(false)
}

func (d *Dog) shutdown(notifyExit bool) error {
	d.stopOnce.Do(func() {
		d.closed.Store(true)
		close(d.stopCh)
	})
	// 先停心跳（它会调 RPC），再等重连循环退出，最后才做注销类调用。
	d.stopHeartbeat()
	d.heartbeatWG.Wait()
	d.reconnectWG.Wait()

	instanceID, serviceName := d.instanceID, d.serviceName

	if serviceName != "" && instanceID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), d.callTimeout)
		err := services.Deregister(ctx, registry.DeregisterRequest{
			ServiceName: serviceName,
			InstanceID:  instanceID,
		})
		cancel()
		if err != nil {
			log.Printf("[%s] %s-%s Deregister failed: %v\n", d.Name, serviceName, instanceID, err)
		} else {
			log.Printf("[%s] %s-%s Deregister success\n", d.Name, serviceName, instanceID)
		}
	}

	if notifyExit {
		// 这里不能走 d.Call()：shutdown 已经把 Dog 标记为 closed，
		// Call 会直接拒绝（否则 Exit 永远发不出去，Door 收不到退出信号）。
		ctx, cancel := context.WithTimeout(context.Background(), d.callTimeout)
		var (
			status interface{}
			err    error
		)
		if cli := d.rpcOrNil(); cli != nil {
			status, err = cli.Call(ctx, "command.Exit", 200)
		} else {
			err = errors.New("rpc client not initialized")
		}
		cancel()
		if err != nil {
			log.Printf("[%s] Exit failed: %v\n", d.Name, err)
		} else {
			log.Printf("[%s] Exit success: %s\n", d.Name, status)
		}
	}

	d.mu.RLock()
	conn := d.clientConn
	d.mu.RUnlock()
	if conn != nil {
		// 客户端 Connect 没有 listener，Close 主要清理调试服务；
		// 真正的连接随 ctx 取消/进程退出关闭。
		_ = conn.Close()
	}
	return nil
}

func (d *Dog) Call(ctx context.Context, mtd string, args ...any) (interface{}, error) {
	cli, err := d.rpc()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, mtd, args...)
}

// CallWithHeader calls a service method with a custom header.
func (d *Dog) CallWithHeader(ctx context.Context, header message.Header, method string, args ...any) (interface{}, error) {
	cli, err := d.rpc()
	if err != nil {
		return nil, err
	}
	return cli.CallWithHeader(ctx, header, method, args...)
}

// Connected 当前是否已连上 Door。
func (d *Dog) Connected() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.connected
}

func (d *Dog) rpc() (*sloth.ServerRpc, error) {
	if d.closed.Load() {
		return nil, ErrDogClosed
	}
	d.mu.RLock()
	cli := d.clientRpc
	d.mu.RUnlock()
	if cli == nil {
		return nil, errors.New("dog: rpc client not initialized")
	}
	return cli, nil
}

func (d *Dog) rpcOrNil() *sloth.ServerRpc {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.clientRpc
}

func (d *Dog) isStopped() bool {
	if d.closed.Load() {
		return true
	}
	select {
	case <-d.stopCh:
		return true
	default:
		return false
	}
}

// sleepCtx 可被 ctx 取消或 Dog 停止打断的等待；被打断返回 false。
func sleepCtx(ctx context.Context, stopCh <-chan struct{}, wait time.Duration) bool {
	if wait <= 0 {
		return true
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-stopCh:
		return false
	case <-timer.C:
		return true
	}
}
