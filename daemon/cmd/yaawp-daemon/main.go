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

	setupLogging()

	// Refuse to run twice. The GUI spawns the daemon on demand and a systemd
	// user service may also start it; a stale second instance would remove the
	// live socket and orphan the first. An advisory lock keeps a single owner.
	if release, ok := acquireLock(); ok {
		defer release()
	} else {
		log.Println("another yaawp-daemon is already running; exiting")
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	eng, err := engine.New(ctx)
	if err != nil {
		log.Fatalf("engine init: %v", err)
	}

	srv := ipc.NewServer(*sockPath, eng)
	eng.SetSink(srv.Broadcast) // engine events go to every connected GUI

	// Start serving before connecting so the socket is ready when the GUI
	// launches. Any QR generated during Start is cached and delivered on connect.
	go func() {
		if err := srv.Serve(ctx); err != nil {
			log.Fatalf("ipc serve: %v", err)
		}
	}()

	if err := eng.Start(); err != nil {
		log.Printf("start warning: %v", err)
	}

	log.Printf("yaawp-daemon running; socket=%s", *sockPath)
	<-ctx.Done()
	log.Println("shutting down")
	eng.Disconnect()
	_ = os.Remove(*sockPath)
}

// setupLogging redirects stdout and stderr to a log file so the daemon leaves a
// diagnosable record even when the GUI launches it detached and discards its
// output. whatsmeow logs to stdout, so redirecting the file descriptors (rather
// than only the standard logger) captures the protocol logs too. The file is
// truncated when it grows past a couple of megabytes to keep it bounded.
func setupLogging() {
	path, err := store.LogPath()
	if err != nil {
		return
	}
	flags := os.O_CREATE | os.O_WRONLY | os.O_APPEND
	if fi, statErr := os.Stat(path); statErr == nil && fi.Size() > 2<<20 {
		flags |= os.O_TRUNC
	}
	f, err := os.OpenFile(path, flags, 0o600)
	if err != nil {
		return
	}
	// Point fds 1 and 2 at the log file; loggers and panics both follow.
	_ = syscall.Dup2(int(f.Fd()), int(os.Stdout.Fd()))
	_ = syscall.Dup2(int(f.Fd()), int(os.Stderr.Fd()))
}

// acquireLock takes an exclusive advisory lock on the daemon lock file. It
// returns a release function and true when this process is the sole instance,
// or false when another instance already holds the lock. The lock is released
// automatically when the process exits.
func acquireLock() (func(), bool) {
	path, err := store.LockPath()
	if err != nil {
		// Without a lock path we cannot guard; allow startup rather than block it.
		return func() {}, true
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return func() {}, true
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = f.Close()
		return nil, false
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, true
}
