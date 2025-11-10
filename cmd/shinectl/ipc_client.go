package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// IPCClient represents a client for communicating with prismctl instances
type IPCClient struct {
	socketPath string
	timeout    time.Duration
}

// NewIPCClient creates a new IPC client for the given socket path
func NewIPCClient(socketPath string) *IPCClient {
	return &IPCClient{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

// ipcCommand represents a command to send via IPC
type ipcCommand struct {
	Action string `json:"action"` // "start", "kill", "status", "stop"
	Prism  string `json:"prism"`  // Prism name for start/kill action
}

// ipcResponse represents a response from IPC
type ipcResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// statusResponse represents the status command response
type statusResponse struct {
	Foreground string        `json:"foreground"`
	Background []string      `json:"background"`
	Prisms     []prismStatus `json:"prisms"`
}

// prismStatus represents individual prism status
type prismStatus struct {
	Name  string `json:"name"`
	PID   int    `json:"pid"`
	State string `json:"state"` // "foreground" or "background"
}

// sendCommand sends a command to the prismctl instance and returns the response
func (c *IPCClient) sendCommand(cmd ipcCommand) (*ipcResponse, error) {
	// Connect to Unix socket with timeout
	conn, err := net.DialTimeout("unix", c.socketPath, c.timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to socket: %w", err)
	}
	defer conn.Close()

	// Set read/write deadlines
	conn.SetDeadline(time.Now().Add(c.timeout))

	// Send command as JSON
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

// Start sends a start command to launch or resume a prism
func (c *IPCClient) Start(prismName string) error {
	cmd := ipcCommand{
		Action: "start",
		Prism:  prismName,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("start failed: %s", resp.Message)
	}

	return nil
}

// Kill sends a kill command to terminate a prism
func (c *IPCClient) Kill(prismName string) error {
	cmd := ipcCommand{
		Action: "kill",
		Prism:  prismName,
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("kill failed: %s", resp.Message)
	}

	return nil
}

// Status retrieves the current status of the prismctl instance
func (c *IPCClient) Status() (*statusResponse, error) {
	cmd := ipcCommand{
		Action: "status",
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("status failed: %s", resp.Message)
	}

	// Parse data field as statusResponse
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status data: %w", err)
	}

	var status statusResponse
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status data: %w", err)
	}

	return &status, nil
}

// Stop sends a stop command to gracefully shut down prismctl
func (c *IPCClient) Stop() error {
	cmd := ipcCommand{
		Action: "stop",
	}

	resp, err := c.sendCommand(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("stop failed: %s", resp.Message)
	}

	return nil
}

// Ping checks if the prismctl instance is responsive
func (c *IPCClient) Ping() error {
	_, err := c.Status()
	return err
}
