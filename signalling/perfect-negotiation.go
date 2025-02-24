package signalling

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type PerfectNegotiation struct {
	signalServer  *websocket.Conn
	peerConn      *webrtc.PeerConnection
	isPolite      bool
	makingOffer   bool
	makingOfferMu sync.Mutex
}

func NewPerfectNegotiation(peerConn *webrtc.PeerConnection, signalServer *websocket.Conn, isPolite bool) *PerfectNegotiation {
	return &PerfectNegotiation{signalServer: signalServer, isPolite: isPolite, peerConn: peerConn}
}

// Initialize WebRTC
func (p *PerfectNegotiation) Negotiate(pc *webrtc.PeerConnection) *webrtc.PeerConnection {
	// WebRTC configuration
	// config := webrtc.Configuration{
	// 	ICEServers: []webrtc.ICEServer{
	// 		{URLs: []string{"stun:stun.l.google.com:19302"}},
	// 	},
	// }

	// Create Peer Connection
	// pc, err := webrtc.NewPeerConnection(config)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Handle negotiation needed
	p.peerConn.OnNegotiationNeeded(func() {
		p.makingOfferMu.Lock()
		p.makingOffer = true
		p.makingOfferMu.Unlock()

		offer, err := p.peerConn.CreateOffer(nil)
		if err != nil {
			log.Println("Error creating offer:", err)
			return
		}

		err = p.peerConn.SetLocalDescription(offer)
		if err != nil {
			log.Println("Error setting local description:", err)
			return
		}

		p.sendSignal(offer)
		p.makingOfferMu.Lock()
		p.makingOffer = false
		p.makingOfferMu.Unlock()
	})

	// Handle ICE candidates
	p.peerConn.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate != nil {
			p.sendSignal(candidate.ToJSON())
		}
	})

	return p.peerConn
}

// Handle incoming messages from the signaling server
func (p *PerfectNegotiation) HandleSignalingMessages() {
	for {
		_, message, err := p.signalServer.ReadMessage()
		if err != nil {
			log.Println("Error reading WebSocket message:", err)
			break
		}

		p.handleMessage(message)
	}
}

// Handle WebRTC signaling messages
func (p *PerfectNegotiation) handleMessage(msg []byte) {
	// Parse the incoming message (assume JSON format)
	var message struct {
		Type      string                   `json:"type"`
		SDP       string                   `json:"sdp,omitempty"`
		Candidate *webrtc.ICECandidateInit `json:"candidate,omitempty"`
	}

	if err := json.Unmarshal(msg, &message); err != nil {
		log.Println("Error parsing message:", err)
		return
	}

	p.makingOfferMu.Lock()
	defer p.makingOfferMu.Unlock()

	switch message.Type {
	case "offer":
		offerCollision := p.makingOffer || p.peerConn.SignalingState() != webrtc.SignalingStateStable

		if offerCollision {
			if p.isPolite {
				log.Println("Offer collision detected: Polite peer defers to remote offer.")
				p.makingOffer = false
				if err := p.peerConn.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: message.SDP}); err != nil {
					log.Println("Error setting remote description:", err)
					return
				}
				answer, err := p.peerConn.CreateAnswer(nil)
				if err != nil {
					log.Println("Error creating answer:", err)
					return
				}
				if err := p.peerConn.SetLocalDescription(answer); err != nil {
					log.Println("Error setting local description:", err)
					return
				}
				p.sendSignal(answer)
			} else {
				log.Println("Offer collision detected: Impolite peer ignores and proceeds.")
			}
		} else {
			if err := p.peerConn.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: message.SDP}); err != nil {
				log.Println("Error setting remote description:", err)
				return
			}
			answer, err := p.peerConn.CreateAnswer(nil)
			if err != nil {
				log.Println("Error creating answer:", err)
				return
			}
			if err := p.peerConn.SetLocalDescription(answer); err != nil {
				log.Println("Error setting local description:", err)
				return
			}
			p.sendSignal(answer)
		}

	case "answer":
		if err := p.peerConn.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: message.SDP}); err != nil {
			log.Println("Error setting remote description:", err)
		}

	case "candidate":
		if message.Candidate != nil {
			if err := p.peerConn.AddICECandidate(*message.Candidate); err != nil {
				log.Println("Error adding ICE candidate:", err)
			}
		}
	}
}

// Send signaling messages
func (p *PerfectNegotiation) sendSignal(data interface{}) {
	msg, err := json.Marshal(data)
	if err != nil {
		log.Println("Error marshalling signal:", err)
		return
	}

	err = p.signalServer.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("Error sending signal:", err)
	}
}

// Connect to the WebSocket signaling server
// func connectToSignalingServer() {
// 	var err error
// 	signalSocket, _, err = websocket.DefaultDialer.Dial("wss://your-signaling-server.com", nil)
// 	if err != nil {
// 		log.Fatal("Error connecting to signaling server:", err)
// 	}
// 	log.Println("Connected to signaling server")

// 	go handleSignalingMessages()
// }

// func main() {
// 	log.Println("Starting WebRTC Peer...")
// 	connectToSignalingServer()
// 	peerConn = createPeerConnection()

// 	select {} // Keep running
// }
