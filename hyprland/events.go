package hyprland

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"sync"

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

type EventHandler struct {
	event.DefaultEventHandler
	currentMonitor string
	mutex          sync.Mutex
}

func NewEventHandler() (*EventHandler, error) {
	initialMonitor, err := getCurrentActiveMonitor()
	if err != nil {
		return nil, fmt.Errorf("failed to get initial active monitor: %w", err)
	}

	return &EventHandler{
		currentMonitor: initialMonitor,
	}, nil
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

func (h *EventHandler) Workspace(w event.WorkspaceName) {
	h.mutex.Lock()
	currentMonitor := h.currentMonitor
	h.mutex.Unlock()

	if currentMonitor == "" {
		var err error
		currentMonitor, err = getCurrentActiveMonitor()
		if err != nil {
			log.Printf("Error getting current active monitor: %v", err)
			return
		}
		h.mutex.Lock()
		h.currentMonitor = currentMonitor
		h.mutex.Unlock()
	}

	monitors, err := listWorkspacesAndMonitors()
	if err != nil {
		log.Printf("Error listing workspaces: %v", err)
		return
	}

	var targetMonitor *Monitor
	for i := range monitors {
		if monitors[i].Name == currentMonitor {
			targetMonitor = &monitors[i]
			break
		}
	}

	if targetMonitor == nil {
		log.Printf("Current monitor not found: %s", currentMonitor)
		return
	}

	var currentWorkspace *Workspace
	totalWorkspaces := len(targetMonitor.Workspaces)
	// Default to 1-based index
	currentWorkspaceIndex := 1

	for i, ws := range targetMonitor.Workspaces {
		if ws.Name == string(w) {
			currentWorkspace = &ws
			currentWorkspaceIndex = i + 1
			break
		}
	}

	var message string
	if currentWorkspace == nil {
		message = fmt.Sprintf("Workspace: %s (created empty) %d/%d", w, totalWorkspaces+1, totalWorkspaces+1)
	} else {
		message = fmt.Sprintf("Workspace: %s %d/%d", w, currentWorkspaceIndex, totalWorkspaces)
	}

	log.Println(message)
	sendNotification(message, "")
}

func (h *EventHandler) CreateWorkspace(w event.WorkspaceName) {
	log.Printf("Workspace created: %s", w)
	sendNotification(fmt.Sprintf("Workspace created: %s", w), "")
}

func (h *EventHandler) DestroyWorkspace(w event.WorkspaceName) {
	log.Printf("Workspace destroyed: %s", w)
	sendNotification(fmt.Sprintf("Workspace Destroyed: %s", w), "")
}

func (h *EventHandler) FocusedMonitor(w event.FocusedMonitor) {
	h.mutex.Lock()
	h.currentMonitor = string(w.MonitorName)
	h.mutex.Unlock()

	workspaceInfo, err := getCurrentWorkspaceInfo(string(w.MonitorName))
	if err != nil {
		log.Printf("Error getting workspace info: %v", err)
		return
	}

	message := fmt.Sprintf("Monitor: %s, Workspace: %s (%s)", w.MonitorName, w.WorkspaceName, workspaceInfo)
	log.Println(message)
	sendNotification(message, "")
}

func StartEventListener(ctx context.Context) error {
	client := event.MustClient()
	defer client.Close()

	handler, err := NewEventHandler()
	if err != nil {
		return fmt.Errorf("failed to create event handler: %w", err)
	}

	return client.Subscribe(
		ctx,
		handler,
		event.EventWorkspace,
		event.EventFocusedMonitor,
		event.EventCreateWorkspace,
		event.EventDestroyWorkspace,
	)
}

func sendNotification(title, message string) {
	cmd := exec.Command("notify-send", title, message)
	if err := cmd.Run(); err != nil {
		log.Printf("Failed to send notification: %v", err)
	}
}
