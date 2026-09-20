package services

import (
	"errors"
	"sync"

	"github.com/w6xian/sloth/v4"
)

// ErrNotInitialized RPC 客户端尚未绑定（Dog 还没连上 Door，或连接已重建但尚未回绑）。
//
// 这是**可恢复**的状态：调用方应当退避重试，而不是当成程序缺陷。
// 判定请用 errors.Is，不要匹配错误文本。
var ErrNotInitialized = errors.New("keeper: rpc client not initialized")

// rpcHolder 保存某一类服务使用的 RPC 客户端，支持重新绑定。
//
// 为什么不是 sync.Once：Dog 断线重连后会拿到一个全新的 *sloth.ServerRpc，
// 必须能把旧引用换掉；Once 一旦 fire 就无法重置，重连后的调用会一直打在死连接上。
type rpcHolder struct {
	mu  sync.RWMutex
	cli *sloth.ServerRpc
}

// init 首次绑定。cli 为 nil 时保留已有客户端，兼容 InitX(nil) 的旧语义。
func (h *rpcHolder) init(cli *sloth.ServerRpc) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if cli != nil || h.cli == nil {
		h.cli = cli
	}
}

// get 返回可用客户端；未绑定时返回错误而不是 nil，
// 避免调用方拿到 nil 后直接解引用 panic（旧实现在 Cache.Get 上就是这样崩的）。
func (h *rpcHolder) get() (*sloth.ServerRpc, error) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if h.cli == nil {
		return nil, ErrNotInitialized
	}
	return h.cli, nil
}

// set 重新绑定（重连 / 测试注入）。
func (h *rpcHolder) set(cli *sloth.ServerRpc) {
	h.mu.Lock()
	h.cli = cli
	h.mu.Unlock()
}

// Rebind 把所有服务客户端切到新的 RPC 连接。
//
// Dog 断线重连后必须调用它，否则 services.* 仍持有已失效的连接。
// cli 为 nil 时不做任何改动（避免一次误调用把全部服务打空）。
func Rebind(cli *sloth.ServerRpc) {
	if cli == nil {
		return
	}
	commandClient.set(cli)
	logClient.set(cli)
	registryClient.set(cli)
	cacheClient.set(cli)
	scriptClient.set(cli)
}

// Reset 解绑所有服务客户端，仅供测试使用（生产代码不要调用）。
func Reset() {
	commandClient.set(nil)
	logClient.set(nil)
	registryClient.set(nil)
	cacheClient.set(nil)
	scriptClient.set(nil)
}
