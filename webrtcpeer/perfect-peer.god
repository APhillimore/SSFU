package peer

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

type PerfectPeer struct {
	*SfuPeer
	isPolite      bool
	makingOffer   bool
	makingOfferMu sync.Mutex
}

// func NewPerfectPeer(id string, Conn *websocket.Conn, PeerConfig *webrtc.Configuration, isPolite bool) (*PerfectPeer, error) {
// 	sfuPeer, err := NewSfuPeer(id, Conn, PeerConfig)
// 	if err != nil {
// 		log.Println("Error creating sfu peer:", err)
// 		return nil, err
// 	}
// 	return &PerfectPeer{SfuPeer: sfuPeer}, nil
// }

// Initialize WebRTC
// func (p *PerfectPeer) Negotiate() {
// 	// Handle negotiation needed
// 	p.AddOnNegotiationNeededHandler(func() {
// 		log.Println("Negotiation needed")
// 	})

// 	p.AddOnTrackHandler(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
// 		log.Println("onTrack", track.ID())
// 	})

// }

func (p *PerfectPeer) SendOffer() {
	if p.makingOffer {
		log.Println("Already making offer")
		return
	}
	p.makingOfferMu.Lock()
	p.makingOffer = true
	p.makingOfferMu.Unlock()
	offer, err := p.CreateOffer(nil)
	if err != nil {
		log.Println("Error creating offer:", err)
		return
	}
	err = p.SetLocalDescription(offer)
	if err != nil {
		log.Println("Error setting local description:", err)
		return
	}
	p.SendSignal(Signal{Description: &offer})
	p.makingOfferMu.Lock()
	p.makingOffer = false
	p.makingOfferMu.Unlock()
}

func (p *PerfectPeer) HandleDescription(description *webrtc.SessionDescription) {
	log.Printf("%sHandleDescription: %s%s", Green, description.Type, Reset)
	// Check for collision
	offerCollision := description.Type == webrtc.SDPTypeOffer &&
		(p.makingOffer || p.SignalingState() != webrtc.SignalingStateStable)

	ignoreOffer := !p.isPolite && offerCollision
	if ignoreOffer {
		log.Println("Ignoring offer due to collision (impolite peer)")
		p.SendSignal(Signal{Retry: true})
		return
	}

	if offerCollision {
		// For polite peer: we need to roll back our existing offer
		// Since Pion doesn't support rollback, we'll wait for our current
		// offer-answer cycle to complete naturally
		log.Println("Offer collision detected: Polite peer waiting for current cycle to complete")
		p.SendSignal(Signal{Retry: true})
		return
	}

	// No collision or we're ready to handle the remote description
	if err := p.SetRemoteDescription(*description); err != nil {
		log.Println("Error setting remote description:", err)
		return
	}

	if description.Type == webrtc.SDPTypeOffer {
		answer, err := p.CreateAnswer(nil)
		if err != nil {
			log.Println("Error creating answer:", err)
			return
		}
		if err := p.SetLocalDescription(answer); err != nil {
			log.Println("Error setting local description:", err)
			return
		}
		p.SendSignal(Signal{Description: &answer})
	}
}

func (p *PerfectPeer) HandleCandidate(candidate *webrtc.ICECandidateInit) {
	log.Printf("%sHandleCandidate: %s%s", Green, candidate.Candidate, Reset)
	if err := p.AddICECandidate(*candidate); err != nil {
		log.Println("Error adding ICE candidate:", err)
	}
}

// Handle WebRTC signaling messages
func (p *PerfectPeer) HandleMessage(msg []byte) {
	msgStr := string(msg)
	if len(msgStr) > 100 {
		msgStr = msgStr[:100] + "..."
	}
	log.Printf("%sHandleMessage: %s%s", Red, msgStr, Reset)
	// Parse the incoming message (assume JSON format)
	var signal Signal

	if err := json.Unmarshal(msg, &signal); err != nil {
		log.Println("Error parsing message:", err)
		return
	}

	p.makingOfferMu.Lock()
	defer p.makingOfferMu.Unlock()

	if signal.Description != nil {
		p.HandleDescription(signal.Description)
	}

	if signal.Candidate != nil {
		p.HandleCandidate(signal.Candidate)
	}
	if signal.Retry {
		log.Println("Received retry request, will retry offer when stable")
		go p.retryWhenStable()
		return
	}

}
func (p *PerfectPeer) retryWhenStable() {
	// Wait for signaling state to become stable
	for p.SignalingState() != webrtc.SignalingStateStable {
		time.Sleep(100 * time.Millisecond)
	}

	// Small additional delay to ensure both peers are ready
	time.Sleep(500 * time.Millisecond)

	// Trigger a new negotiation
	p.SendOffer()
}

// Send signaling messages
func (p *PerfectPeer) SendSignal(signal Signal) {
	msg, err := json.Marshal(signal)
	if err != nil {
		log.Println("Error marshalling signal:", err)
		return
	}
	s := string(msg)
	if len(s) > 100 {
		s = s[:100] + "..."
	}
	log.Printf("Sending signal: %s", s)

	err = p.Conn.WriteMessage(websocket.TextMessage, msg)
	if err != nil {
		log.Println("Error sending signal:", err)
	}
}
