package service

import (
	"context"

	"github.com/w6xian/keeper/internal/fsm"
)

type Cache struct {
	fsmStore fsm.IFSM
}

func NewCache(fsmStore fsm.IFSM) *Cache {
	return &Cache{fsmStore: fsmStore}
}

// Get/Set/Del cache
func (c *Cache) Get(ctx context.Context, key string) ([]byte, error) {
	return c.fsmStore.Get(key)
}

func (c *Cache) Set(ctx context.Context, key string, value []byte) ([]byte, error) {
	return nil, c.fsmStore.Set(key, value)
}

func (c *Cache) Del(ctx context.Context, key string) ([]byte, error) {
	return nil, c.fsmStore.Del(key)
}
