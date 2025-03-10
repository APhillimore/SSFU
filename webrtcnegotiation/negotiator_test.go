package webrtcnegotiation

import (
	"fmt"
	"testing"

	"github.com/pion/webrtc/v3"
	"github.com/stretchr/testify/assert"
)

func ExampleWebRtcNegotiator_HandleOffer() {
	peerConnection, _ := webrtc.NewPeerConnection(webrtc.Configuration{})
	handleSetRemoteDescription := func(description webrtc.SessionDescription) error {
		fmt.Println("Set remote description called")
		return nil
	}
	negotiatorConfig := WebRTCNegotiatorConfig{
		ID:                         "example-id",
		IsPolite:                   true,
		HandleSetRemoteDescription: handleSetRemoteDescription,
	}
	negotiator := NewWebRtcNegotiator(negotiatorConfig)
	description, _ := peerConnection.CreateOffer(nil)
	negotiator.HandleOffer(&description, peerConnection.SignalingState())
	// Output:
	// Set remote description called
}

func TestWebRtcNegotiator(t *testing.T) {
	t.Run("Handle answer", func(t *testing.T) {
		t.Parallel()
		setLocalDescriptionCalled := 0
		handleSetLocalDescription := func(description webrtc.SessionDescription) error {
			setLocalDescriptionCalled++
			return nil
		}

		negotiatorConfig := WebRTCNegotiatorConfig{
			ID:                         "example-id",
			IsPolite:                   true,
			HandleSetLocalDescription:  handleSetLocalDescription,
			HandleCreateOffer:          nil,
			HandleSetRemoteDescription: nil,
		}
		negotiator := NewWebRtcNegotiator(negotiatorConfig)
		answer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeAnswer,
			SDP:  "dummy sdp",
		}
		negotiator.HandleAnswer(&answer)
		assert.Equal(t, 1, setLocalDescriptionCalled)
	})
	t.Run("Handle offer", func(t *testing.T) {
		t.Parallel()
		setRemoteDescriptionCalled := 0
		handleSetRemoteDescription := func(description webrtc.SessionDescription) error {
			setRemoteDescriptionCalled++
			return nil
		}
		negotiatorConfig := WebRTCNegotiatorConfig{
			ID:                         "example-id",
			IsPolite:                   true,
			HandleSetRemoteDescription: handleSetRemoteDescription,
		}
		negotiator := NewWebRtcNegotiator(negotiatorConfig)
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  "dummy sdp",
		}
		negotiator.HandleOffer(&offer, webrtc.SignalingStateStable)
		assert.Equal(t, 1, setRemoteDescriptionCalled)
	})
	t.Run("Handle offer with collision", func(t *testing.T) {
		t.Parallel()
		setRemoteDescriptionCalled := 0
		setLocalDescriptionCalled := 0
		handleSetRemoteDescription := func(description webrtc.SessionDescription) error {
			setRemoteDescriptionCalled++
			return nil
		}
		handleSetLocalDescription := func(description webrtc.SessionDescription) error {
			setLocalDescriptionCalled++
			return nil
		}
		handleCreateOffer := func() (webrtc.SessionDescription, error) {
			return webrtc.SessionDescription{
				Type: webrtc.SDPTypeOffer,
				SDP:  "dummy sdp",
			}, nil
		}
		handleSendRemoteDescription := func(description webrtc.SessionDescription) error {
			return nil
		}
		negotiatorConfig := WebRTCNegotiatorConfig{
			ID:                         "example-id",
			IsPolite:                   true,
			HandleSetRemoteDescription: handleSetRemoteDescription,
			HandleSetLocalDescription:  handleSetLocalDescription,
			HandleCreateOffer:          handleCreateOffer,
			HandleSendOffer:            handleSendRemoteDescription,
		}

		negotiator := NewWebRtcNegotiator(negotiatorConfig)
		offer := webrtc.SessionDescription{
			Type: webrtc.SDPTypeOffer,
			SDP:  "dummy sdp",
		}
		negotiator.SendOffer()
		negotiator.HandleOffer(&offer, webrtc.SignalingStateStable)
		assert.Equal(t, 0, setRemoteDescriptionCalled)
		assert.Equal(t, 1, setLocalDescriptionCalled)
	})
}
