package service

import (
	"context"
	"errors"

	"github.com/w6xian/keeper/registry"
)

// ErrInstanceNotFound 心跳打到了不存在的实例（未注册或已被 TTL 剔除）。
//
// 用 errors.Is 判定；不要匹配错误文本。
var ErrInstanceNotFound = errors.New("instance not found, please register first")

type RegistryService struct {
	Store *registry.RegistryStore
}

func NewRegistryService() *RegistryService {
	return &RegistryService{
		Store: registry.NewRegistryStore(),
	}
}

// Close 停止注册中心的过期剔除协程（Door 关闭时调用，避免 goroutine 泄漏）。
func (s *RegistryService) Close() error {
	if s.Store == nil {
		return nil
	}
	return s.Store.Close()
}

func (s *RegistryService) Register(ctx context.Context, req registry.RegisterRequest) (registry.RegisterResponse, error) {
	s.Store.Register(req.Instance)
	return registry.RegisterResponse{TTL: int64(registry.DefaultTTL.Seconds())}, nil
}

func (s *RegistryService) Deregister(ctx context.Context, req registry.DeregisterRequest) (string, error) {
	s.Store.Deregister(req.ServiceName, req.InstanceID)
	return "ok", nil
}

func (s *RegistryService) Heartbeat(ctx context.Context, req registry.HeartbeatRequest) ([]byte, error) {
	if !s.Store.Heartbeat(req.ServiceName, req.InstanceID) {
		return nil, ErrInstanceNotFound
	}
	return []byte("ok"), nil
}

func (s *RegistryService) Discovery(ctx context.Context, req registry.DiscoveryRequest) (registry.DiscoveryResponse, error) {
	instances := s.Store.GetInstances(req.ServiceName)
	return registry.DiscoveryResponse{Instances: instances}, nil
}
