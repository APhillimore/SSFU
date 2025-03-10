package webrtcnegotiation

import (
	"log"
	"sync"

	"github.com/pion/webrtc/v3"
)

type IWebRTCNegotiator interface {
	ID() string
}

type WebRTCNegotiatorConfig struct {
	ID                         string
	IsPolite                   bool
	HandleSetRemoteDescription func(description webrtc.SessionDescription) error
	HandleSetLocalDescription  func(description webrtc.SessionDescription) error
	HandleAddICECandidate      func(candidate webrtc.ICECandidateInit) error
	HandleSendOffer            func(description webrtc.SessionDescription) error
	HandleCreateOffer          func() (webrtc.SessionDescription, error)
}

type WebRtcNegotiator struct {
	makingOffer                 bool
	makingOfferMu               sync.Mutex
	isPolite                    bool
	id                          string
	handleSetRemoteDescription  func(description webrtc.SessionDescription) error
	handleSetLocalDescription   func(description webrtc.SessionDescription) error
	handleAddICECandidate       func(candidate webrtc.ICECandidateInit) error
	handleCreateOffer           func() (webrtc.SessionDescription, error)
	handleSendRemoteDescription func(description webrtc.SessionDescription) error
}

/*
NewWebRtcNegotiator creates a new WebRTC negotiator.
*/
func NewWebRtcNegotiator(config WebRTCNegotiatorConfig) *WebRtcNegotiator {
	return &WebRtcNegotiator{
		isPolite:                    config.IsPolite,
		makingOffer:                 false,
		id:                          config.ID,
		handleSetRemoteDescription:  config.HandleSetRemoteDescription,
		handleSetLocalDescription:   config.HandleSetLocalDescription,
		handleAddICECandidate:       config.HandleAddICECandidate,
		handleSendRemoteDescription: config.HandleSendOffer,
		handleCreateOffer:           config.HandleCreateOffer,
	}
}

func (n *WebRtcNegotiator) ID() string {
	return n.id
}

/*
HandleOffer handles an offer from a remote peer.
*/
func (n *WebRtcNegotiator) HandleOffer(offer *webrtc.SessionDescription, signalingState webrtc.SignalingState) {
	offerCollision := n.makingOffer || signalingState != webrtc.SignalingStateStable
	ignoreOffer := !n.isPolite && offerCollision
	if ignoreOffer {
		log.Println("Ignoring offer due to collision (impolite peer)")
		// retry message
		return
	}
	if offerCollision {
		log.Println("Offer collision detected: Polite peer waiting for current cycle to complete")
		n.makingOfferMu.Lock()
		n.makingOffer = false
		n.makingOfferMu.Unlock()
		// retry message
		return
	}
	// No collision or we're ready to handle the remote description
	if err := n.handleSetRemoteDescription(*offer); err != nil {
		log.Println("Error setting remote description:", err)
		return
	}
}

/*
HandleAnswer handles an answer from a remote peer.
*/
func (n *WebRtcNegotiator) HandleAnswer(answer *webrtc.SessionDescription) {
	if err := n.handleSetLocalDescription(*answer); err != nil {
		log.Println("Error setting local description:", err)
		n.makingOfferMu.Lock()
		n.makingOffer = false
		n.makingOfferMu.Unlock()
		return
	}
}

/*
HandleCandidate handles an ICE candidate from a remote peer.
*/
func (n *WebRtcNegotiator) HandleCandidate(candidate *webrtc.ICECandidate) {
	if err := n.handleAddICECandidate(candidate.ToJSON()); err != nil {
		log.Println("Error adding ICE candidate:", err)
	}
}

/*
SendOffer sends an offer to a remote peer.
*/
func (n *WebRtcNegotiator) SendOffer() {
	if n.makingOffer {
		log.Println("Already making offer")
		return
	}
	n.makingOfferMu.Lock()
	n.makingOffer = true
	n.makingOfferMu.Unlock()
	offer, err := n.handleCreateOffer()
	if err != nil {
		log.Println("Error creating offer:", err)
		return
	}
	err = n.handleSetLocalDescription(offer)
	if err != nil {
		log.Println("Error setting local description:", err)
		n.makingOfferMu.Lock()
		n.makingOffer = false
		n.makingOfferMu.Unlock()
		return
	}
	err = n.handleSendRemoteDescription(offer)
	if err != nil {
		log.Println("Error sending signal:", err)
		return
	}
}
