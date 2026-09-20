package keeper

import (
	"time"

	"github.com/w6xian/keeper/utils/fsm"
)

type DogOption func(*Dog)

type DoorOption func(*Door)

type IWatcher interface {
}

func WithDogName(name string) DogOption {
	return func(d *Dog) {
		d.Name = name
	}
}

// WithDogWatcher 注册一个可被服务端反调的服务对象（方法由 sloth 反射注册）。
func WithDogWatcher(watcher IWatcher) DogOption {
	return func(d *Dog) {
		d.Watcher = watcher
	}
}

// WithDogAutoReconnect 是否开启断线自动重连（默认开启）。
//
// 关闭后连接断开即结束，适合一次性命令（start/stop/reload 这类短进程）。
func WithDogAutoReconnect(enabled bool) DogOption {
	return func(d *Dog) {
		d.autoReconnect = enabled
	}
}

// WithDogDialTimeout 拨号等待 OnReady 的超时（默认 3s）。
func WithDogDialTimeout(timeout time.Duration) DogOption {
	return func(d *Dog) {
		if timeout > 0 {
			d.dialTimeout = timeout
		}
	}
}

// WithDogHeartbeatInterval 心跳间隔（默认 5s，应小于注册中心 TTL 的一半）。
func WithDogHeartbeatInterval(interval time.Duration) DogOption {
	return func(d *Dog) {
		if interval > 0 {
			d.heartbeatInterval = interval
		}
	}
}

// WithDogReconnectBackoff 重连退避：base * 2^(n-1)，上限 max（默认 500ms / 15s）。
func WithDogReconnectBackoff(base, max time.Duration) DogOption {
	return func(d *Dog) {
		if base > 0 {
			d.reconnectBase = base
		}
		if max > 0 {
			d.reconnectMax = max
		}
	}
}

func WithDoorName(name string) DoorOption {
	return func(d *Door) {
		d.Name = name
	}
}

func WithFSMStore(fsmStore fsm.IFSM) DoorOption {
	return func(d *Door) {
		d.fsmStore = fsmStore
	}
}

func WithDoorAddr(addr string) DoorOption {
	return func(d *Door) {
		d.addr = addr
	}
}
