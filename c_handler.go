package keeper

import (
	"context"
	"log"
	"net/http"

	"github.com/w6xian/sloth/v3/slots"
	"github.com/w6xian/sloth/v3/types"
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
	log.Printf("[%s] Dog connection %s closed\n", h.dog.Name, h.dog.serviceName)
	return nil
}

// OnError handles errors
func (h *Handler) OnError(ctx context.Context, r *http.Response, c types.IConnRpc, ch types.IConnInfo, err error) error {
	h.dog.stopHeartbeat()
	h.dog.mu.Lock()
	h.dog.connected = false
	h.dog.mu.Unlock()
	log.Printf("[%s] Dog connection %s error: %v\n", h.dog.Name, h.dog.serviceName, err)
	return nil
}

// OnOpen is called when connection is opened
func (h *Handler) OnReady(ctx context.Context, r *http.Response, c types.IConnRpc, ch types.IConnInfo) error {
	h.ready <- nil
	return nil
}
