// Command yaawp-daemon is the headless service that runs the whatsmeow engine
// and exposes a Unix socket IPC interface to the GUI.
package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"cebi.tr/yaawp/internal/engine"
	"cebi.tr/yaawp/internal/ipc"
	"cebi.tr/yaawp/internal/store"
)

func main() {
	defaultSock, _ := store.SocketPath()
	sockPath := flag.String("socket", defaultSock, "Unix domain socket path for IPC")
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	eng, err := engine.New(ctx)
	if err != nil {
		log.Fatalf("engine init: %v", err)
	}

	srv := ipc.NewServer(*sockPath, eng)
	eng.SetSink(srv.Broadcast) // engine events go to every connected GUI

	if err := eng.Start(); err != nil {
		log.Printf("start warning: %v", err)
	}

	go func() {
		if err := srv.Serve(ctx); err != nil {
			log.Fatalf("ipc serve: %v", err)
		}
	}()

	log.Printf("yaawp-daemon running; socket=%s", *sockPath)
	<-ctx.Done()
	log.Println("shutting down")
	eng.Disconnect()
	_ = os.Remove(*sockPath)
}
