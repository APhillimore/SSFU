package wsserver

import (
	"log"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestHandlerList(t *testing.T) {
	t.Run("Can add/remove handlers", func(t *testing.T) {
		list := handlerList[*WsMessageHandler]{}
		handler := NewWsMessageHandler(func(connection *WsConnection, message []byte) error { return nil })
		list.Add(handler)
		assert.Equal(t, 1, list.Count())
		handler2 := NewWsMessageHandler(func(connection *WsConnection, message []byte) error { return nil })
		list.Add(handler2)
		assert.Equal(t, 2, list.Count())
		list.Remove(handler)
		assert.Equal(t, 1, list.Count())
		list.Remove(handler2)
		assert.Equal(t, 0, list.Count())
	})
	t.Run("Will call handlers in order - WsMessageHandler", func(t *testing.T) {
		list := handlerList[*WsMessageHandler]{}
		called := []string{}
		handler1 := NewWsMessageHandler(func(connection *WsConnection, message []byte) error {
			log.Println("handler1")
			called = append(called, "handler1")
			return nil
		})
		handler2 := NewWsMessageHandler(func(connection *WsConnection, message []byte) error {
			log.Println("handler2")
			called = append(called, "handler2")
			return nil
		})
		list.Add(handler1)
		list.Add(handler2)
		err := list.Call(NewWsConnection(&websocket.Conn{}), []byte("message"))
		assert.Nil(t, err)
		assert.Equal(t, []string{"handler1", "handler2"}, called)
	})
	t.Run("Will call handlers in order - WsCloseHandler", func(t *testing.T) {
		list := handlerList[*WsCloseHandler]{}
		called := []string{}
		handler1 := NewWsCloseHandler(func(connection *WsConnection) error {
			log.Println("handler1")
			called = append(called, "handler1")
			return nil
		})
		handler2 := NewWsCloseHandler(func(connection *WsConnection) error {
			log.Println("handler2")
			called = append(called, "handler2")
			return nil
		})
		list.Add(handler1)
		list.Add(handler2)
		err := list.Call(NewWsConnection(&websocket.Conn{}))
		assert.Nil(t, err)
		assert.Equal(t, []string{"handler1", "handler2"}, called)
	})
}
