package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bnema/gypr/hyprland"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the event listener in a separate goroutine
	go func() {
		if err := hyprland.StartEventListener(ctx); err != nil {
			log.Printf("Event listener stopped: %v", err)
		}
	}()

	// Wait for termination signal
	<-sigCh
	log.Println("Received termination signal")

	// Give some time for clean up
	log.Println("Shutting down...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 0)
	defer shutdownCancel()

	// Wait for clean shutdown or timeout
	<-shutdownCtx.Done()
	fmt.Println("Shutdown complete")
}
