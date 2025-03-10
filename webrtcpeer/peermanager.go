package webrtcpeer

import (
	"errors"
	"maps"
	"sync"

	"github.com/pion/webrtc/v3"
)

type PeerLike interface {
	ID() string
	SignalingState() webrtc.SignalingState
}

type PeerManager[T PeerLike] struct {
	peers map[string]T
	mu    sync.Mutex
}

func NewPeerManager() *PeerManager[PeerLike] {
	return &PeerManager[PeerLike]{
		peers: make(map[string]PeerLike),
	}
}

func (pm *PeerManager[T]) AddPeer(peer T) (*T, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	peerID := peer.ID()
	if _, ok := pm.peers[peerID]; ok {
		return nil, errors.New("peer already exists")
	}
	pm.peers[peerID] = peer
	return &peer, nil
}

func (pm *PeerManager[T]) RemovePeer(peer T) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if _, ok := pm.peers[peer.ID()]; !ok {
		return errors.New("peer not found")
	}
	delete(pm.peers, peer.ID())
	return nil
}

func (pm *PeerManager[T]) GetPeer(id string) (*T, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	peer, ok := pm.peers[id]
	if !ok {
		return nil, errors.New("peer not found")
	}
	return &peer, nil
}

func (pm *PeerManager[T]) GetPeers() map[string]T {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return maps.Clone(pm.peers)
}

func (pm *PeerManager[T]) CountPeers() int {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return len(pm.peers)
}
