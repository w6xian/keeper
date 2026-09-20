package services

import (
	"context"
	"encoding/json"

	"github.com/w6xian/keeper/registry"
	"github.com/w6xian/sloth/v4"
)

var registryClient rpcHolder

// InitRegistry 绑定 registry 服务的 RPC 客户端。
func InitRegistry(cli *sloth.ServerRpc) *RegistryService {
	registryClient.init(cli)
	return &RegistryService{}
}

type RegistryService struct{}

func Register(ctx context.Context, req registry.RegisterRequest) (*registry.RegisterResponse, error) {
	cli, err := registryClient.get()
	if err != nil {
		return nil, err
	}
	resp, err := cli.Call(ctx, "registry.Register", req)
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
	cli, err := registryClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "registry.Deregister", req)
	return err
}

func Heartbeat(ctx context.Context, req registry.HeartbeatRequest) error {
	cli, err := registryClient.get()
	if err != nil {
		return err
	}
	_, err = cli.Call(ctx, "registry.Heartbeat", req)
	return err
}

func Discovery(ctx context.Context, req registry.DiscoveryRequest) (*registry.DiscoveryResponse, error) {
	cli, err := registryClient.get()
	if err != nil {
		return nil, err
	}
	resp, err := cli.Call(ctx, "registry.Discovery", req)
	if err != nil {
		return nil, err
	}
	var res registry.DiscoveryResponse
	if err := json.Unmarshal(resp, &res); err != nil {
		return nil, err
	}
	return &res, nil
}
