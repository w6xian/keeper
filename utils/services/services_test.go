package services

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/sloth/v4"
)

// shortCtx 所有断言用的 ctx 都带超时：未连接的 client 上调用可能阻塞，
// 测试不能因为"卡住"而挂死。
func shortCtx(t *testing.T) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	t.Cleanup(cancel)
	return ctx
}

// TestUninitializedReturnsErrorNotPanic 覆盖"未初始化就 panic"的缺陷。
// 旧实现里 Cache.Get 少了 nil 检查，InitX(nil) 之后调用就是 nil 解引用崩溃。
func TestUninitializedReturnsErrorNotPanic(t *testing.T) {
	Reset()
	defer Reset()
	ctx := shortCtx(t)

	cases := map[string]func() error{
		"cache.Get":    func() error { _, err := Get(ctx, "k"); return err },
		"cache.Set":    func() error { return Set(ctx, "k", []byte("v")) },
		"cache.Del":    func() error { return Del(ctx, "k") },
		"command.Exit": func() error { _, err := Exit(ctx, 200); return err },
		"command.Keep": func() error { _, err := KeepAlive(ctx, 200); return err },
		"command.Stop": func() error { _, err := StopService(ctx, "s"); return err },
		"command.Run":  func() error { _, err := StartService(ctx, "s"); return err },
		"command.Rel":  func() error { _, err := ReloadService(ctx, "s"); return err },
		"log.Info":     func() error { return Info(ctx, "m") },
		"log.Debug":    func() error { return Debug(ctx, "m") },
		"log.Warn":     func() error { return Warn(ctx, "m") },
		"log.Error":    func() error { return Error(ctx, "m") },
		"log.Log":      func() error { return Log(ctx, LogRequest{Level: "info"}) },
		"reg.Register": func() error { _, err := Register(ctx, registry.RegisterRequest{}); return err },
		"reg.Dereg":    func() error { return Deregister(ctx, registry.DeregisterRequest{}) },
		"reg.Beat":     func() error { return Heartbeat(ctx, registry.HeartbeatRequest{}) },
		"reg.Disc":     func() error { _, err := Discovery(ctx, registry.DiscoveryRequest{}); return err },
		"script.Run":   func() error { _, err := Run(ctx, "s"); return err },
		"script.File":  func() error { _, err := LoadFile(ctx, "f"); return err },
	}
	for name, fn := range cases {
		t.Run(name, func(t *testing.T) {
			if err := fn(); !errors.Is(err, ErrNotInitialized) {
				t.Fatalf("err = %v, want ErrNotInitialized", err)
			}
		})
	}
}

func TestInitNilDoesNotPanic(t *testing.T) {
	Reset()
	defer Reset()

	if InitCache(nil) == nil || InitCommand(nil) == nil || InitLog(nil) == nil ||
		InitRegistry(nil) == nil || InitScript(nil) == nil {
		t.Fatal("InitX(nil) should return a non-nil service handle")
	}
}

// TestRebind 覆盖 Dog 重连的能力：必须能把 services 里的旧连接换掉。
func TestRebind(t *testing.T) {
	Reset()
	defer Reset()

	Rebind(sloth.DefaultClient())
	ctx := shortCtx(t)
	if err := Info(ctx, "x"); errors.Is(err, ErrNotInitialized) {
		t.Fatal("client should be bound after Rebind")
	}

	// nil 必须是 no-op：误调用不能把全部服务打空。
	Rebind(nil)
	if err := Info(ctx, "x"); errors.Is(err, ErrNotInitialized) {
		t.Fatal("Rebind(nil) must not unbind existing clients")
	}

	Reset()
	if err := Info(ctx, "x"); !errors.Is(err, ErrNotInitialized) {
		t.Fatal("Reset should unbind all clients")
	}
}

// TestCacheBucketConcurrent 用 -race 暴露全局 bucket 的并发写/读。
func TestCacheBucketConcurrent(t *testing.T) {
	Reset()
	defer Reset()

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx := context.Background()
			for j := 0; j < 100; j++ {
				Use(ctx, string(rune('a'+i)))
				_ = currentBucket()
				_, _ = Get(ctx, "k")
			}
		}(i)
	}
	wg.Wait()
}
