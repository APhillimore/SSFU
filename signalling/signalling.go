package signalling

import (
	"encoding/json"

	"github.com/pion/webrtc/v3"
)

type MessageType string

const (
	MessageTypeJoinRoom  MessageType = "join-room"
	MessageTypeOffer     MessageType = "offer"
	MessageTypeAnswer    MessageType = "answer"
	MessageTypeCandidate MessageType = "candidate"
)

type JoinRoomStruct struct {
	Type       string `json:"type"`
	RoomID     string `json:"roomId"`
	MemberID   string `json:"memberId"`
	MemberType string `json:"memberType"`
}

var (
	Red   = "\033[31m"
	Green = "\033[32m"
	Reset = "\033[0m"
)

type Signal struct {
	Description *webrtc.SessionDescription `json:"description,omitempty"`
	Candidate   *webrtc.ICECandidateInit   `json:"candidate,omitempty"`
	Retry       bool                       `json:"retry,omitempty"`
}

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
