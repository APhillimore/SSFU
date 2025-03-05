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
	"simple-forwarding-unit/webrtcnegotiation"
	"simple-forwarding-unit/wsserver"
	"syscall"
	"time"
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

	webRtcWsManager.OnConnectionHandlers.Add(wsserver.NewWsConnectionHandler(func(connection *wsserver.WsConnection) error {
		webRtcSignallingWsManager.Negotiators.Add(webrtcnegotiation.NewWebRtcNegotiator(connection.ID(), true))
		return nil
	}))

	webRtcWsManager.OnMessageHandlers.Add(wsserver.NewWsMessageHandler(func(connection *wsserver.WsConnection, message []byte) error {
		msg := map[string]interface{}{}
		err := json.Unmarshal(message, &msg)
		if err != nil {
			return errors.New("error unmarshalling message: " + err.Error())
		}
		log.Println("hello", msg)
		return nil
	}))

	webRtcWsManager.OnMessageHandlers.Add(wsserver.NewWsMessageHandler(func(connection *wsserver.WsConnection, message []byte) error {
		// Signalling
		msg := map[string]interface{}{}
		err := json.Unmarshal(message, &msg)
		if err != nil {
			return errors.New("error unmarshalling message: " + err.Error())
		}
		if _, ok := msg["description"]; ok {
			// webRtcSignallingWsManager.HandleOffer(msg)
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
