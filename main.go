package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"

	"simple-forwarding-unit/signalling"
)

var (
	/* Store the tracks from sources*/
	// sources = make(map[string]*webrtc.TrackLocalStaticRTP)
	addr = flag.String("addr", "localhost:8081", "http service address")
	// sourcesMu = sync.Mutex{}
)

type WebSocketServer struct {
	clients   map[*websocket.Conn]bool
	clientsMu sync.Mutex

	peers   map[string]*signalling.Peer
	peersMu sync.Mutex

	upgrader websocket.Upgrader

	rooms   map[string]*signalling.Room
	roomsMu sync.Mutex

	// rooms   map[string]*signalling.Room
	// roomsMu sync.Mutex
}

func NewWebSocketServer() *WebSocketServer {
	return &WebSocketServer{
		clients: make(map[*websocket.Conn]bool),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		peers: make(map[string]*signalling.Peer),
		rooms: make(map[string]*signalling.Room),
	}
}

func (s *WebSocketServer) handleJoinRoom(peer *signalling.Peer, message []byte) {
	var joinRoomMessage signalling.JoinRoomMessage
	err := json.Unmarshal(message, &joinRoomMessage)
	if err != nil {
		log.Println("Error unmarshalling join room message:", err)
		return
	}

	room, ok := s.rooms[joinRoomMessage.Room]
	if !ok {
		room = signalling.NewRoom(joinRoomMessage.Room)
		s.rooms[joinRoomMessage.Room] = room
	}

	room.HandleJoinRoomMessage(peer, message)

}

func (s *WebSocketServer) handlePeerClose(peer *signalling.Peer) {
	s.roomsMu.Lock()
	defer s.roomsMu.Unlock()
	for _, room := range s.rooms {
		room.RemovePeer(peer.ID)
	}
	peer.Close()

	s.peersMu.Lock()
	defer s.peersMu.Unlock()
	delete(s.peers, peer.ID)
}

func (s *WebSocketServer) handleClientClose(conn *websocket.Conn) {
	s.clientsMu.Lock()
	delete(s.clients, conn)
	s.clientsMu.Unlock()
	conn.Close()
}

func (s *WebSocketServer) handleWsSignalling(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}

	defer conn.Close()

	command := &signalling.CommandStruct{
		Data: "Connected to signalling server awaiting offer",
		Type: "signalling_ready",
	}
	err = conn.WriteJSON(command)
	if err != nil {
		log.Println("write:", err)
	}

	// Register client
	s.clientsMu.Lock()
	s.clients[conn] = true
	s.clientsMu.Unlock()

	id := r.URL.Query().Get("id")
	if id == "" {
		id = uuid.New().String()
	}
	log.Println("\033[32mHandling new connection: ", id, "\033[0m")

	peer, err := signalling.NewPeer(id, conn)
	if err != nil {
		log.Println("error creating peer connection:", err)
		return
	}

	peer.AddOnTrackHandler(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, localTrack *webrtc.TrackLocalStaticRTP) {
		log.Println("TEST ADD HANDLER")
	})

	if oldPeer, ok := s.peers[id]; ok {
		log.Println("\033[31mPeer already exists suggesting the client did not close the connection properly", id, "\033[0m")
		s.handlePeerClose(oldPeer)
	}
	s.peersMu.Lock()
	s.peers[id] = peer
	s.peersMu.Unlock()

	// Cleanup on disconnect
	defer func() {
		log.Println("\033[31mClosing connection: ", id, "\033[0m")

		s.handlePeerClose(peer)

		s.handleClientClose(conn)
	}()

	// Handle connection...
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			// Handle disconnection
			s.handlePeerClose(peer)
			return
		}
		if messageType == websocket.CloseMessage {
			s.handlePeerClose(peer)
			s.handleClientClose(conn)
			break
		}
		var msg signalling.SignalingMessage
		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("unmarshal:", err)
			break
		}
		switch msg.Type {
		case "offer":
			peer.HandleWsOffer(message)
		case "answer":
			peer.HandleWsAnswer(message)
		case "candidate":
			peer.HandleWsCandidate(message)
		case "join_room":
			s.handleJoinRoom(peer, message)
		default:
			peer.HandleWsUnhandled(message)
		}
	}
}

func viewerHTMLHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "viewer.html")
}

func sourceHTMLHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "source.html")
}

func main() {
	log.Println("Starting server on", *addr)
	flag.Parse()
	wsServer := NewWebSocketServer()

	// Create a channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Setup your HTTP server
	server := &http.Server{
		Addr: *addr, // adjust port as needed

	}

	http.HandleFunc("/signalling", wsServer.handleWsSignalling)
	http.HandleFunc("/viewer", viewerHTMLHandler)
	http.HandleFunc("/source", sourceHTMLHandler)

	// Start server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal
	<-stop

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close all websocket connections
	wsServer.clientsMu.Lock()
	for client := range wsServer.clients {
		client.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		client.Close()
	}
	wsServer.clientsMu.Unlock()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}
