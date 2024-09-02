package hyprland

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"

	"github.com/thiagokokada/hyprland-go/event"
)

type Workspace struct {
	ID              int
	Name            string
	Monitor         string
	MonitorID       int
	WindowCount     int
	HasFullscreen   bool
	LastWindow      string
	LastWindowTitle string
}

type Monitor struct {
	Name       string
	ID         int
	Workspaces []Workspace
}

func ListActiveWorkspaces() ([]Workspace, error) {
	cmd := exec.Command("hyprctl", "workspaces", "-j")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute hyprctl: %w", err)
	}

	var workspaces []Workspace
	if err := json.Unmarshal(output, &workspaces); err != nil {
		return nil, fmt.Errorf("failed to parse hyprctl output: %w", err)
	}

	return workspaces, nil
}

type EventHandler struct {
	event.DefaultEventHandler
}

func (h *EventHandler) Workspace(w event.WorkspaceName) {
	log.Printf("Workspace: %s", w)
	sendNotification("Workspace", fmt.Sprintf("%s", w))
}

func (h *EventHandler) FocusedMonitor(w event.FocusedMonitor) {
	monitorName := string(w.MonitorName)
	workspaceInfo, err := GetCurrentWorkspaceInfo(monitorName)
	if err != nil {
		log.Printf("Error getting workspace info: %v", err)
		return
	}

	message := fmt.Sprintf("Monitor: %s, Workspace: %s (%s)", monitorName, w.WorkspaceName, workspaceInfo)
	log.Println(message)
	sendNotification("Focus", message)
}

func StartEventListener(ctx context.Context) error {
	client := event.MustClient()
	defer client.Close()

	handler := &EventHandler{}

	return client.Subscribe(
		ctx,
		handler,
		event.EventWorkspace,
		event.EventFocusedMonitor,
	)
}

func sendNotification(title, message string) {
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}
