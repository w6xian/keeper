package keeper

import (
	"context"
	"fmt"

	"github.com/w6xian/sloth/v2"
	"github.com/w6xian/sloth/v2/bucket"
	"github.com/w6xian/sloth/v2/types"
)

type ServerHandler struct {
	server           *sloth.ServerRpc
	onPendingHandler func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error
	onCloseHandler   func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error
	onErrorHandler   func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo, err error) error
}

// OnClose implements wsocket.IServerHandleMessage.
func (h *ServerHandler) OnClose(ctx context.Context, s types.IBucket, ch bucket.IChannel) error {
	fmt.Println("OnClose")
	return nil
}

// OnError implements wsocket.IServerHandleMessage.
func (h *ServerHandler) OnError(ctx context.Context, s types.IBucket, ch bucket.IChannel, err error) error {
	fmt.Println("OnError:", err)
	return nil
}

// OnOpen implements wsocket.IServerHandleMessage.
func (h *ServerHandler) OnOpen(ctx context.Context, s types.IBucket, ch bucket.IChannel) error {
	fmt.Println("OnOpen")
	return nil
}

func (h *ServerHandler) OnData(ctx context.Context, s types.IBucket, ch bucket.IChannel, msgType int, message []byte) error {
	fmt.Println("OnData:", msgType, string(message), ch.UserId())

	return nil
}
