package signalling

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

type CommandStruct struct {
	Data string `json:"data"`
	Type string `json:"type"`
}

type OfferStruct struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
	ID   string `json:"id"` // Source ID
}

type CandidateStruct struct {
	Candidate webrtc.ICECandidateInit `json:"candidate"`
	Type      string                  `json:"type"`
	ID        string                  `json:"id"` // Source ID
}

type AnswerStruct struct {
	SDP  string `json:"sdp"`
	Type string `json:"type"`
	ID   string `json:"id"` // Source ID
}

type SignalingMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type SessionDescription struct {
	Type string `json:"type"` // "offer", "answer", or "pranswer"
	SDP  string `json:"sdp"`  // SDP payload
}

type IceCandidate struct {
	Candidate        string `json:"candidate"`
	SDPMid           string `json:"sdpMid"`
	SDPMLineIndex    uint16 `json:"sdpMLineIndex"`
	UsernameFragment string `json:"usernameFragment,omitempty"`
}

// type Signalling struct {
// 	peers map[string]*webrtc.PeerConnection
// }

// func NewSignalling() *Signalling {
// 	return &Signalling{}
// }

// func (s *Signalling) AddPeer(id string, peer *webrtc.PeerConnection) {
// 	s.peers[id] = peer
// }

// func (s *Signalling) RemovePeer(id string) {
// 	delete(s.peers, id)
// }

// func (s *Signalling) GetPeer(id string) *Peer {
// 	return s.peers[id]
// }
