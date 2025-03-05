package rooms

import (
	"maps"
	"sync"
)

type RoomManager struct {
	Rooms   map[string]*Room
	RoomsMu sync.Mutex
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

/* CreateRoom creates a new room if it doesn't exist and returns the room */
func (rm *RoomManager) CreateRoom(id string) *Room {
	rm.RoomsMu.Lock()
	defer rm.RoomsMu.Unlock()
	if _, ok := rm.Rooms[id]; ok {
		return rm.Rooms[id]
	}
	room := NewRoom(id)
	rm.Rooms[id] = room
	return room
}

func (rm *RoomManager) DestroyRoom(id string) {
	rm.RoomsMu.Lock()
	defer rm.RoomsMu.Unlock()
	room := rm.Rooms[id]
	if room == nil {
		return
	}
	room.Close()
	delete(rm.Rooms, id)
}

func (rm *RoomManager) GetRoom(id string) *Room {
	rm.RoomsMu.Lock()
	defer rm.RoomsMu.Unlock()
	return rm.Rooms[id]
}

func (rm *RoomManager) GetRooms() map[string]*Room {
	rm.RoomsMu.Lock()
	defer rm.RoomsMu.Unlock()
	return maps.Clone(rm.Rooms)
}
