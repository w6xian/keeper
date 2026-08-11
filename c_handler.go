package keeper

import (
	"context"
	"net/http"

	"github.com/w6xian/sloth/v3/slots"
	"github.com/w6xian/sloth/v3/types"
	"go.uber.org/zap"
)

type Handler struct {
	slots.Client
	ready chan error
	dog   *Dog
}

// OnClose is called when connection is closed
func (h *Handler) OnClose(ctx context.Context, r *http.Response, c types.IConnRpc, ch types.IConnInfo) error {
	h.dog.stopHeartbeat()
	h.dog.mu.Lock()
	h.dog.connected = false
	h.dog.mu.Unlock()
	h.dog.logger.Warn("dog connection closed",
		zap.String("dog", h.dog.Name),
		zap.String("service", h.dog.serviceName),
		zap.String("instanceID", h.dog.instanceID),
	)
	return nil
}

// OnError handles errors
func (h *Handler) OnError(ctx context.Context, r *http.Response, c types.IConnRpc, ch types.IConnInfo, err error) error {
	h.dog.stopHeartbeat()
	h.dog.mu.Lock()
	h.dog.connected = false
	h.dog.mu.Unlock()
	h.dog.logger.Warn("dog connection error",
		zap.String("dog", h.dog.Name),
		zap.String("service", h.dog.serviceName),
		zap.String("instanceID", h.dog.instanceID),
		zap.Error(err),
	)
	return nil
}

// OnOpen is called when connection is opened
func (h *Handler) OnReady(ctx context.Context, r *http.Response, c types.IConnRpc, ch types.IConnInfo) error {
	h.ready <- nil
	return nil
}
