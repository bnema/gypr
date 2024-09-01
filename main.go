package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bnema/gypr/hyprland"
)

func main() {
	log.Println("Starting Hyprland Notifier...")

	events, errors := hyprland.ListenForEvents()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case event := <-events:
			handleEvent(event)
		case err := <-errors:
			log.Printf("Error: %v", err)
			return
		case <-ticker.C:
			log.Println("Hyprland Notifier is still running...")
		case sig := <-sigChan:
			log.Printf("Received signal: %v. Shutting down...", sig)
			return
		}
	}
}

func handleEvent(event hyprland.Event) {
	switch event.Type {
	case "workspace":
		workspaceInfo := hyprland.ParseWorkspaceInfo(event.Data)
		position := hyprland.GlobalMonitorWorkspaces.GetWorkspacePosition("current", workspaceInfo.ID)
		log.Printf("Switched to workspace: %s (%s)", workspaceInfo.ID, position)
	case "createworkspace":
		workspaceInfo := hyprland.ParseWorkspaceInfo(event.Data)
		log.Printf("New workspace created: %s", workspaceInfo.ID)
	case "destroyworkspace":
		log.Printf("Workspace destroyed: %s", event.Data)
	case "focusedmon":
		monitorInfo := hyprland.ParseMonitorInfo(event.Data)
		log.Printf("Focused monitor changed: %s, Workspace: %s", monitorInfo.Name, monitorInfo.Workspace)
		position := hyprland.GlobalMonitorWorkspaces.GetWorkspacePosition(monitorInfo.Name, monitorInfo.Workspace)
		log.Printf("Current workspace position: %s", position)
	case "activewindow":
		log.Printf("Active window changed: %s", event.Data)
	default:
		log.Printf("Unhandled event: %s - %s", event.Type, event.Data)
	}
}
