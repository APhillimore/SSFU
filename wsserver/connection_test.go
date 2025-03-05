package wsserver

import (
	"testing"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWsConnection(t *testing.T) {
	t.Run("Can add/remove connections", func(t *testing.T) {
		conn := &websocket.Conn{}
		wsConnection1 := NewWsConnection(conn)
		wsConnection2 := NewWsConnection(conn)
		list := connectionList[*WsConnection]{}
		list.Add(wsConnection1)
		assert.Equal(t, 1, list.Count())
		list.Add(wsConnection2)
		assert.Equal(t, 2, list.Count())
		list.Remove(wsConnection1)
		assert.Equal(t, 1, list.Count())
		list.Remove(wsConnection2)
		assert.Equal(t, 0, list.Count())
	})
}
