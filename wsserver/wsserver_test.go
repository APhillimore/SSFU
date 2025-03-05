package wsserver_test

import (
	"log"
	"net/http"
	"strings"
	"testing"
	"time"

	"net/http/httptest"

	"simple-forwarding-unit/wsserver"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestWsServer(t *testing.T) {
	t.Run("Can add/remove on message handler", func(t *testing.T) {
		wsServer := wsserver.NewWsManager()
		handler := wsserver.NewWsMessageHandler(func(connection *wsserver.WsConnection, message []byte) error {
			log.Println(string(message))
			return nil
		})
		wsServer.OnMessageHandlers.Add(handler)
		assert.Equal(t, 1, wsServer.OnMessageHandlers.Count())
		wsServer.OnMessageHandlers.Remove(handler)
		assert.Equal(t, 0, wsServer.OnMessageHandlers.Count())
	})
	t.Run("Can add/remove connection", func(t *testing.T) {
		// Create test server
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			upgrader := websocket.Upgrader{}
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()
		}))
		defer s.Close()

		// Convert http://... to ws://...
		wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

		// Connect to test server
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		wsServer := wsserver.NewWsManager()
		wsConnection := wsServer.Connections.Add(wsserver.NewWsConnection(conn))
		assert.Equal(t, 1, wsServer.Connections.Count())
		assert.Equal(t, conn, wsConnection.Conn())

		wsServer.Connections.Remove(wsConnection)
		assert.Equal(t, 0, wsServer.Connections.Count())
	})
	t.Run("Message is sent to all handlers", func(t *testing.T) {

		wsManager := wsserver.NewWsManager()

		called1 := 0
		handler1 := wsserver.NewWsMessageHandler(func(connection *wsserver.WsConnection, message []byte) error {
			log.Println(string(message))
			called1++
			return nil
		})
		wsManager.OnMessageHandlers.Add(handler1)

		called2 := 0
		handler2 := wsserver.NewWsMessageHandler(func(connection *wsserver.WsConnection, message []byte) error {
			log.Println(string(message))
			called2++
			return nil
		})
		wsManager.OnMessageHandlers.Add(handler2)

		// Create test server
		s := httptest.NewServer(http.HandlerFunc(wsManager.WebsocketEndpointHandler))
		defer s.Close()

		// Convert http://... to ws://...
		wsURL := "ws" + strings.TrimPrefix(s.URL, "http")

		// Connect to test server
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		assert.NoError(t, err)
		defer conn.Close()

		conn.WriteMessage(websocket.TextMessage, []byte("hello"))
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 1, called1)
		assert.Equal(t, 1, called2)
		conn.WriteMessage(websocket.TextMessage, []byte("hello"))
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 2, called1)
		assert.Equal(t, 2, called2)
		wsManager.OnMessageHandlers.Remove(handler1)
		conn.WriteMessage(websocket.TextMessage, []byte("hello"))
		time.Sleep(10 * time.Millisecond)
		assert.Equal(t, 2, called1)
		assert.Equal(t, 3, called2)
	})
}
