package rooms

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoomMembers(t *testing.T) {
	room := NewRoom("1")
	member := NewRoomViewerMember("1")
	room.AddMember(member)

	members := room.GetMembers()
	assert.Equal(t, 1, len(members))
	assert.Equal(t, member, members[0])

	member2 := NewRoomViewerMember("2")
	room.AddMember(member2)

	members = room.GetMembers()
	assert.Equal(t, 2, len(members))
	assert.Equal(t, member2, members[1])
}

func TestAddTrack(t *testing.T) {
	room := NewRoom("1")

	videoTrack := NewRoomVideoTrack("1")
	audioTrack := NewRoomAudioTrack("2")

	room.AddTrack(videoTrack)
	room.AddTrack(audioTrack)

	tracks := room.GetTracks()
	assert.Equal(t, 2, len(tracks))
	assert.Equal(t, videoTrack, tracks["1"])
	assert.Equal(t, audioTrack, tracks["2"])

	member := NewRoomViewerMember("1")
	room.AddMember(member)

	tracks = member.GetTracks()
	assert.Equal(t, 2, len(tracks))
	assert.Equal(t, videoTrack, tracks["1"])
	assert.Equal(t, audioTrack, tracks["2"])
}

func TestRoomTrackHandlers(t *testing.T) {
	t.Run("track handler lifecycle", func(t *testing.T) {
		room := NewRoom("1")
		member := NewRoomViewerMember("1")
		room.AddMember(member)

		trackEvents := []string{}
		handlerFunc := func(track Track) {
			trackEvents = append(trackEvents, "track_added:"+track.ID())
		}

		// Test adding handler
		handler := member.AddOnTrackHandler(handlerFunc)
		assert.NotEmpty(t, handler, "handler ID should not be empty")

		// Test handler is called when track is added
		videoTrack := NewRoomVideoTrack("video1")
		room.AddTrack(videoTrack)
		assert.Equal(t, []string{"track_added:video1"}, trackEvents)

		// Test multiple tracks
		audioTrack := NewRoomVideoTrack("audio1")
		room.AddTrack(audioTrack)
		assert.Equal(t, []string{
			"track_added:video1",
			"track_added:audio1",
		}, trackEvents)

		// Test removing handler stops future notifications
		member.RemoveOnTrackHandler(handler)
		newTrack := NewRoomVideoTrack("video2")
		room.AddTrack(newTrack)
		assert.Equal(t, []string{
			"track_added:video1",
			"track_added:audio1",
		}, trackEvents, "no new events should be recorded after handler removal")

		// Verify handlers list is empty
		assert.Empty(t, member.GetOnTrackHandlers(), "handlers should be empty after removal")
	})

	t.Run("multiple handlers", func(t *testing.T) {
		room := NewRoom("1")
		member := NewRoomViewerMember("1")
		room.AddMember(member)

		handler1Called := false
		handler2Called := false

		handler1 := member.AddOnTrackHandler(func(track Track) {
			handler1Called = true
		})
		handler2 := member.AddOnTrackHandler(func(track Track) {
			handler2Called = true
		})

		// Both handlers should be called
		track := NewRoomVideoTrack("1")
		room.AddTrack(track)
		assert.True(t, handler1Called, "handler1 should be called")
		assert.True(t, handler2Called, "handler2 should be called")

		// Remove one handler
		member.RemoveOnTrackHandler(handler1)
		assert.Equal(t, 1, len(member.GetOnTrackHandlers()),
			"should have one handler remaining")

		member.RemoveOnTrackHandler(handler2)
		assert.Equal(t, 0, len(member.GetOnTrackHandlers()),
			"should have no handlers remaining")
	})

	t.Run("handler removal with invalid ID", func(t *testing.T) {
		member := NewRoomViewerMember("1")

		handler := member.AddOnTrackHandler(func(track Track) {})
		assert.Equal(t, 1, len(member.GetOnTrackHandlers()))

		// Remove with invalid ID should not affect valid handlers
		member.RemoveOnTrackHandler(&TrackHandler{id: "invalid-id"})
		assert.Equal(t, 1, len(member.GetOnTrackHandlers()),
			"valid handler should remain after removing invalid ID")

		// Remove with valid ID should work
		member.RemoveOnTrackHandler(handler)
		assert.Equal(t, 0, len(member.GetOnTrackHandlers()),
			"handler should be removed with valid ID")
	})
}
