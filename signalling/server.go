package signalling

import (
	"log"
	"net/http"
	"simple-forwarding-unit/peer"
	"simple-forwarding-unit/rooms"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type WebSocketWebRTCSignallingServer struct {
	upgrader             websocket.Upgrader
	connections          map[string]*websocket.Conn
	connectionsMu        sync.Mutex
	peerTypes            map[string]string
	peers                map[string]*peer.PerfectPeer
	peersMu              sync.Mutex
	onNewPeer            map[string]func(peer *webrtc.PeerConnection)
	onNewPeerMu          sync.Mutex
	uninitializedPeers   map[string]*peer.PerfectPeer
	uninitializedPeersMu sync.Mutex
	roomManager          *rooms.RoomManager
}

func NewWebSocketWebRTCSignallingServer() *WebSocketWebRTCSignallingServer {
	return &WebSocketWebRTCSignallingServer{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connections:          make(map[string]*websocket.Conn),
		connectionsMu:        sync.Mutex{},
		peerTypes:            make(map[string]string),
		peers:                make(map[string]*peer.PerfectPeer),
		peersMu:              sync.Mutex{},
		onNewPeer:            make(map[string]func(peer *webrtc.PeerConnection)),
		onNewPeerMu:          sync.Mutex{},
		uninitializedPeers:   make(map[string]*peer.PerfectPeer),
		uninitializedPeersMu: sync.Mutex{},
		roomManager:          rooms.NewRoomManager(),
	}
}

func (s *WebSocketWebRTCSignallingServer) HandleWsSignalling(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	id := r.URL.Query().Get("id")
	ptype := r.URL.Query().Get("type")
	readRooms := r.URL.Query().Get("readRooms")
	writeRooms := r.URL.Query().Get("writeRooms")
	if id == "" {
		id = uuid.New().String()
	}
	s.peerTypes[id] = ptype
	s.connectionsMu.Lock()
	s.connections[id] = conn
	s.connectionsMu.Unlock()

	log.Println("New connection:", id)

	config := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	}

	pc, err := Newpeer.PerfectPeer(id, conn, &config, true)
	if err != nil {
		log.Println("error creating peer connection:", err)
		return
	}

	if readRooms != "" {
		roomNames := strings.Split(readRooms, ",")
		for _, roomName := range roomNames {
			room := s.roomManager.CreateRoom(roomName)
			room.AddMember(rooms.NewRoomViewerMember(id, pc))
		}
	}
	if writeRooms != "" {
		roomNames := strings.Split(writeRooms, ",")
		for _, roomName := range roomNames {
			room := s.roomManager.CreateRoom(roomName)
			room.AddMember(rooms.NewRoomSourceMember(id, pc))
		}
	}

	pc.InitializePeerConnection()

	defer s.lifeCycleCleanup(pc)

	////////////////////////////
	s.peersMu.Lock()
	s.peers[id] = pc
	if ptype == "viewer" {
		s.uninitializedPeersMu.Lock()
		s.uninitializedPeers[id] = pc
		s.uninitializedPeersMu.Unlock()
	}
	////////////////////////////

	pc.AddOnTrackHandler(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Printf("New track received %s", track.ID())
		localTrack, err := pc.ConvertRemoteTrackToLocalTrack(track)
		if err != nil {
			log.Printf("Error converting remote track to local track: %v", err)
			return
		}

		s.peersMu.Lock()
		for id, peer := range s.peers {
			if s.peerTypes[id] == "sourceonly" {
				continue
			}
			if id == pc.ID {
				continue
			}
			peer.AddPeerTrack(localTrack)
			peer.Conn.WriteJSON(map[string]interface{}{
				"type": "track",
				"id":   track.ID(),
			})
		}
		s.peersMu.Unlock()

		// send offer to late joiners
		s.uninitializedPeersMu.Lock()
		for _, peer := range s.uninitializedPeers {
			peer.SendOffer()
		}
		s.uninitializedPeersMu.Unlock()
	})

	pc.AddOnICECandidateHandler(func(candidate *webrtc.ICECandidate) {
		// log.Println("ICE candidate:", candidate)
		if candidate != nil {
			c := candidate.ToJSON()
			pc.SendSignal(Signal{Candidate: &c})
		}
	})
	tracksAdded := false
	log.Println("Peers:", s.peers)
	if ptype != "sourceonly" {
		for id, peer := range s.peers {
			if id == pc.ID {
				continue
			}
			peer.LocalTracksMu.Lock()
			log.Println("Peer:", peer.ID, "Local tracks:", peer.LocalTracks)
			for _, track := range peer.LocalTracks {
				log.Println("Adding track:", track.ID())
				_, err := pc.AddPeerTrack(track)
				if err != nil {
					log.Println("Error adding peer track:", err)
				}
				tracksAdded = true
			}
			peer.LocalTracksMu.Unlock()
		}
	}
	log.Println("Tracks added:", tracksAdded)
	if tracksAdded {
		// Send offer to late joiners
		s.uninitializedPeersMu.Lock()
		if _, ok := s.uninitializedPeers[pc.ID]; ok {
			delete(s.uninitializedPeers, pc.ID)
			pc.SendOffer()
		}
		s.uninitializedPeersMu.Unlock()

	}

	pc.AddOnConnectionStateChangeHandler(func(state webrtc.PeerConnectionState) {
		log.Printf(Red+"Peer %s connection state changed to %v"+Reset, id, state)

		if state == webrtc.PeerConnectionStateDisconnected {
			s.lifeCycleCleanup(pc)
		}
	})

	s.peersMu.Unlock()

	for {
		mt, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("[%s]error reading message: %v", id, err)
			break
		}
		s.uninitializedPeersMu.Lock()
		delete(s.uninitializedPeers, id)
		s.uninitializedPeersMu.Unlock()
		if mt == websocket.TextMessage {
			pc.HandleMessage(message)
		}
		if mt == websocket.CloseMessage {
			pc.Close()
			break
		}
	}

}

func (s *WebSocketWebRTCSignallingServer) Shutdown() {
	s.connectionsMu.Lock()
	defer s.connectionsMu.Unlock()
	for _, conn := range s.connections {
		conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		conn.Close()
	}
}

func (s *WebSocketWebRTCSignallingServer) lifeCycleCleanup(pc *peer.PerfectPeer) {
	log.Println("Cleaning up peer:", pc.ID)
	// remove this peer's tracks from all peers
	s.peersMu.Lock()
	s.uninitializedPeersMu.Lock()

	// tracks := pc.GetMyTrackIDs()
	for _, peer := range s.peers {
		if peer.ID == pc.ID {
			continue
		}
		removedTracks := false
		peer.LocalTracksMu.Lock()
		pc.TrackMapMu.Lock()
		for _, localTrackID := range pc.TrackMap {
			err := peer.RemoveSendingTrack(localTrackID)
			if err != nil {
				log.Println("Error removing track:", err)
				continue
			} else {
				removedTracks = true
			}
			log.Println("Removed track:", localTrackID)
		}
		pc.TrackMapMu.Unlock()
		peer.LocalTracksMu.Unlock()
		if len(peer.GetSenders()) == 0 {
			s.uninitializedPeers[peer.ID] = peer
		} else if removedTracks {
			peer.SendOffer()
		}
	}
	log.Println("Local tracks:", pc.LocalTracks)
	for _, track := range pc.LocalTracks {
		pc.RemoveLocalTrack(track.ID())
	}
	pc.Shutdown()

	// remove peer
	delete(s.peers, pc.ID)
	delete(s.uninitializedPeers, pc.ID)
	delete(s.peerTypes, pc.ID)
	s.uninitializedPeersMu.Unlock()
	s.peersMu.Unlock()
	log.Println("Peer removed:", pc.ID)
}
