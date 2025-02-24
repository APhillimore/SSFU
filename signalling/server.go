package signalling

import (
	"log"
	"net/http"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebSocketServer struct {
	upgrader      websocket.Upgrader
	connections   map[string]*websocket.Conn
	connectionsMu sync.Mutex
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (s *WebSocketServer) HandleWsSignalling(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	id := r.URL.Query().Get("id")
	if id == "" {
		id = uuid.New().String()
	}
	s.connectionsMu.Lock()
	s.connections[id] = conn
	s.connectionsMu.Unlock()

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	// Create Peer Connection
	pc, err := webrtc.NewPeerConnection(config)
	if err != nil {
		log.Println("error creating peer connection:", err)
		return
	}

	NewPerfectNegotiation(pc, conn, true)

}

func (s *WebSocketServer) Shutdown() {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()
	for _, conn := range s.connections {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}
}
