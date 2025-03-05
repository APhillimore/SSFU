package wsserver

import (
	"slices"
	"sync"

	"github.com/google/uuid"
)

type Handler interface {
	ID() string
	Call(args any)
}

type WsHandler struct {
	id string
}

func (h *WsHandler) ID() string {
	return h.id
}

type WsMessageHandler struct {
	WsHandler
	handler func(message []byte)
}

func NewWsMessageHandler(handlerFunction func(message []byte)) *WsMessageHandler {
	return &WsMessageHandler{
		WsHandler: WsHandler{
			id: uuid.New().String(),
		},
		handler: handlerFunction,
	}
}

func (h *WsMessageHandler) Call(args any) {
	if message, ok := args.([]byte); ok {
		h.handler(message)
	}
}

type WsCloseHandler struct {
	WsHandler
	handler func()
}

func NewWsCloseHandler(handlerFunction func()) *WsCloseHandler {
	return &WsCloseHandler{
		WsHandler: WsHandler{
			id: uuid.New().String(),
		},
		handler: handlerFunction,
	}
}

func (h *WsCloseHandler) Call(args any) {
	if args == nil {
		h.handler()
	}
}

type handlerList[T Handler] struct {
	handlers []T
	mu       sync.RWMutex
}

func (hl *handlerList[T]) Add(handler T) T {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	hl.handlers = append(hl.handlers, handler)
	return handler
}

func (hl *handlerList[T]) Remove(handler T) bool {
	hl.mu.Lock()
	defer hl.mu.Unlock()
	originalLen := len(hl.handlers)
	hl.handlers = slices.DeleteFunc(hl.handlers, func(h T) bool {
		return h.ID() == handler.ID()
	})
	return len(hl.handlers) < originalLen
}

func (hl *handlerList[T]) Count() int {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	return len(hl.handlers)
}

func (hl *handlerList[T]) Call(args any) {
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	for _, h := range hl.handlers {
		h.Call(args)
	}
}
