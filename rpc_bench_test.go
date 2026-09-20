package keeper

import (
	"context"
	"io"
	"net"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/w6xian/keeper/utils/services"
)

func benchFreeAddr(b *testing.B) string {
	b.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	_ = ln.Close()
	return addr
}

func benchWaitReady(b *testing.B, addr string) {
	b.Helper()
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 300*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	b.Fatalf("door is not listening on %s", addr)
}

// BenchmarkKeeperRPCRoundTrip keeper 的一次 RPC 往返（TCP 传输）。
// 与 BenchmarkRawTCPRoundTrip 对照，可以算出 RPC 框架本身加了多少开销。
func BenchmarkKeeperRPCRoundTrip(b *testing.B) {
	services.Reset()
	defer services.Reset()

	addr := benchFreeAddr(b)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	wg := &sync.WaitGroup{}
	door := NewDoor(ctx, wg,
		WithDoorAddr(addr),
		WithDoorName(filepath.Join(b.TempDir(), "bench.door")),
	)
	go func() { _ = door.Start() }()
	benchWaitReady(b, addr)

	dog := NewDog(ctx, addr, "/ws", WithDogAutoReconnect(false), WithDogHeartbeatInterval(time.Hour))
	dog.InitService()
	if err := dog.KeepAlive(); err != nil {
		b.Fatalf("KeepAlive() error = %v", err)
	}
	defer func() { _ = dog.Close() }()
	defer func() { _ = door.Stop() }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := services.KeepAlive(ctx, 200); err != nil {
			b.Fatalf("rpc error = %v", err)
		}
	}
}

// BenchmarkRawTCPRoundTrip 对照组：不经任何 RPC 框架的裸 TCP 回环往返。
func BenchmarkRawTCPRoundTrip(b *testing.B) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()

	go func() {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		defer func() { _ = conn.Close() }()
		_, _ = io.Copy(conn, conn)
	}()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		b.Fatalf("dial: %v", err)
	}
	defer func() { _ = conn.Close() }()

	payload := []byte("keepalive-200")
	buf := make([]byte, len(payload))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := conn.Write(payload); err != nil {
			b.Fatalf("write: %v", err)
		}
		if _, err := io.ReadFull(conn, buf); err != nil {
			b.Fatalf("read: %v", err)
		}
	}
}
