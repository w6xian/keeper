package keeper

import (
	"context"
	"fmt"

	"github.com/gorilla/websocket"
	"github.com/w6xian/sloth/v2"
	"github.com/w6xian/sloth/v2/types"
)

type Handler struct {
	server           *sloth.ServerRpc
	onPendingHandler func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error
}

func (h *Handler) OnConnected(f func(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error) {
	h.onPendingHandler = f
}

// OnClose is called when connection is closed
func (h *Handler) OnClose(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error {
	fmt.Println("OnClose:", ch.GetUserId())
	return nil
}

// OnData handles received messages
func (h *Handler) OnData(ctx context.Context, c types.IConnRpc, ch types.IConnInfo, msgType int, message []byte) error {
	if msgType == websocket.TextMessage {
		fmt.Println("HandleMessage:", 1, string(message))
	}

	return nil
}

// OnError handles errors
func (h *Handler) OnError(ctx context.Context, c types.IConnRpc, ch types.IConnInfo, err error) error {
	fmt.Println("OnError:", err.Error())
	return nil
}

// OnOpen is called when connection is opened
func (h *Handler) OnOpen(ctx context.Context, c types.IConnRpc, ch types.IConnInfo) error {
	fmt.Println("OnOpen:", ch.GetUserId(), h.server)
	if h.onPendingHandler != nil {
		return h.onPendingHandler(ctx, c, ch)
	}
	return nil
}
