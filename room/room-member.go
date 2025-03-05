package rooms

import (
	"maps"
	"slices"
	"sync"

	"github.com/google/uuid"
)

type Member interface {
	ID() string
}

type RoomMember struct {
	id string
}

func (rm *RoomMember) ID() string {
	return rm.id
}

type TrackHandler struct {
	id      string
	handler func(track Track)
}

type RoomViewerMember struct {
	RoomMember
	Tracks            map[string]Track
	tracksMu          sync.Mutex
	onTrackHandlers   []TrackHandler
	onTrackHandlersMu sync.Mutex
}

func NewRoomViewerMember(id string) *RoomViewerMember {
	return &RoomViewerMember{
		RoomMember: RoomMember{
			id: id,
		},
		Tracks:            make(map[string]Track),
		tracksMu:          sync.Mutex{},
		onTrackHandlers:   []TrackHandler{},
		onTrackHandlersMu: sync.Mutex{},
	}
}

func (rm *RoomViewerMember) AddOnTrackHandler(handler func(track Track)) *TrackHandler {
	rm.onTrackHandlersMu.Lock()
	defer rm.onTrackHandlersMu.Unlock()

	trackHandler := TrackHandler{
		id:      uuid.New().String(),
		handler: handler,
	}

	rm.onTrackHandlers = append(rm.onTrackHandlers, trackHandler)
	return &trackHandler
}

func (rm *RoomViewerMember) RemoveOnTrackHandler(handler *TrackHandler) {
	rm.onTrackHandlersMu.Lock()
	defer rm.onTrackHandlersMu.Unlock()

	rm.onTrackHandlers = slices.DeleteFunc(rm.onTrackHandlers, func(h TrackHandler) bool {
		return h.id == handler.id
	})
}

func (rm *RoomViewerMember) GetOnTrackHandlers() []TrackHandler {
	rm.onTrackHandlersMu.Lock()
	defer rm.onTrackHandlersMu.Unlock()
	handlers := make([]TrackHandler, len(rm.onTrackHandlers))
	copy(handlers, rm.onTrackHandlers)
	return handlers
}

func (rm *RoomViewerMember) AddTrack(track Track) {
	rm.tracksMu.Lock()
	defer rm.tracksMu.Unlock()
	rm.Tracks[track.ID()] = track
	rm.onTrackHandlersMu.Lock()
	defer rm.onTrackHandlersMu.Unlock()
	for _, handler := range rm.onTrackHandlers {
		handler.handler(track)
	}
}

func (rm *RoomViewerMember) RemoveTrack(track Track) {
	rm.tracksMu.Lock()
	defer rm.tracksMu.Unlock()

	delete(rm.Tracks, track.ID())
}

func (rm *RoomViewerMember) GetTracks() map[string]Track {
	rm.tracksMu.Lock()
	defer rm.tracksMu.Unlock()
	return maps.Clone(rm.Tracks)
}

type RoomSourceMember struct {
	RoomMember
}

func NewRoomSourceMember(id string) *RoomSourceMember {
	return &RoomSourceMember{
		RoomMember: RoomMember{
			id: id,
		},
	}
}
