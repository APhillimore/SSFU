package wsserver

import (
	"errors"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/google/uuid"
)

type Handler interface {
	ID() string
	Call(args ...any) error
}

type WsHandler struct {
	id string
}

func (h *WsHandler) ID() string {
	return h.id
}

type WsMessageHandler struct {
	WsHandler
	handler func(connection *WsConnection, message []byte) error
}

func NewWsMessageHandler(handlerFunction func(connection *WsConnection, message []byte) error) *WsMessageHandler {
	return &WsMessageHandler{
		WsHandler: WsHandler{
			id: uuid.New().String(),
		},
		handler: handlerFunction,
	}
}

func (h *WsMessageHandler) Call(args ...any) error {
	if len(args) != 2 {
		return errors.New("expected 2 argument, got " + strconv.Itoa(len(args)))
	}
	connection, ok := args[0].(*WsConnection)
	if !ok {
		return errors.New("expected *WsConnection, got " + reflect.TypeOf(args[0]).String())
	}
	message, ok := args[1].([]byte)
	if !ok {
		return errors.New("expected []byte, got " + reflect.TypeOf(args[1]).String())
	}
	h.handler(connection, message)
	return nil
}

type WsCloseHandler struct {
	WsHandler
	handler func(connection *WsConnection) error
}

func NewWsCloseHandler(handlerFunction func(connection *WsConnection) error) *WsCloseHandler {
	return &WsCloseHandler{
		WsHandler: WsHandler{
			id: uuid.New().String(),
		},
		handler: handlerFunction,
	}
}

func (h *WsCloseHandler) Call(args ...any) error {
	if len(args) != 1 {
		return errors.New("expected 1 argument, got " + strconv.Itoa(len(args)))
	}
	connection, ok := args[0].(*WsConnection)
	if !ok {
		return errors.New("expected *WsConnection, got " + reflect.TypeOf(args[0]).String())
	}
	h.handler(connection)
	return nil
}

type WsConnectionHandler struct {
	WsHandler
	handler func(connection *WsConnection) error
}

func NewWsConnectionHandler(handlerFunction func(connection *WsConnection) error) *WsConnectionHandler {
	return &WsConnectionHandler{
		WsHandler: WsHandler{
			id: uuid.New().String(),
		},
		handler: handlerFunction,
	}
}

func (h *WsConnectionHandler) Call(args ...any) error {
	if len(args) != 1 {
		return errors.New("expected 1 argument, got " + strconv.Itoa(len(args)))
	}
	connection, ok := args[0].(*WsConnection)
	if !ok {
		return errors.New("expected *WsConnection, got " + reflect.TypeOf(args[0]).String())
	}
	h.handler(connection)
	return nil
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

func (hl *handlerList[T]) Call(args ...any) error {
	e := make([]string, 0)
	hl.mu.RLock()
	defer hl.mu.RUnlock()
	for _, h := range hl.handlers {
		err := h.Call(args...)
		if err != nil {
			e = append(e, err.Error())
		}
	}
	if len(e) > 0 {
		return errors.New(strings.Join(e, ",\n"))
	}
	return nil
}
