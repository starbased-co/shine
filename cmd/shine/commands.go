package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// findShinectlSocket finds the shinectl socket path
func findShinectlSocket() (string, error) {
	uid := os.Getuid()
	socketsDir := fmt.Sprintf("/run/user/%d/shine", uid)

	// Look for shine.*.sock
	pattern := filepath.Join(socketsDir, "shine.*.sock")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to search for socket: %w", err)
	}

	if len(matches) == 0 {
		return "", fmt.Errorf("shinectl is not running (socket not found)")
	}

	// Return the first match (should only be one)
	return matches[0], nil
}

// findPrismctlSockets finds all prismctl sockets
func findPrismctlSockets() ([]string, error) {
	uid := os.Getuid()
	socketsDir := fmt.Sprintf("/run/user/%d/shine", uid)

	pattern := filepath.Join(socketsDir, "prism-*.sock")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for sockets: %w", err)
	}

	return matches, nil
}

// ipcCommand represents a command to send via IPC
type ipcCommand struct {
	Action string `json:"action"`
	Prism  string `json:"prism,omitempty"`
}

// ipcResponse represents a response from IPC
type ipcResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// statusResponse represents the status command response from prismctl
type statusResponse struct {
	Foreground string        `json:"foreground"`
	Background []string      `json:"background"`
	Prisms     []prismStatus `json:"prisms"`
}

// prismStatus represents individual prism status
type prismStatus struct {
	Name  string `json:"name"`
	PID   int    `json:"pid"`
	State string `json:"state"`
}

// sendIPCCommand sends a command to a Unix socket and returns the response
func sendIPCCommand(socketPath string, cmd ipcCommand) (*ipcResponse, error) {
	conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Send command
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response
	reader := bufio.NewReader(conn)
	var resp ipcResponse
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}

// cmdStart starts or resumes the shinectl service
func cmdStart() error {
	// Check if shinectl is already running
	_, err := findShinectlSocket()
	if err == nil {
		Success("shinectl is already running")
		return nil
	}

	Info("Starting shinectl service...")

	// Find shinectl binary
	shinectlBin, err := exec.LookPath("shinectl")
	if err != nil {
		return fmt.Errorf("shinectl not found in PATH: %w", err)
	}

	// Start shinectl in background
	cmd := exec.Command(shinectlBin)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shinectl: %w", err)
	}

	// Wait for socket to appear
	for i := 0; i < 50; i++ {
		if _, err := findShinectlSocket(); err == nil {
			Success(fmt.Sprintf("shinectl started (PID: %d)", cmd.Process.Pid))
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("shinectl started but socket not created within timeout")
}

// cmdStop gracefully stops all panels and shinectl
func cmdStop() error {
	Info("Stopping shine service...")

	// Find all prismctl sockets
	sockets, err := findPrismctlSockets()
	if err != nil {
		return err
	}

	if len(sockets) == 0 {
		Warning("No panels running")
		return nil
	}

	// Send stop command to each panel
	for _, socket := range sockets {
		instanceName := extractInstanceName(socket)
		Muted(fmt.Sprintf("Stopping %s...", instanceName))

		cmd := ipcCommand{Action: "stop"}
		_, err := sendIPCCommand(socket, cmd)
		if err != nil {
			Warning(fmt.Sprintf("Failed to stop %s: %v", instanceName, err))
		}
	}

	Success(fmt.Sprintf("Stopped %d panel(s)", len(sockets)))
	return nil
}

// cmdReload reloads the configuration and updates panels
func cmdReload() error {
	Info("Reloading configuration...")

	// For now, we need to manually send SIGHUP to shinectl
	// In the future, we could have shinectl listen on its own socket

	Warning("Config reload via IPC not yet implemented")
	Info("To reload config, send SIGHUP to shinectl process:")
	Muted("  pkill -HUP shinectl")

	return nil
}

// cmdStatus shows the status of all panels
func cmdStatus() error {
	// Find all prismctl sockets
	sockets, err := findPrismctlSockets()
	if err != nil {
		return err
	}

	if len(sockets) == 0 {
		Warning("No panels running")
		Info("Start panels with: shine start")
		return nil
	}

	Header(fmt.Sprintf("Shine Status (%d panel(s))", len(sockets)))

	// Query each panel
	for _, socket := range sockets {
		instanceName := extractInstanceName(socket)
		fmt.Println()
		fmt.Printf("%s %s\n", styleBold.Render("Panel:"), instanceName)
		fmt.Printf("%s %s\n", styleMuted.Render("Socket:"), socket)

		// Query status
		cmd := ipcCommand{Action: "status"}
		resp, err := sendIPCCommand(socket, cmd)
		if err != nil {
			Error(fmt.Sprintf("Failed to query: %v", err))
			continue
		}

		if !resp.Success {
			Error(resp.Message)
			continue
		}

		// Parse status data
		dataBytes, _ := json.Marshal(resp.Data)
		var status statusResponse
		if err := json.Unmarshal(dataBytes, &status); err != nil {
			Error(fmt.Sprintf("Failed to parse status: %v", err))
			continue
		}

		// Display status
		fmt.Println(StatusBox(status.Foreground, len(status.Background), len(status.Prisms)))

		// Show prisms table
		if len(status.Prisms) > 0 {
			table := NewTable("Prism", "PID", "State")
			for _, prism := range status.Prisms {
				stateStr := prism.State
				if prism.State == "foreground" {
					stateStr = styleSuccess.Render("foreground")
				} else {
					stateStr = styleMuted.Render("background")
				}
				table.AddRow(prism.Name, fmt.Sprintf("%d", prism.PID), stateStr)
			}
			fmt.Println()
			table.Print()
		}
	}

	return nil
}

// cmdLogs shows logs for a specific panel or all panels
func cmdLogs(panelID string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := filepath.Join(home, ".local", "share", "shine", "logs")

	if panelID == "" {
		// Show all logs
		Info(fmt.Sprintf("Log directory: %s", logDir))

		files, err := os.ReadDir(logDir)
		if err != nil {
			return fmt.Errorf("failed to read log directory: %w", err)
		}

		if len(files) == 0 {
			Warning("No log files found")
			return nil
		}

		table := NewTable("Log File", "Size")
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			info, _ := file.Info()
			size := "?"
			if info != nil {
				size = fmt.Sprintf("%d bytes", info.Size())
			}
			table.AddRow(file.Name(), size)
		}

		table.Print()
		fmt.Println()
		Info("View a log with: shine logs <filename>")
		return nil
	}

	// Show specific log file
	logPath := filepath.Join(logDir, panelID)
	if !strings.HasSuffix(logPath, ".log") {
		logPath += ".log"
	}

	// Check if file exists
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logPath)
	}

	// Tail the log file (last 50 lines)
	cmd := exec.Command("tail", "-n", "50", logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to read log: %w", err)
	}

	return nil
}

// extractInstanceName extracts the instance name from a socket path
// e.g., "/run/user/1000/shine/prism-clock.sock" -> "clock"
func extractInstanceName(socketPath string) string {
	base := filepath.Base(socketPath)
	// Remove "prism-" prefix and ".sock" suffix
	name := strings.TrimPrefix(base, "prism-")
	name = strings.TrimSuffix(name, ".sock")
	return name
}
