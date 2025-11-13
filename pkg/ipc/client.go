package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client represents a client for communicating with prismctl instances
type Client struct {
	socketPath string
	timeout    time.Duration
}

// NewClient creates a new IPC client for the given socket path
func NewClient(socketPath string) *Client {
	return &Client{
		socketPath: socketPath,
		timeout:    5 * time.Second,
	}
}

// Send sends a command to the prismctl instance and returns the response
func (c *Client) Send(cmd Command) (*Response, error) {
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
	var resp Response
	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}

// Start sends a start command to launch or resume a prism
func (c *Client) Start(prismName string) error {
	cmd := Command{
		Action: "start",
		Prism:  prismName,
	}

	resp, err := c.Send(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("start failed: %s", resp.Message)
	}

	return nil
}

// Kill sends a kill command to terminate a prism
func (c *Client) Kill(prismName string) error {
	cmd := Command{
		Action: "kill",
		Prism:  prismName,
	}

	resp, err := c.Send(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("kill failed: %s", resp.Message)
	}

	return nil
}

// Status retrieves the current status of the prismctl instance
func (c *Client) Status() (*StatusResponse, error) {
	cmd := Command{
		Action: "status",
	}

	resp, err := c.Send(cmd)
	if err != nil {
		return nil, err
	}

	if !resp.Success {
		return nil, fmt.Errorf("status failed: %s", resp.Message)
	}

	// Parse data field as StatusResponse
	dataBytes, err := json.Marshal(resp.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status data: %w", err)
	}

	var status StatusResponse
	if err := json.Unmarshal(dataBytes, &status); err != nil {
		return nil, fmt.Errorf("failed to parse status data: %w", err)
	}

	return &status, nil
}

// Stop sends a stop command to gracefully shut down prismctl
func (c *Client) Stop() error {
	cmd := Command{
		Action: "stop",
	}

	resp, err := c.Send(cmd)
	if err != nil {
		return err
	}

	if !resp.Success {
		return fmt.Errorf("stop failed: %s", resp.Message)
	}

	return nil
}

// Ping checks if the prismctl instance is responsive
func (c *Client) Ping() error {
	_, err := c.Status()
	return err
}
