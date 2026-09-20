package keeper

import (
	"context"
	"log"

	"github.com/w6xian/sloth/v4/bucket"
)

// Handler 处理 Dog（客户端）侧的连接事件。
//
// 传输换成 TCP 后必须实现 handler.TcpClientHandleMessage：TCP 没有 HTTP 握手，
// 旧的 *http.Response 版钩子（slots.Client / handler.IClientHandleMessage）
// 在 TCP 客户端上会被静默忽略（见 option.WithTcpClientHandleMessage 的说明），
// 继续用它等于"连上了但永远收不到 OnReady"。
type Handler struct {
	ready chan error
	dog   *Dog
}

// OnConnect 连接建立前触发；返回错误会中断本次拨号。
func (h *Handler) OnConnect(ctx context.Context, addr string) error {
	log.Printf("[%s] connecting to %s\n", h.dog.Name, addr)
	return nil
}

// OnReady 连接就绪（TCP 拨号成功、读写循环已启动）。
func (h *Handler) OnReady(ctx context.Context, ch bucket.IChannel) error {
	notifyReady(h.ready, nil)
	h.dog.markConnected()
	return nil
}

// OnData 收到服务端推送的裸数据。
func (h *Handler) OnData(ctx context.Context, ch bucket.IChannel, msg []byte) error {
	return nil
}

// OnClose 连接关闭：停止心跳并触发重连（若开启）。
func (h *Handler) OnClose(ctx context.Context, ch bucket.IChannel) error {
	log.Printf("[%s] connection closed\n", h.dog.Name)
	h.dog.markDisconnected(nil)
	return nil
}

// OnError 连接出错：与 OnClose 同样处理（都会导致后续调用失败）。
func (h *Handler) OnError(ctx context.Context, ch bucket.IChannel, err error) error {
	log.Printf("[%s] connection error: %v\n", h.dog.Name, err)
	h.dog.markDisconnected(err)
	return nil
}

// notifyReady 非阻塞通知：ready 是容量为 1 的缓冲通道，
// 首次 OnReady 后即使调用方已超时离开，这里也不会把 goroutine 卡住。
func notifyReady(ready chan error, err error) {
	if ready == nil {
		return
	}
	select {
	case ready <- err:
	default:
	}
}
