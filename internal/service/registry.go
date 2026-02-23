package service

import (
	"context"
	"errors"
	"keeper/internal/registry"
)

type RegistryService struct {
	Store *registry.RegistryStore
}

func NewRegistryService() *RegistryService {
	return &RegistryService{
		Store: registry.NewRegistryStore(),
	}
}

func (s *RegistryService) Register(ctx context.Context, req registry.RegisterRequest) (registry.RegisterResponse, error) {
	s.Store.Register(req.Instance)
	return registry.RegisterResponse{TTL: int64(registry.DefaultTTL.Seconds())}, nil
}

func (s *RegistryService) Deregister(ctx context.Context, req registry.DeregisterRequest) (string, error) {
	s.Store.Deregister(req.ServiceName, req.InstanceID)
	return "ok", nil
}

func (s *RegistryService) Heartbeat(ctx context.Context, req registry.HeartbeatRequest) (string, error) {
	if !s.Store.Heartbeat(req.ServiceName, req.InstanceID) {
		return "", errors.New("instance not found, please register first")
	}
	return "ok", nil
}

func (s *RegistryService) Discovery(ctx context.Context, req registry.DiscoveryRequest) (registry.DiscoveryResponse, error) {
	instances := s.Store.GetInstances(req.ServiceName)
	return registry.DiscoveryResponse{Instances: instances}, nil
}
