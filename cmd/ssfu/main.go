package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"simple-forwarding-unit/signalling"
	"simple-forwarding-unit/webrtcnegotiation"
	"simple-forwarding-unit/webrtcpeer"
	"simple-forwarding-unit/wsserver"
	"syscall"
	"time"

	"github.com/aggregator-cloud/rooms"
	"github.com/pion/webrtc/v3"
)

var (
	addr = flag.String("addr", "localhost:8081", "http service address")
)

func main() {
	log.Println("SSFU")

	log.Println("Starting server on", *addr)
	flag.Parse()

	// Create a channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Setup your HTTP server
	server := &http.Server{
		Addr: *addr,
	}

	webRtcWsManager := wsserver.NewWsManager()
	webRtcSignallingWsManager := webrtcnegotiation.NewWebRTCNegotiationManager()
	roomManager := rooms.NewRoomManager()
	peerManager := webrtcpeer.NewPeerManager()
	webRtcWsManager.OnConnectionHandlers.Add(wsserver.NewWsConnectionHandler(func(connection *wsserver.WsConnection) error {
		peer, err := webrtcpeer.NewSfuPeer(connection.ID(), &webrtc.Configuration{})
		if err != nil {
			return err
		}
		_, err = peerManager.AddPeer(peer)
		if err != nil {
			return err
		}

		negotiatorConfig := webrtcnegotiation.WebRTCNegotiatorConfig{
			ID:                         connection.ID(),
			IsPolite:                   true,
			HandleSetRemoteDescription: peer.SetRemoteDescription,
			HandleSetLocalDescription:  peer.SetLocalDescription,
			HandleCreateOffer: func() (webrtc.SessionDescription, error) {
				return peer.CreateOffer(nil)
			},
			HandleAddICECandidate: peer.AddICECandidate,
			HandleSendOffer: func(description webrtc.SessionDescription) error {
				return connection.Conn().WriteJSON(map[string]interface{}{
					"type":        "offer",
					"description": description,
				})
			},
		}

		negotiator := webRtcSignallingWsManager.Negotiators.Add(webrtcnegotiation.NewWebRtcNegotiator(negotiatorConfig))
		peer.AddOnICECandidateHandler(func(candidate *webrtc.ICECandidate) {
			negotiator.HandleCandidate(candidate)
		})
		return nil
	}))

	webRtcWsManager.OnMessageHandlers.Add(wsserver.NewWsMessageHandler(func(connection *wsserver.WsConnection, message []byte) error {
		// Signalling
		msg := map[string]interface{}{}
		err := json.Unmarshal(message, &msg)
		if err != nil {
			return errors.New("error unmarshalling message: " + err.Error())
		}

		msgType := msg["type"]
		if msgType == "join-room" {
			decoded := signalling.JoinRoomStruct{}
			err := json.Unmarshal(message, &decoded)
			if err != nil {
				return errors.New("error unmarshalling message: " + err.Error())
			}
			// Create message structs
			room := roomManager.CreateRoom(decoded.RoomID)
			member := rooms.NewRoomMember(decoded.MemberID, decoded.MemberType)
			room.AddMember(member)
		}

		peer, err := peerManager.GetPeer(connection.ID())
		if err != nil {
			return errors.New("peer not found")
		}
		negotiator, err := webRtcSignallingWsManager.Negotiators.GetByID(connection.ID())
		if err != nil {
			return errors.New("negotiator not found")
		}

		if _, ok := msg["description"]; ok {
			description := msg["description"].(*webrtc.SessionDescription)
			if description.Type == webrtc.SDPTypeOffer {
				negotiator.HandleOffer(description, (*peer).SignalingState())
			} else {
				negotiator.HandleAnswer(description)
			}
			log.Println("offer or answer ", msg)
		}

		return nil
	}))

	http.HandleFunc("/", webRtcWsManager.WebsocketEndpointHandler)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal
	<-stop

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

}
