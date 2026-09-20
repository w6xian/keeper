package keeper

import (
	"context"
	"errors"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/keeper/utils/services"
)

func freeAddr(t *testing.T) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func waitTCPReady(t *testing.T, addr string) {
	t.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("door is not listening on %s", addr)
}

// TestDoorDogTCPRoundTrip sloth v4 + TCP 传输的端到端验证：
// Dog 拨号 → 调用 command / registry / script 三个服务 → Stop 通知 Door 退出。
func TestDoorDogTCPRoundTrip(t *testing.T) {
	services.Reset()
	t.Cleanup(services.Reset)

	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	door := NewDoor(ctx, wg,
		WithDoorAddr(addr),
		WithDoorName(filepath.Join(t.TempDir(), "test.door")),
	)
	serveErr := make(chan error, 1)
	go func() { serveErr <- door.Start() }()
	waitTCPReady(t, addr)

	dog := NewDog(ctx, addr, "/ws", WithDogAutoReconnect(false))
	dog.InitService()
	if err := dog.KeepAlive(); err != nil {
		t.Fatalf("KeepAlive() error = %v", err)
	}
	if !dog.Connected() {
		t.Fatal("dog.Connected() = false after KeepAlive")
	}

	callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
	defer callCancel()

	if _, err := services.KeepAlive(callCtx, 200); err != nil {
		t.Fatalf("command.KeepAlive over TCP error = %v", err)
	}
	if _, err := services.Register(callCtx, registry.RegisterRequest{
		Instance: registry.ServiceInstance{ID: "i1", Name: "svc"},
	}); err != nil {
		t.Fatalf("registry.Register over TCP error = %v", err)
	}
	discovered, err := services.Discovery(callCtx, registry.DiscoveryRequest{ServiceName: "svc"})
	if err != nil {
		t.Fatalf("registry.Discovery over TCP error = %v", err)
	}
	if len(discovered.Instances) != 1 || discovered.Instances[0].ID != "i1" {
		t.Fatalf("Discovery() = %v, want exactly [i1]", discovered.Instances)
	}
	if err := services.Heartbeat(callCtx, registry.HeartbeatRequest{ServiceName: "svc", InstanceID: "i1"}); err != nil {
		t.Fatalf("registry.Heartbeat over TCP error = %v", err)
	}
	if err := services.Heartbeat(callCtx, registry.HeartbeatRequest{ServiceName: "svc", InstanceID: "ghost"}); err == nil {
		t.Fatal("Heartbeat() on unknown instance should fail")
	}

	// Stop 会走 command.Exit → 服务端 wg.Done()。
	if err := dog.Stop(); err != nil {
		t.Fatalf("dog.Stop() error = %v", err)
	}
	if err := dog.Stop(); err != nil {
		t.Fatalf("second dog.Stop() error = %v", err)
	}
	if _, err := dog.Call(context.Background(), "command.KeepAlive", 200); !errors.Is(err, ErrDogClosed) {
		t.Fatalf("Call() after stop error = %v, want ErrDogClosed", err)
	}

	if err := door.Stop(); err != nil {
		t.Fatalf("door.Stop() error = %v", err)
	}
	select {
	case err := <-serveErr:
		if err != nil && !errors.Is(err, net.ErrClosed) {
			t.Logf("serve returned: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("door did not stop after Stop()")
	}
}

// TestDogReconnectsAfterDoorRestart 高可用的核心场景：Door 重启后 Dog 自动重连并回绑 RPC。
func TestDogReconnectsAfterDoorRestart(t *testing.T) {
	services.Reset()
	t.Cleanup(services.Reset)

	addr := freeAddr(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg1 := &sync.WaitGroup{}
	door1 := NewDoor(ctx, wg1,
		WithDoorAddr(addr),
		WithDoorName(filepath.Join(t.TempDir(), "door1")),
	)
	serveErr1 := make(chan error, 1)
	go func() { serveErr1 <- door1.Start() }()
	waitTCPReady(t, addr)

	dog := NewDog(ctx, addr, "/ws",
		WithDogReconnectBackoff(200*time.Millisecond, 1*time.Second),
		WithDogHeartbeatInterval(time.Hour), // 本测试只关心重连，不发心跳
	)
	dog.InitService()
	if err := dog.KeepAlive(); err != nil {
		t.Fatalf("KeepAlive() error = %v", err)
	}

	// 模拟 Door 宕机
	if err := door1.Stop(); err != nil {
		t.Fatalf("door1.Stop() error = %v", err)
	}
	<-serveErr1

	// 同一地址重启 Door
	wg2 := &sync.WaitGroup{}
	door2 := NewDoor(ctx, wg2,
		WithDoorAddr(addr),
		WithDoorName(filepath.Join(t.TempDir(), "door2")),
	)
	serveErr2 := make(chan error, 1)
	go func() { serveErr2 <- door2.Start() }()
	waitTCPReady(t, addr)

	reconnected := false
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if dog.Connected() {
			reconnected = true
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if !reconnected {
		t.Fatal("dog did not reconnect after door restart")
	}

	// 重连后 services 必须指向新连接，否则所有调用都会打在死连接上。
	callCtx, callCancel := context.WithTimeout(ctx, 5*time.Second)
	defer callCancel()
	deadline = time.Now().Add(5 * time.Second)
	var lastErr error
	for time.Now().Before(deadline) {
		if _, lastErr = services.KeepAlive(callCtx, 200); lastErr == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if lastErr != nil {
		t.Fatalf("services call after reconnect error = %v", lastErr)
	}

	if err := dog.Stop(); err != nil {
		t.Fatalf("dog.Stop() error = %v", err)
	}
	if err := door2.Stop(); err != nil {
		t.Fatalf("door2.Stop() error = %v", err)
	}
	<-serveErr2
}
