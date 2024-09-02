package hyprland

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func listWorkspacesAndMonitors() ([]Monitor, error) {
	cmd := exec.Command("hyprctl", "workspaces")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute hyprctl: %w", err)
	}

	monitors := make(map[string]*Monitor)
	var currentWorkspace *Workspace

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "workspace ID") {
			parts := strings.Split(line, " ")
			id, _ := strconv.Atoi(parts[2])
			name := strings.Trim(parts[3], "()")
			monitor := strings.TrimSuffix(parts[6], ":")

			currentWorkspace = &Workspace{
				ID:      id,
				Name:    name,
				Monitor: monitor,
			}

			if _, exists := monitors[monitor]; !exists {
				monitors[monitor] = &Monitor{Name: monitor, Workspaces: []Workspace{}}
			}
		} else if currentWorkspace != nil {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

				switch key {
				case "monitorID":
					currentWorkspace.MonitorID, _ = strconv.Atoi(value)
					monitors[currentWorkspace.Monitor].ID = currentWorkspace.MonitorID
				case "windows":
					currentWorkspace.WindowCount, _ = strconv.Atoi(value)
				case "hasfullscreen":
					currentWorkspace.HasFullscreen = value != "0"
				case "lastwindow":
					currentWorkspace.LastWindow = value
				case "lastwindowtitle":
					currentWorkspace.LastWindowTitle = value
					monitors[currentWorkspace.Monitor].Workspaces = append(monitors[currentWorkspace.Monitor].Workspaces, *currentWorkspace)
					currentWorkspace = nil
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading hyprctl output: %w", err)
	}

	var result []Monitor
	for _, monitor := range monitors {
		result = append(result, *monitor)
	}

	return result, nil
}

func getCurrentWorkspaceInfo(monitorName string) (string, error) {
	monitors, err := listWorkspacesAndMonitors()
	if err != nil {
		return "", err
	}

	for _, monitor := range monitors {
		if monitor.Name == monitorName {
			activeWorkspace := 0
			totalWorkspaces := len(monitor.Workspaces)
			for i, workspace := range monitor.Workspaces {
				if workspace.WindowCount > 0 {
					activeWorkspace = i + 1
					break
				}
			}
			return fmt.Sprintf("%d/%d", activeWorkspace, totalWorkspaces), nil
		}
	}

	return "", fmt.Errorf("monitor %s not found", monitorName)
}

func getCurrentActiveMonitor() (string, error) {
	cmd := exec.Command("hyprctl", "activeworkspace")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to execute hyprctl: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "on monitor") {
			parts := strings.Split(line, "on monitor")
			if len(parts) > 1 {
				return strings.TrimSpace(strings.TrimSuffix(parts[1], ":")), nil
			}
		}
	}

	return "", fmt.Errorf("active monitor not found in hyprctl output")
}
