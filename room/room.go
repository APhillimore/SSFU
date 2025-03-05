package rooms

import (
	"maps"
	"sync"
)

type Room struct {
	ID        string
	Tracks    map[string]Track
	tracksMu  sync.Mutex
	Members   map[string]Member
	membersMu sync.Mutex
}

func NewRoom(id string) *Room {
	return &Room{
		ID:        id,
		Tracks:    make(map[string]Track),
		tracksMu:  sync.Mutex{},
		Members:   make(map[string]Member),
		membersMu: sync.Mutex{},
	}
}

func (r *Room) AddMember(member Member) {
	r.membersMu.Lock()
	defer r.membersMu.Unlock()
	r.Members[member.ID()] = member

	// add tracks to viewers
	if viewer, ok := member.(*RoomViewerMember); ok {
		for _, track := range r.GetTracks() {
			viewer.AddTrack(track)
		}
	}
}

func (r *Room) RemoveMember(member Member) {
	r.membersMu.Lock()
	defer r.membersMu.Unlock()
	delete(r.Members, member.ID())
}

func (r *Room) GetMembers() []Member {
	r.membersMu.Lock()
	defer r.membersMu.Unlock()
	members := make([]Member, 0, len(r.Members))
	for _, member := range r.Members {
		members = append(members, member)
	}
	return members
}

func (r *Room) AddTrack(track Track) {
	r.tracksMu.Lock()
	defer r.tracksMu.Unlock()
	r.Tracks[track.ID()] = track
	r.membersMu.Lock()
	defer r.membersMu.Unlock()
	for _, member := range r.Members {
		if viewer, ok := member.(*RoomViewerMember); ok {
			viewer.AddTrack(track)
		}
	}
}

func (r *Room) RemoveTrack(track Track) {
	r.tracksMu.Lock()
	defer r.tracksMu.Unlock()
	delete(r.Tracks, track.ID())
}

func (r *Room) GetTracks() map[string]Track {
	r.tracksMu.Lock()
	defer r.tracksMu.Unlock()
	return maps.Clone(r.Tracks)
}

func (r *Room) Close() {
	r.tracksMu.Lock()
	defer r.tracksMu.Unlock()
	r.Tracks = make(map[string]Track)

	r.membersMu.Lock()
	defer r.membersMu.Unlock()
	r.Members = make(map[string]Member)
}
