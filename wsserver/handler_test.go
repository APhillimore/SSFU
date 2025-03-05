package wsserver

import (
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHandlerList(t *testing.T) {
	t.Run("Can add/remove handlers", func(t *testing.T) {
		list := handlerList[*WsMessageHandler]{}
		handler := NewWsMessageHandler(func(message []byte) {})
		list.Add(handler)
		assert.Equal(t, 1, list.Count())
		handler2 := NewWsMessageHandler(func(message []byte) {})
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
		handler1 := NewWsMessageHandler(func(message []byte) {
			log.Println("handler1")
			called = append(called, "handler1")
		})
		handler2 := NewWsMessageHandler(func(message []byte) {
			log.Println("handler2")
			called = append(called, "handler2")
		})
		list.Add(handler1)
		list.Add(handler2)
		list.Call([]byte("message"))
		assert.Equal(t, []string{"handler1", "handler2"}, called)
	})
	t.Run("Will call handlers in order - WsCloseHandler", func(t *testing.T) {
		list := handlerList[*WsCloseHandler]{}
		called := []string{}
		handler1 := NewWsCloseHandler(func() {
			log.Println("handler1")
			called = append(called, "handler1")
		})
		handler2 := NewWsCloseHandler(func() {
			log.Println("handler2")
			called = append(called, "handler2")
		})
		list.Add(handler1)
		list.Add(handler2)
		list.Call(nil)
		assert.Equal(t, []string{"handler1", "handler2"}, called)
	})
}
