package services

import (
	"context"
	"sync"

	"github.com/w6xian/sloth/v4"
)

var (
	cacheClient rpcHolder

	// bucketMu 保护全局 bucket 名：Use 在业务 goroutine 里写，Get/Set/Del 在
	// 别的 goroutine 里读，旧实现是无保护的全局变量（data race，且两个 goroutine
	// 用不同 bucket 时会互相串到对方的 bucket 上）。
	bucketMu   sync.RWMutex
	bucketName string
)

// InitCache 绑定 cache 服务的 RPC 客户端。
func InitCache(cli *sloth.ServerRpc) *Cache {
	cacheClient.init(cli)
	return &Cache{}
}

type Cache struct{}

// Use 设置后续 Get/Set/Del 使用的 bucket。
//
// 注意 bucket 是**进程级全局**的：并发使用多个 bucket 请各自建 Cache 作用域，
// 不要依赖 Use 的时序。
func Use(ctx context.Context, key string) *Cache {
	InitCache(nil)
	bucketMu.Lock()
	bucketName = key
	bucketMu.Unlock()
	return &Cache{}
}

func currentBucket() string {
	bucketMu.RLock()
	defer bucketMu.RUnlock()
	return bucketName
}

// Get cache value by key
func Get(ctx context.Context, key string) ([]byte, error) {
	cli, err := cacheClient.get()
	if err != nil {
		return nil, err
	}
	return cli.Call(ctx, "cache.Get", currentBucket(), key)
}

// Set cache value by key
func Set(ctx context.Context, key string, value []byte) error {
	cli, err := cacheClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "cache.Set", currentBucket(), key, value)
	return err
}

// Del cache value by key
func Del(ctx context.Context, key string) error {
	cli, err := cacheClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "cache.Del", currentBucket(), key)
	return err
}
