package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ipcCommand represents a command sent via IPC
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

// ipcServer manages the Unix socket IPC server
type ipcServer struct {
	socketPath string
	listener   net.Listener
	supervisor *supervisor
	wg         sync.WaitGroup
	stopCh     chan struct{}
}

// newIPCServer creates a new IPC server
func newIPCServer(instance string, supervisor *supervisor) (*ipcServer, error) {
	// Use XDG runtime directory
	uid := os.Getuid()
	runtimeDir := fmt.Sprintf("/run/user/%d/shine", uid)

	// Create directory if needed
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create runtime directory: %w", err)
	}

	// Extract basename from instance path (handles "./test/fixtures/test-prism" -> "test-prism")
	instanceName := filepath.Base(instance)

	// Socket path
	socketPath := filepath.Join(runtimeDir, fmt.Sprintf("prism-%s.sock", instanceName))

	// Remove stale socket if exists
	_ = os.Remove(socketPath)

	// Create Unix socket listener
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create Unix socket: %w", err)
	}

	// Set socket permissions to user-only
	if err := os.Chmod(socketPath, 0600); err != nil {
		listener.Close()
		return nil, fmt.Errorf("failed to set socket permissions: %w", err)
	}

	log.Printf("IPC server listening on: %s", socketPath)

	return &ipcServer{
		socketPath: socketPath,
		listener:   listener,
		supervisor: supervisor,
		stopCh:     make(chan struct{}),
	}, nil
}

// serve starts accepting IPC connections
func (s *ipcServer) serve() {
	s.wg.Add(1)
	defer s.wg.Done()

	for {
		select {
		case <-s.stopCh:
			return
		default:
		}

		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}

		// Handle connection in goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection processes a single IPC connection
func (s *ipcServer) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		s.sendError(conn, "failed to read command")
		return
	}

	// Parse command
	var cmd ipcCommand
	if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &cmd); err != nil {
		s.sendError(conn, fmt.Sprintf("invalid JSON: %v", err))
		return
	}

	// Process command
	switch cmd.Action {
	case "start":
		s.handleStart(conn, cmd)
	case "kill":
		s.handleKill(conn, cmd)
	case "status":
		s.handleStatus(conn)
	case "stop":
		s.handleStop(conn)
	default:
		s.sendError(conn, fmt.Sprintf("unknown action: %s", cmd.Action))
	}
}

// handleStart processes a start command (idempotent launch/resume)
func (s *ipcServer) handleStart(conn net.Conn, cmd ipcCommand) {
	if cmd.Prism == "" {
		s.sendError(conn, "prism name required for start action")
		return
	}

	log.Printf("IPC: Received start request for %s", cmd.Prism)

	if err := s.supervisor.start(cmd.Prism); err != nil {
		s.sendError(conn, fmt.Sprintf("start failed: %v", err))
		return
	}

	s.sendSuccess(conn, "prism started/resumed", nil)
}

// handleKill processes a kill command
func (s *ipcServer) handleKill(conn net.Conn, cmd ipcCommand) {
	if cmd.Prism == "" {
		s.sendError(conn, "prism name required for kill action")
		return
	}

	log.Printf("IPC: Received kill request for %s", cmd.Prism)

	if err := s.supervisor.killPrism(cmd.Prism); err != nil {
		s.sendError(conn, fmt.Sprintf("kill failed: %v", err))
		return
	}

	s.sendSuccess(conn, "prism killed", nil)
}

// handleStatus processes a status query
func (s *ipcServer) handleStatus(conn net.Conn) {
	s.supervisor.mu.Lock()
	defer s.supervisor.mu.Unlock()

	// Build status response
	foreground := ""
	background := []string{}
	prisms := []prismStatus{}

	for i, p := range s.supervisor.prismList {
		state := "background"
		if i == 0 {
			state = "foreground"
			foreground = p.name
		} else {
			background = append(background, p.name)
		}

		prisms = append(prisms, prismStatus{
			Name:  p.name,
			PID:   p.pid,
			State: state,
		})
	}

	data := statusResponse{
		Foreground: foreground,
		Background: background,
		Prisms:     prisms,
	}

	s.sendSuccess(conn, "status ok", data)
}

// handleStop processes a stop command
func (s *ipcServer) handleStop(conn net.Conn) {
	log.Printf("IPC: Received stop request")
	s.sendSuccess(conn, "stopping", nil)

	// Trigger shutdown
	go s.supervisor.shutdown()
}

// sendSuccess sends a success response
func (s *ipcServer) sendSuccess(conn net.Conn, message string, data any) {
	resp := ipcResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	s.sendResponse(conn, resp)
}

// sendError sends an error response
func (s *ipcServer) sendError(conn net.Conn, message string) {
	resp := ipcResponse{
		Success: false,
		Message: message,
	}
	s.sendResponse(conn, resp)
}

// sendResponse sends a JSON response
func (s *ipcServer) sendResponse(conn net.Conn, resp ipcResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	data = append(data, '\n')
	if _, err := conn.Write(data); err != nil {
		log.Printf("Error writing response: %v", err)
	}
}

// stop stops the IPC server
func (s *ipcServer) stop() {
	close(s.stopCh)
	s.listener.Close()
	s.wg.Wait()

	// Clean up socket file
	_ = os.Remove(s.socketPath)

	log.Printf("IPC server stopped")
}
