package rooms

/*
	RoomTrack is a track that is shared between members.
	Tracks may be webrtc tracks or other types of tracks.
*/

type Track interface {
	ID() string
}

type RoomTrack struct {
	Track
	id string
}

func (rt *RoomTrack) ID() string {
	return rt.id
}

type RoomVideoTrack struct {
	RoomTrack
}

func NewRoomVideoTrack(id string) *RoomVideoTrack {
	return &RoomVideoTrack{
		RoomTrack: RoomTrack{
			id: id,
		},
	}
}

type RoomAudioTrack struct {
	RoomTrack
}

func NewRoomAudioTrack(id string) *RoomAudioTrack {
	return &RoomAudioTrack{
		RoomTrack: RoomTrack{
			id: id,
		},
	}
}
