package webrtcpeer

import (
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

func TestPeerManager(t *testing.T) {
	t.Run("Add peer", func(t *testing.T) {
		t.Parallel()
		peer, err := NewSfuPeer("1", &webrtc.Configuration{})
		if err != nil {
			t.Fatal(err)
		}
		pm := NewPeerManager()
		peer2, err := pm.AddPeer(peer)
		assert.Nil(t, err)
		assert.Equal(t, peer.ID(), (*peer2).ID())
		assert.Equal(t, 1, pm.CountPeers())
	})
	t.Run("Remove peer", func(t *testing.T) {
		t.Parallel()
		peer, err := NewSfuPeer("1", &webrtc.Configuration{})
		if err != nil {
			t.Fatal(err)
		}
		pm := NewPeerManager()
		_, err = pm.AddPeer(peer)
		assert.Nil(t, err)
		assert.Equal(t, 1, pm.CountPeers())
		err = pm.RemovePeer(peer)
		assert.Nil(t, err)
		assert.Equal(t, 0, pm.CountPeers())
	})

	t.Run("Get peer", func(t *testing.T) {
		t.Parallel()
		peer, err := NewSfuPeer("example-id", &webrtc.Configuration{})
		if err != nil {
			t.Fatal(err)
		}
		pm := NewPeerManager()
		pm.AddPeer(peer)
		peer2, err := pm.GetPeer("example-id")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, peer.ID(), (*peer2).ID())
	})
	t.Run("Get non-existent peer", func(t *testing.T) {
		t.Parallel()
		pm := NewPeerManager()
		peer, err := pm.GetPeer("1")
		assert.Nil(t, peer)
		assert.Error(t, err)
	})
	t.Run("Get all peers", func(t *testing.T) {
		t.Parallel()
		peer, err := NewSfuPeer("1", &webrtc.Configuration{})
		if err != nil {
			t.Fatal(err)
		}
		peer2, err := NewSfuPeer("2", &webrtc.Configuration{})
		if err != nil {
			t.Fatal(err)
		}
		pm := NewPeerManager()
		pm.AddPeer(peer)
		pm.AddPeer(peer2)
		assert.Equal(t, 2, pm.CountPeers())
		peers := pm.GetPeers()
		assert.Equal(t, 2, len(peers))
		assert.Contains(t, peers, peer.ID())
		assert.Contains(t, peers, peer2.ID())
	})
}
