package signalling

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/pion/logging"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
)

type SfuPeer struct {
	*webrtc.PeerConnection
	ID                                 string
	Conn                               *websocket.Conn
	PeerConfig                         *webrtc.Configuration
	LocalTracks                        map[string]*webrtc.TrackLocalStaticRTP
	LocalTracksMu                      sync.Mutex
	Negotiating                        bool
	OnConnectionStateChangeHandlers    map[string]func(connectionState webrtc.PeerConnectionState)
	OnDataChannelHandlers              map[string]func(dataChannel *webrtc.DataChannel)
	OnICECandidateHandlers             map[string]func(candidate *webrtc.ICECandidate)
	OnICEConnectionStateChangeHandlers map[string]func(connectionState webrtc.ICEConnectionState)
	OnICEGatheringStateChangeHandlers  map[string]func(gatheringState webrtc.ICEGathererState)
	OnNegotiationNeededHandlers        map[string]func()
	OnSignalingStateChangeHandlers     map[string]func(signalingState webrtc.SignalingState)
	OnTrackHandlers                    map[string]func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)
	log                                logging.LeveledLogger
}

func NewSfuPeer(id string, conn *websocket.Conn, PeerConfig *webrtc.Configuration) (*SfuPeer, error) {
	peer, err := webrtc.NewPeerConnection(*PeerConfig)
	if err != nil {
		return nil, err
	}

	return &SfuPeer{
		PeerConnection:                     peer,
		ID:                                 id,
		Conn:                               conn,
		PeerConfig:                         PeerConfig,
		LocalTracks:                        make(map[string]*webrtc.TrackLocalStaticRTP),
		LocalTracksMu:                      sync.Mutex{},
		OnConnectionStateChangeHandlers:    make(map[string]func(connectionState webrtc.PeerConnectionState)),
		OnDataChannelHandlers:              make(map[string]func(dataChannel *webrtc.DataChannel)),
		OnICECandidateHandlers:             make(map[string]func(candidate *webrtc.ICECandidate)),
		OnICEConnectionStateChangeHandlers: make(map[string]func(connectionState webrtc.ICEConnectionState)),
		OnICEGatheringStateChangeHandlers:  make(map[string]func(gatheringState webrtc.ICEGathererState)),
		OnNegotiationNeededHandlers:        make(map[string]func()),
		OnSignalingStateChangeHandlers:     make(map[string]func(signalingState webrtc.SignalingState)),
		OnTrackHandlers:                    make(map[string]func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)),
		log:                                logging.NewDefaultLoggerFactory().NewLogger("sfu-peer-" + id),
	}, nil
}

func (p *SfuPeer) RecreatePeerConnection() (*webrtc.PeerConnection, error) {
	peer, err := webrtc.NewPeerConnection(*p.PeerConfig)
	if err != nil {
		return nil, err
	}
	p.PeerConnection = peer
	p.InitializePeerConnection()
	return peer, nil
}

func (p *SfuPeer) InitializePeerConnection() {

	p.OnConnectionStateChange(func(connectionState webrtc.PeerConnectionState) {
		for _, handler := range p.OnConnectionStateChangeHandlers {
			handler(connectionState)
		}
	})

	p.OnDataChannel(func(dataChannel *webrtc.DataChannel) {
		for _, handler := range p.OnDataChannelHandlers {
			handler(dataChannel)
		}
	})

	p.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		for _, handler := range p.OnICECandidateHandlers {
			handler(candidate)
		}
	})

	p.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		for _, handler := range p.OnICEConnectionStateChangeHandlers {
			handler(connectionState)
		}
	})

	p.OnICEGatheringStateChange(func(gatheringState webrtc.ICEGathererState) {
		for _, handler := range p.OnICEGatheringStateChangeHandlers {
			handler(gatheringState)
		}
	})

	p.OnNegotiationNeeded(func() {
		for _, handler := range p.OnNegotiationNeededHandlers {
			handler()
		}
	})

	p.OnSignalingStateChange(func(signalingState webrtc.SignalingState) {
		for _, handler := range p.OnSignalingStateChangeHandlers {
			handler(signalingState)
		}
	})

	p.OnTrack(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		for _, handler := range p.OnTrackHandlers {
			handler(remoteTrack, receiver)
		}
	})

}

func (p *SfuPeer) AddOnConnectionStateChangeHandler(handler func(connectionState webrtc.PeerConnectionState)) string {
	id := uuid.New().String()
	p.OnConnectionStateChangeHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnConnectionStateChangeHandler(id string) {
	delete(p.OnConnectionStateChangeHandlers, id)
}

func (p *SfuPeer) AddOnDataChannelHandler(handler func(dataChannel *webrtc.DataChannel)) string {
	id := uuid.New().String()
	p.OnDataChannelHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnDataChannelHandler(id string) {
	delete(p.OnDataChannelHandlers, id)
}

func (p *SfuPeer) AddOnICECandidateHandler(handler func(candidate *webrtc.ICECandidate)) string {
	id := uuid.New().String()
	p.OnICECandidateHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnICECandidateHandler(id string) {
	delete(p.OnICECandidateHandlers, id)
}

func (p *SfuPeer) AddOnICEConnectionStateChangeHandler(handler func(connectionState webrtc.ICEConnectionState)) string {
	id := uuid.New().String()
	p.OnICEConnectionStateChangeHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnICEConnectionStateChangeHandler(id string) {
	delete(p.OnICEConnectionStateChangeHandlers, id)
}

func (p *SfuPeer) AddOnICEGatheringStateChangeHandler(handler func(gatheringState webrtc.ICEGathererState)) string {
	id := uuid.New().String()
	p.OnICEGatheringStateChangeHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnICEGatheringStateChangeHandler(id string) {
	delete(p.OnICEGatheringStateChangeHandlers, id)
}

func (p *SfuPeer) AddOnNegotiationNeededHandler(handler func()) string {
	id := uuid.New().String()
	p.OnNegotiationNeededHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnNegotiationNeededHandler(id string) {
	delete(p.OnNegotiationNeededHandlers, id)
}

func (p *SfuPeer) AddOnSignalingStateChangeHandler(handler func(signalingState webrtc.SignalingState)) string {
	id := uuid.New().String()
	p.OnSignalingStateChangeHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnSignalingStateChangeHandler(id string) {
	delete(p.OnSignalingStateChangeHandlers, id)
}

func (p *SfuPeer) AddOnTrackHandler(handler func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver)) string {
	id := uuid.New().String()
	p.OnTrackHandlers[id] = handler
	return id
}

func (p *SfuPeer) RemoveOnTrackHandler(id string) {
	delete(p.OnTrackHandlers, id)
}

func (p *SfuPeer) GetMyTrackIDs() []string {
	receivers := p.GetReceivers()
	trackIDs := make([]string, 0, len(receivers))
	for _, r := range receivers {
		if r.Track() != nil {
			trackIDs = append(trackIDs, r.Track().ID())
		}
	}
	return trackIDs
}

func (p *SfuPeer) IsMyTrack(trackID string) bool {
	return slices.Contains(p.GetMyTrackIDs(), trackID)
}

func (p *SfuPeer) GetOthersTrackIDs() []string {
	sender := p.GetSenders()
	trackIDs := make([]string, 0, len(sender))
	for _, s := range sender {
		if s.Track() != nil {
			trackIDs = append(trackIDs, s.Track().ID())
		}
	}
	return trackIDs
}

func (p *SfuPeer) IsAlreadySendingTrack(trackID string) bool {
	return slices.Contains(p.GetOthersTrackIDs(), trackID)
}

func (p *SfuPeer) GetTrackByID(trackID string) *webrtc.RTPSender {
	for _, s := range p.GetSenders() {
		if s.Track() != nil && s.Track().ID() == trackID {
			return s
		}
	}
	return nil
}
func (p *SfuPeer) RemoveSendingTrack(trackID string) error {
	if p.IsAlreadySendingTrack(trackID) {
		track := p.GetTrackByID(trackID)
		if track != nil {
			err := p.RemoveTrack(track)
			if err != nil {
				return err
			}
			return nil
		}
	}
	return fmt.Errorf("track %s is not being sent by this peer", trackID)
}

/* Takes a remote track and converts it to a local track. Renegotiation for all peers should be fired following this. */
func (p *SfuPeer) ConvertRemoteTrackToLocalTrack(remoteTrack *webrtc.TrackRemote) (*webrtc.TrackLocalStaticRTP, error) {

	removeTrackID := remoteTrack.ID()

	if !p.IsMyTrack(removeTrackID) {
		p.log.Warnf("This track does not belong to this peer %s", removeTrackID)
		return nil, fmt.Errorf("this track does not belong to this peer %s", removeTrackID)
	}

	codecCap := remoteTrack.Codec().RTPCodecCapability
	codecCap.RTCPFeedback = nil // Clear RTCP feedback to avoid compatibility issues
	localTrack, err := webrtc.NewTrackLocalStaticRTP(codecCap, remoteTrack.ID(), p.ID)

	p.LocalTracksMu.Lock()
	p.LocalTracks[removeTrackID] = localTrack
	p.LocalTracksMu.Unlock()

	// Start copying packets from the remote track to the local track
	go func(remoteTrack *webrtc.TrackRemote, localTrack *webrtc.TrackLocalStaticRTP) {
		rtpBuf := make([]byte, 1500)
		p.log.Infof("Copying packets from remote track to local track: %s", removeTrackID)
		for {
			i, _, err := remoteTrack.Read(rtpBuf)
			if err != nil {
				p.log.Errorf(" Error reading from remote track: %s", err)
				return
			}

			// Remove debug logging and immediately write to local track
			if _, err = localTrack.Write(rtpBuf[:i]); err != nil {
				p.log.Errorf(" Error writing to local track: %s", err)
				return
			}
		}
	}(remoteTrack, localTrack)

	// TODO: Blindly sending keyframes helps connect a late viewer but it's not a good long term solution
	go func() {
		for {
			time.Sleep(time.Second * 3)
			err := p.WriteRTCP([]rtcp.Packet{
				&rtcp.PictureLossIndication{
					MediaSSRC: uint32(remoteTrack.SSRC()),
				},
			})
			if err != nil {
				p.log.Errorf("Error sending PLI: %v", err)
				return
			}
		}
	}()

	return localTrack, err
}

func (p *SfuPeer) RemoveLocalTrack(trackID string) {
	p.LocalTracksMu.Lock()
	delete(p.LocalTracks, trackID)
	p.LocalTracksMu.Unlock()
}

func (p *SfuPeer) AddPeerTrack(track *webrtc.TrackLocalStaticRTP) (*webrtc.RTPSender, error) {
	trackID := track.ID()
	if p.IsMyTrack(trackID) {
		p.log.Warnf("This track belongs to this peer %s", trackID)
		return nil, fmt.Errorf("this track belongs to this peer %s", trackID)
	}
	if p.IsAlreadySendingTrack(trackID) {
		p.log.Warnf("This track is already being sent by this peer %s", trackID)
		return nil, fmt.Errorf("this track is already being sent by this peer %s", trackID)
	}
	return p.AddTrack(track)
}
