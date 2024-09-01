package hyprland

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Event represents a Hyprland event
type Event struct {
	Type string
	Data string
}

// WorkspaceInfo contains information about a workspace
type WorkspaceInfo struct {
	ID   string
	Name string
}

// MonitorInfo contains information about a monitor
type MonitorInfo struct {
	Name      string
	Workspace string
}

// MonitorWorkspaces keeps track of workspaces for each monitor
type MonitorWorkspaces struct {
	sync.RWMutex
	workspaces map[string]map[string]bool // monitor name -> map of workspace names
	current    string                     // current workspace
}

// Global instance of MonitorWorkspaces
var GlobalMonitorWorkspaces = &MonitorWorkspaces{
	workspaces: make(map[string]map[string]bool),
	current:    "",
}

func ParseWorkspaceInfo(data string) WorkspaceInfo {
	parts := strings.SplitN(data, ",", 2)
	if len(parts) == 2 {
		return WorkspaceInfo{ID: parts[0], Name: parts[1]}
	}
	// If there's no comma, use the whole string as both ID and Name
	return WorkspaceInfo{ID: data, Name: data}
}

func ParseMonitorInfo(data string) MonitorInfo {
	parts := strings.SplitN(data, ",", 2)
	if len(parts) != 2 {
		return MonitorInfo{}
	}
	return MonitorInfo{Name: parts[0], Workspace: parts[1]}
}

func (mw *MonitorWorkspaces) AddWorkspace(monitor, workspace string) {
	mw.Lock()
	defer mw.Unlock()
	if _, exists := mw.workspaces[monitor]; !exists {
		mw.workspaces[monitor] = make(map[string]bool)
	}
	mw.workspaces[monitor][workspace] = true
	mw.current = workspace
}

func (mw *MonitorWorkspaces) RemoveWorkspace(monitor, workspace string) {
	mw.Lock()
	defer mw.Unlock()
	if _, exists := mw.workspaces[monitor]; exists {
		delete(mw.workspaces[monitor], workspace)
	}
}

func sendNotification(title, message string) error {
	cmd := exec.Command("notify-send", title, message)
	return cmd.Run()
}

func (mw *MonitorWorkspaces) GetWorkspacePosition(monitor, workspace string) string {
	mw.RLock()
	defer mw.RUnlock()
	if workspaces, exists := mw.workspaces[monitor]; exists {
		var keys []string
		for k := range workspaces {
			keys = append(keys, k)
		}
		for i, w := range keys {
			if w == workspace {
				return fmt.Sprintf("%d/%d", i+1, len(keys))
			}
		}
	}
	return "1/1" // Default to 1/1 if not found
}

func ListenForEvents() (<-chan Event, <-chan error) {
	events := make(chan Event)
	errors := make(chan error)
	go func() {
		socketPath := fmt.Sprintf("%s/hypr/%s/.socket2.sock", os.Getenv("XDG_RUNTIME_DIR"), os.Getenv("HYPRLAND_INSTANCE_SIGNATURE"))
		log.Printf("Attempting to connect to socket: %s", socketPath)
		conn, err := net.Dial("unix", socketPath)
		if err != nil {
			log.Printf("Failed to connect to Hyprland socket: %v", err)
			errors <- fmt.Errorf("failed to connect to Hyprland socket: %w", err)
			return
		}
		defer conn.Close()
		log.Println("Connected to Hyprland socket. Listening for events...")
		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			line := scanner.Text()
			log.Printf("Received event: %s", line)
			parts := strings.SplitN(line, ">>", 2)
			if len(parts) == 2 {
				event := Event{Type: parts[0], Data: parts[1]}
				events <- event

				// Handle workspace and monitor events
				switch event.Type {
				case "workspacev2":
					workspaceInfo := ParseWorkspaceInfo(event.Data)
					GlobalMonitorWorkspaces.AddWorkspace("current", workspaceInfo.Name)
					position := GlobalMonitorWorkspaces.GetWorkspacePosition("current", workspaceInfo.Name)
					log.Printf("Switched to workspace %s (%s)", workspaceInfo.Name, position)
					err := sendNotification("Workspace Changed", fmt.Sprintf("Switched to workspace %s (%s)", workspaceInfo.Name, position))
					if err != nil {
						log.Printf("Failed to send notification: %v", err)
					}
				case "createworkspacev2":
					workspaceInfo := ParseWorkspaceInfo(event.Data)
					GlobalMonitorWorkspaces.AddWorkspace("current", workspaceInfo.Name)
					position := GlobalMonitorWorkspaces.GetWorkspacePosition("current", workspaceInfo.Name)
					log.Printf("New workspace created: %s (%s)", workspaceInfo.Name, position)
					err := sendNotification("New Workspace", fmt.Sprintf("Created workspace %s (%s)", workspaceInfo.Name, position))
					if err != nil {
						log.Printf("Failed to send notification: %v", err)
					}
				case "destroyworkspacev2":
					workspaceInfo := ParseWorkspaceInfo(event.Data)
					GlobalMonitorWorkspaces.RemoveWorkspace("current", workspaceInfo.Name)
					log.Printf("Workspace destroyed: %s", workspaceInfo.Name)
					err := sendNotification("Workspace Destroyed", fmt.Sprintf("Destroyed workspace %s", workspaceInfo.Name))
					if err != nil {
						log.Printf("Failed to send notification: %v", err)
					}
				case "focusedmon":
					monitorInfo := ParseMonitorInfo(event.Data)
					GlobalMonitorWorkspaces.AddWorkspace(monitorInfo.Name, monitorInfo.Workspace)
					position := GlobalMonitorWorkspaces.GetWorkspacePosition(monitorInfo.Name, monitorInfo.Workspace)
					log.Printf("Focused monitor: %s (%s)", monitorInfo.Name, position)
					err := sendNotification("Monitor Focused", fmt.Sprintf("Focused monitor %s (%s)", monitorInfo.Name, position))
					if err != nil {
						log.Printf("Failed to send notification: %v", err)
					}
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Printf("Error reading from socket: %v", err)
			errors <- fmt.Errorf("error reading from socket: %w", err)
		}
	}()
	return events, errors
}
