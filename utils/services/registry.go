package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/sloth"
)

var (
	registryOnce sync.Once
	registrySvc  *RegistryService
)

func InitRegistry(cli *sloth.ServerRpc) *RegistryService {
	registryOnce.Do(func() {
		registrySvc = &RegistryService{cli: cli}
	})
	return registrySvc
}

type RegistryService struct {
	cli *sloth.ServerRpc
}

func Register(ctx context.Context, req registry.RegisterRequest) (*registry.RegisterResponse, error) {
	newRegistry := InitRegistry(nil)
	if newRegistry.cli == nil {
		return nil, fmt.Errorf("registry client is nil")
	}
	resp, err := newRegistry.cli.Call(ctx, "registry.Register", req)
	if err != nil {
		return nil, err
	}
	var res registry.RegisterResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func Deregister(ctx context.Context, req registry.DeregisterRequest) error {
	newRegistry := InitRegistry(nil)
	if newRegistry.cli == nil {
		return fmt.Errorf("registry client is nil")
	}
	_, err := newRegistry.cli.Call(ctx, "registry.Deregister", req)
	return err
}

func Heartbeat(ctx context.Context, req registry.HeartbeatRequest) error {
	newRegistry := InitRegistry(nil)
	if newRegistry.cli == nil {
		return fmt.Errorf("registry client is nil")
	}
	_, err := newRegistry.cli.Call(ctx, "registry.Heartbeat", req)
	return err
}

func Discovery(ctx context.Context, req registry.DiscoveryRequest) (*registry.DiscoveryResponse, error) {
	newRegistry := InitRegistry(nil)
	if newRegistry.cli == nil {
		return nil, fmt.Errorf("registry client is nil")
	}
	resp, err := newRegistry.cli.Call(ctx, "registry.Discovery", req)
	if err != nil {
		return nil, err
	}
	var res registry.DiscoveryResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
