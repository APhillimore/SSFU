package signalling

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
)

/*
Peer is responsible for handling the lifecycle of the relationship between one client and the server
When a track is created for a peer, it is converted to a local track and stored in the LocalTracks.
The local track is then forwarded to other peers in a room.
*/

type Peer struct {
	ID                                 string
	Peer                               *webrtc.PeerConnection
	Conn                               *websocket.Conn
	LocalTracks                        map[string]*webrtc.TrackLocalStaticRTP
	LocalTracksMu                      sync.RWMutex
	OnConnectionStateChangeHandlers    []func(connectionState webrtc.PeerConnectionState)
	OnTrackHandlers                    []func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, localTrack *webrtc.TrackLocalStaticRTP)
	OnICEConnectionStateChangeHandlers []func(connectionState webrtc.ICEConnectionState)
	OnNegotiationNeededHandlers        []func()
	Negotiating                        bool
	// DataChannels      map[string]*webrtc.TrackRemote
	// DataChannelsMu    sync.RWMutex
}

func NewPeer(id string, conn *websocket.Conn) (*Peer, error) {
	config := webrtc.Configuration{
		// ICEServers: []webrtc.ICEServer{
		// 	{
		// 		URLs: []string{
		// 			"stun:stun.l.google.com:19302",
		// 			"stun:stun1.l.google.com:19302",
		// 			"stun:stun2.l.google.com:19302",
		// 		},
		// 	},
		// },
	}
	peerConnection, err := webrtc.NewPeerConnection(config)
	if err != nil {
		return nil, err
	}

	peer := &Peer{
		ID:            id,
		Peer:          peerConnection,
		Conn:          conn,
		LocalTracks:   make(map[string]*webrtc.TrackLocalStaticRTP),
		LocalTracksMu: sync.RWMutex{},

		OnConnectionStateChangeHandlers: make([]func(connectionState webrtc.PeerConnectionState), 0),

		OnTrackHandlers:                    make([]func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, localTrack *webrtc.TrackLocalStaticRTP), 0),
		OnICEConnectionStateChangeHandlers: make([]func(connectionState webrtc.ICEConnectionState), 0),

		OnNegotiationNeededHandlers: make([]func(), 0),
		Negotiating:                 false,
	}

	peerConnection.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		for _, handler := range peer.OnConnectionStateChangeHandlers {
			handler(connectionState)
		}
	})

	peerConnection.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		fmt.Println("New track received:", remoteTrack.ID(), "Kind:", remoteTrack.Kind())

		// Convert the remote track (from the peer) to a local track which can be forwarded to other peers

		trackID := remoteTrack.ID()

		if localTrack, ok := peer.LocalTracks[trackID]; ok {
			log.Println("Track already exists:", trackID)
			for _, handler := range peer.OnTrackHandlers {
				handler(remoteTrack, receiver, localTrack)
			}
			return
		}

		localTrack, err := peer.ConvertTrack(remoteTrack)
		if err != nil {
			log.Println("error converting track:", err)
			return
		}

		for _, handler := range peer.OnTrackHandlers {
			handler(remoteTrack, receiver, localTrack)
		}
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		if candidate == nil {
			return
		}
		log.Println("\033[32m", peer.ID, " ICE candidate:", candidate.ToJSON().Candidate, "\033[0m")
		conn.WriteJSON(CandidateStruct{
			ID:        peer.ID,
			Candidate: candidate.ToJSON(),
			Type:      "candidate",
		})

	})

	peerConnection.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Println(peer.ID, " ICE connection state:", connectionState)
		for _, handler := range peer.OnICEConnectionStateChangeHandlers {
			handler(connectionState)
		}
	})

	peerConnection.OnNegotiationNeeded(func() {
		fmt.Println(peer.ID, " Negotiation needed")
		for _, handler := range peer.OnNegotiationNeededHandlers {
			handler()
		}
	})

	peerConnection.OnSignalingStateChange(func(signalingState webrtc.SignalingState) {
		fmt.Println(peer.ID, " Signaling state:", signalingState)
	})

	for _, receiver := range peerConnection.GetReceivers() {
		if receiver.Track() != nil && receiver.Track().Kind() == webrtc.RTPCodecTypeVideo {
			remoteTrack := receiver.Track()
			trackID := remoteTrack.ID()
			if _, ok := peer.LocalTracks[trackID]; ok {
				log.Println(peer.ID, "Track already exists:", trackID)
				continue
			}
			localTrack, _ := webrtc.NewTrackLocalStaticRTP(remoteTrack.Codec().RTPCodecCapability, trackID, id)
			peer.LocalTracksMu.Lock()
			peer.LocalTracks[trackID] = localTrack
			peer.LocalTracksMu.Unlock()
		}
	}
	return peer, nil
}

func (p *Peer) ConvertTrack(remoteTrack *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error) {
	localTrack, err := webrtc.NewTrackLocalStaticRTP(
		remoteTrack.Codec().RTPCodecCapability, remoteTrack.ID(), p.ID,
	)
	if err != nil {
		log.Println("error creating local track:", err)
		return nil, err
	}
	p.LocalTracksMu.Lock()
	p.LocalTracks[remoteTrack.ID()] = localTrack
	p.LocalTracksMu.Unlock()

	// Start copying packets from the remote track to the local track
	go func(remoteTrack *webrtc.TrackRemote, localTrack *webrtc.TrackLocalStaticRTP) {
		rtpBuf := make([]byte, 1500)
		for {
			i, _, err := remoteTrack.Read(rtpBuf)
			if err != nil {
				log.Println(p.ID, " Error reading from remote track:", err)
				return
			}

			// Remove debug logging and immediately write to local track
			if _, err = localTrack.Write(rtpBuf[:i]); err != nil {
				log.Println(p.ID, " Error writing to local track:", err)
				return
			}
		}
	}(remoteTrack, localTrack)

	return localTrack, nil
}

func (p *Peer) Close() error {
	return p.Peer.Close()
}

func (p *Peer) AddOnConnectionStateChangeHandler(handler func(connectionState webrtc.PeerConnectionState)) {
	p.OnConnectionStateChangeHandlers = append(p.OnConnectionStateChangeHandlers, handler)
}

func (p *Peer) AddOnTrackHandler(handler func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, localTrack *webrtc.TrackLocalStaticRTP)) {
	p.OnTrackHandlers = append(p.OnTrackHandlers, handler)
}

func (p *Peer) AddOnNegotiationNeededHandler(handler func()) {
	p.OnNegotiationNeededHandlers = append(p.OnNegotiationNeededHandlers, handler)
}

func (p *Peer) AddOnICEConnectionStateChangeHandler(handler func(connectionState webrtc.ICEConnectionState)) {
	p.OnICEConnectionStateChangeHandlers = append(p.OnICEConnectionStateChangeHandlers, handler)
}

/* Handles the WebRTC offer from the client */
func (p *Peer) HandleWsOffer(message []byte) error {
	var offer OfferStruct
	err := json.Unmarshal(message, &offer)
	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}
	log.Println("offer recieved from:", offer.ID)

	// Set remote description with error handling
	err = p.Peer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  offer.SDP,
	})
	if err != nil {
		log.Println("error setting remote description:", err)
		return err
	}

	answer, err := p.Peer.CreateAnswer(nil)
	if err != nil {
		log.Println("error creating answer:", err)
		return err
	}
	p.Peer.SetLocalDescription(answer)
	err = p.Conn.WriteJSON(answer)
	if err != nil {
		log.Println("error sending answer:", err)
		return err
	}
	return nil
}

/* Handles the WebRTC answer from the client */
func (p *Peer) HandleWsAnswer(message []byte) error {
	var answer AnswerStruct
	err := json.Unmarshal(message, &answer)
	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}
	log.Println("answer recieved from:", answer.ID)

	// Set remote description with error handling
	err = p.Peer.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeAnswer,
		SDP:  answer.SDP,
	})
	if err != nil {
		log.Println("error setting remote description:", err)
		return err
	}
	return nil
}

/* Handles unhandled messages from the client */
func (p *Peer) HandleWsUnhandled(message []byte) error {
	var msg SignalingMessage
	err := json.Unmarshal(message, &msg)
	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}
	log.Println("unhandled message:", msg)
	return nil
}

/* Handles the ICE candidate from the client */
func (p *Peer) HandleWsCandidate(message []byte) error {
	var candidate CandidateStruct
	err := json.Unmarshal(message, &candidate)
	if err != nil {
		log.Println("unmarshal:", err)
		return err
	}
	log.Println("candidate recieved from:", candidate.ID)
	if candidate.Candidate.Candidate != "" {
		err = p.Peer.AddICECandidate(webrtc.ICECandidateInit{
			Candidate:     candidate.Candidate.Candidate,
			SDPMid:        candidate.Candidate.SDPMid,
			SDPMLineIndex: candidate.Candidate.SDPMLineIndex,
		})
		if err != nil {
			log.Println("Failed to add ICE candidate:", err)
		}
	}
	log.Println("Added ICE candidate for:", candidate.ID)
	return nil
}
