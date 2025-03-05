package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"simple-forwarding-unit/signalling"
	"syscall"
	"time"
)

var (
	addr        = flag.String("addr", "localhost:8081", "http service address")
	roomManager = signalling.NewRoomManager()
)

func viewerHTMLHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "viewer.html")
}

func sourceHTMLHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "source.html")
}

func perfectNegotiationHTMLHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "perfect-negotiation.js")
}

func main() {
	log.Println("Starting server on", *addr)
	flag.Parse()
	wsServer := signalling.NewWebSocketWebRTCSignallingServer()

	// Create a channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Setup your HTTP server
	server := &http.Server{
		Addr: *addr,
	}

	http.HandleFunc("/signalling", wsServer.HandleWsSignalling)
	http.HandleFunc("/viewer", viewerHTMLHandler)
	http.HandleFunc("/source", sourceHTMLHandler)
	http.HandleFunc("/perfect-negotiation.js", perfectNegotiationHTMLHandler)
	// Start server in goroutine
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

	wsServer.Shutdown()

	// Shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}
}
