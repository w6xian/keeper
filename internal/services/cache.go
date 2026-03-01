package services

import (
	"context"
	"fmt"
	"sync"

	"github.com/w6xian/sloth"
)

var (
	cacheOnce sync.Once
	cache     *Cache
)

func InitCache(cli *sloth.ServerRpc) *Cache {
	cacheOnce.Do(func() {
		cache = &Cache{cli: cli}
	})
	return cache
}

type Cache struct {
	cli *sloth.ServerRpc
}

// Get cache value by key
func Get(ctx context.Context, key string) ([]byte, error) {
	newCache := InitCache(nil)
	return newCache.cli.Call(ctx, "cache.Get", key)
}

// Set cache value by key
func Set(ctx context.Context, key string, value []byte) error {
	newCache := InitCache(nil)
	if newCache.cli == nil {
		return fmt.Errorf("cache client is nil")
	}
	_, err := newCache.cli.Call(ctx, "cache.Set", key, value)
	return err
}

// Del cache value by key
func Del(ctx context.Context, key string) error {
	newCache := InitCache(nil)
	if newCache.cli == nil {
		return fmt.Errorf("cache client is nil")
	}
	_, err := newCache.cli.Call(ctx, "cache.Del", key)
	return err
}
