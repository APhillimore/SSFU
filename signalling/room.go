// A room is a collection of tracks from sources
package signalling

import (
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/pion/webrtc/v3"
)

type RoomPeerType string

const (
	RoomPeerTypeViewer RoomPeerType = "viewer"
	RoomPeerTypeSource RoomPeerType = "source"
	RoomPeerTypeBoth   RoomPeerType = "both"
)

type RoomPeers struct {
	ID   string
	Peer *Peer
	Type RoomPeerType
}

type Room struct {
	ID string
	// Tracks   map[string]*webrtc.TrackRemote
	// TracksMu sync.Mutex
	Peers   map[string]*RoomPeers
	PeersMu sync.Mutex
}

func NewRoom(id string) *Room {
	return &Room{
		ID: id,
		// Tracks:   make(map[string]*webrtc.TrackRemote),
		// TracksMu: sync.Mutex{},
		Peers:   make(map[string]*RoomPeers),
		PeersMu: sync.Mutex{},
	}
}

func (r *Room) AddPeer(peer *Peer, peerType RoomPeerType) {
	r.PeersMu.Lock()
	r.Peers[peer.ID] = &RoomPeers{
		ID:   peer.ID,
		Type: peerType,
		Peer: peer,
	}
	r.PeersMu.Unlock()
	log.Println("\033[32mAdded peer ", peer.ID, " to room:", r.ID, " type:", peerType, "\033[0m")

	// Set up track handling for sources
	if peerType == RoomPeerTypeSource || peerType == RoomPeerTypeBoth {
		peer.AddOnTrackHandler(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, localTrack *webrtc.TrackLocalStaticRTP) {
			r.PeersMu.Lock()
			defer r.PeersMu.Unlock()

			log.Println("\033[32mOnTrack: Adding track:", localTrack.ID(), " from source:", peer.ID, " to room:", r.ID, "\033[0m")

			// Forward new tracks to all existing viewers
			for peerId, roomPeer := range r.Peers {
				if peerId == peer.ID {
					continue
				}
				// Forward to all peers that can receive (viewers or both)
				if roomPeer.Type == RoomPeerTypeViewer || roomPeer.Type == RoomPeerTypeBoth {
					roomPeer.Peer.Peer.AddTrack(localTrack)
				}
			}
		})
		if peerType == RoomPeerTypeSource || peerType == RoomPeerTypeBoth {
			log.Println("\033[32mAdding tracks from peer:", peer.ID, peer.LocalTracks, " Peers:", r.Peers, "\033[0m")
			for _, roomPeer := range r.Peers {
				if roomPeer.ID == peer.ID || roomPeer.Type == RoomPeerTypeSource {
					continue
				}
				// TODO are you potentially reflecting the same track to the same peer?
				for _, track := range peer.LocalTracks {
					if _, err := roomPeer.Peer.Peer.AddTrack(track); err != nil {
						log.Printf("Error adding track to peer %s: %v", roomPeer.ID, err)
						continue
					}
				}
				// Trigger renegotiation
				log.Println("\033[32mTriggering renegotiation for peer:", roomPeer.ID, "\033[0m")
				offer, err := roomPeer.Peer.Peer.CreateOffer(nil)
				if err != nil {
					log.Printf("Error creating offer for peer %s: %v", roomPeer.ID, err)
					continue
				}
				roomPeer.Peer.Peer.SetLocalDescription(offer)
				roomPeer.Peer.Conn.WriteJSON(OfferStruct{
					SDP:  offer.SDP,
					Type: "offer",
					ID:   peer.ID,
				})
			}
		}
	}

	// Forward existing tracks to new peers

	// When adding a new viewer to the room
	if peerType == RoomPeerTypeViewer || peerType == RoomPeerTypeBoth {
		peer.AddOnNegotiationNeededHandler(func() {
			log.Printf("\033[32mNegotiation needed for peer: %s\033[0m", peer.ID)

			if peer.Negotiating {
				log.Printf("Skipping negotiation - already in progress for peer: %s", peer.ID)
				return
			}

			peer.Negotiating = true
			defer func() {
				time.Sleep(time.Second * 1)
				peer.Negotiating = false
			}()

			// Check if we're stable before starting negotiation
			if peer.Peer.SignalingState() != webrtc.SignalingStateStable {
				log.Printf("Skipping negotiation - signaling state not stable: %s", peer.Peer.SignalingState())
				return
			}

			offer, err := peer.Peer.CreateOffer(nil)
			if err != nil {
				log.Printf("Error creating offer: %v", err)
				return
			}

			err = peer.Peer.SetLocalDescription(offer)
			if err != nil {
				log.Printf("Error setting local description: %v", err)
				return
			}

			peer.Conn.WriteJSON(OfferStruct{
				SDP:  offer.SDP,
				Type: "offer",
				ID:   peer.ID,
			})
		})
		peer.AddOnConnectionStateChangeHandler(func(state webrtc.PeerConnectionState) {
			log.Printf("\033[32mViewer %s connection state changed to %s\033[0m", peer.ID, state)

			if state == webrtc.PeerConnectionStateConnected {
				time.Sleep(time.Millisecond * 500)

				r.PeersMu.Lock()
				defer r.PeersMu.Unlock()

				// Clear any existing tracks first
				// senders := peer.Peer.GetSenders()
				// for _, sender := range senders {
				// 	if err := peer.Peer.RemoveTrack(sender); err != nil {
				// 		log.Printf("Error removing old track: %v", err)
				// 	}
				// }

				// Disable automatic negotiation temporarily
				peer.Peer.OnNegotiationNeeded(func() {
					// Do nothing - we'll handle negotiation manually
				})

				tracksAdded := false
				// Batch add all tracks
				for _, sourcePeer := range r.Peers {
					if sourcePeer.Type == RoomPeerTypeSource {
						for _, track := range sourcePeer.Peer.LocalTracks {
							log.Printf("\033[32mForwarding existing track to %s from %s track: %s\033[0m",
								peer.ID, sourcePeer.ID, track.ID())

							// Check if the track already exists
							existing := peer.Peer.GetSenders()
							for _, sender := range existing {
								if sender.Track() != nil && sender.Track().ID() == track.ID() {
									continue
								}
							}

							if _, err := peer.Peer.AddTrack(track); err != nil {
								log.Printf("Error adding track: %v", err)
								continue
							}
							tracksAdded = true
						}
					}
				}

				// Only negotiate if we actually added tracks
				if tracksAdded {
					if peer.Peer.SignalingState() == webrtc.SignalingStateStable {
						offer, err := peer.Peer.CreateOffer(nil)
						if err != nil {
							log.Printf("Error creating offer: %v", err)
							return
						}

						err = peer.Peer.SetLocalDescription(offer)
						if err != nil {
							log.Printf("Error setting local description: %v", err)
							return
						}

						peer.Conn.WriteJSON(OfferStruct{
							SDP:  offer.SDP,
							Type: "offer",
							ID:   peer.ID,
						})
					}
				}
			} else if state == webrtc.PeerConnectionStateFailed ||
				state == webrtc.PeerConnectionStateClosed ||
				state == webrtc.PeerConnectionStateDisconnected {
				// Log disconnection for debugging
				log.Printf("\033[33mViewer %s disconnected with state %s\033[0m", peer.ID, state)
				r.RemovePeer(peer.ID)
			}
		})
	}
}

// func (r *Room) AddTracksToPeer(peer *Peer) {
// 	for _, roomPeer := range r.Peers {
// 		if roomPeer.Peer.ID == peer.ID || roomPeer.Type == RoomPeerTypeSource {
// 			continue
// 		}
// 		for _, track := range roomPeer.Peer.Peer.GetReceivers() {
// 			if track.Track() != nil && track.Track().Kind() == webrtc.RTPCodecTypeVideo {
// 				peer.Peer.AddTrack(track.Track())
// 			}
// 		}
// 	}
// }

type JoinRoomMessage struct {
	Type       string       `json:"type"`        // Type of message
	ID         string       `json:"id"`          // ID of peer to join room
	Room       string       `json:"room"`        // Name of room to join
	MemberType RoomPeerType `json:"member_type"` // Type of member to join room
}

func (r *Room) HandleJoinRoomMessage(peer *Peer, message []byte) {
	log.Println("Peer:", peer.ID, "joined room:", r.ID)
	var joinRoomMessage JoinRoomMessage
	err := json.Unmarshal(message, &joinRoomMessage)
	if err != nil {
		log.Println("Error unmarshalling join room message:", err)
		return
	}
	r.AddPeer(peer, joinRoomMessage.MemberType)

	// if joinRoomMessage.MemberType == RoomPeerTypeSource {
	// 	peer.AddOnTrackHandler(func(remoteTrack *webrtc.TrackRemote, receiver *webrtc.RTPReceiver, localTrack *webrtc.TrackLocalStaticRTP) {
	// 		log.Println("Adding track to room:", r.ID, remoteTrack.ID())
	// 	})
	// }

	// for _, roomPeer := range r.Peers {
	// 	if roomPeer.Type == RoomPeerTypeViewer || roomPeer.Type == RoomPeerTypeBoth {
	// 		r.AddTracksToPeer(roomPeer.Peer)
	// 	}
	// }

	// if joinRoomMessage.MemberType == RoomMemberTypeSource {
	// 	r.AddSource(peer)
	// } else if joinRoomMessage.MemberType == RoomMemberTypeBoth {
	// 	r.AddPeer(peer)
	// }
}

func (r *Room) RemovePeer(peerId string) {
	r.PeersMu.Lock()
	defer r.PeersMu.Unlock()

	peer, exists := r.Peers[peerId]
	if !exists {
		return
	}

	// If the peer is a source, remove their tracks from all viewers
	if peer.Type == RoomPeerTypeSource || peer.Type == RoomPeerTypeBoth {
		for _, roomPeer := range r.Peers {
			if roomPeer.Type == RoomPeerTypeViewer || roomPeer.Type == RoomPeerTypeBoth {
				// Remove all tracks that came from the disconnecting peer
				for _, sender := range roomPeer.Peer.Peer.GetSenders() {
					// TODO clean LocalTracks
					if err := roomPeer.Peer.Peer.RemoveTrack(sender); err != nil {
						log.Printf("Error removing track from peer %s: %v", roomPeer.Peer.ID, err)
					}
				}
			}
		}
	}

	// Clean up the peer's connection
	if peer.Peer.Peer != nil {
		peer.Peer.Peer.Close()
	}

	delete(r.Peers, peerId)
	log.Println("\033[32mRemoved peer", peerId, "from room:", r.ID, "\033[0m")
}
