package service

import (
	"context"

	"github.com/w6xian/keeper/utils/fsm"
)

type Cache struct {
	fsmStore fsm.IFSM
}

func NewCache(fsmStore fsm.IFSM) *Cache {
	return &Cache{fsmStore: fsmStore}
}

// Get/Set/Del cache
func (c *Cache) Get(ctx context.Context, bucket, key string) ([]byte, error) {
	value, err := c.fsmStore.Get(bucket, key)
	return value, err
}

func (c *Cache) Set(ctx context.Context, bucket, key string, value []byte) ([]byte, error) {
	return nil, c.fsmStore.Set(bucket, key, value)
}

func (c *Cache) Del(ctx context.Context, bucket, key string) ([]byte, error) {
	return nil, c.fsmStore.Del(bucket, key)
}
