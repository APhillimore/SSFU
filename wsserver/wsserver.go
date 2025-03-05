package wsserver

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WsManager struct {
	upgrader          websocket.Upgrader
	Connections       connectionList[*WsConnection]
	OnMessageHandlers handlerList[*WsMessageHandler]
	OnCloseHandlers   handlerList[*WsCloseHandler]
}

func NewWsManager() *WsManager {
	return &WsManager{
		upgrader:          websocket.Upgrader{},
		Connections:       connectionList[*WsConnection]{},
		OnMessageHandlers: handlerList[*WsMessageHandler]{},
		OnCloseHandlers:   handlerList[*WsCloseHandler]{},
	}
}

func (s *WsManager) WebsocketEndpointHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("error upgrading to websocket:", err)
		return
	}
	connection := s.Connections.Add(NewWsConnection(conn))
	defer s.Connections.Remove(connection)
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[%s]error reading message: %v", connection.ID(), err)
			break
		}

		if messageType == websocket.TextMessage {
			s.OnMessageHandlers.Call(message)
		}
		if messageType == websocket.CloseMessage {
			// shutdown connection
			break
		}
	}
}

// Add ID getter methods to existing handler types
func (h *WsMessageHandler) GetID() string {
	return h.id
}

func (h *WsCloseHandler) GetID() string {
	return h.id
}
