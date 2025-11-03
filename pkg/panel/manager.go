package panel

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

// Instance represents a running panel instance
type Instance struct {
	Name        string
	Command     *exec.Cmd
	Config      *Config
	Remote      *RemoteControl
	WindowID    string // Window ID from kitty @ launch
	WindowTitle string // For window matching in shared instance mode
}

// kittyInstance tracks the shared Kitty instance
type kittyInstance struct {
	socketPath string
	pid        int
}

// Manager manages panel instances
type Manager struct {
	instances     map[string]*Instance
	kittyInstance *kittyInstance // Detected Kitty instance with remote control
	mu            sync.RWMutex
}

// NewManager creates a new panel manager
func NewManager() *Manager {
	return &Manager{
		instances: make(map[string]*Instance),
	}
}

// testSocket tests if a socket accepts remote control connections
func (m *Manager) testSocket(socketPath string) bool {
	var testCmd *exec.Cmd
	if socketPath != "" {
		testCmd = exec.Command("kitty", "@", "--to", socketPath, "ls")
	} else {
		// Use default socket
		testCmd = exec.Command("kitty", "@", "ls")
	}
	return testCmd.Run() == nil
}

// detectKittySocket finds a Kitty instance with remote control enabled
func (m *Manager) detectKittySocket() (string, error) {
	// Check if we already have an instance
	if m.kittyInstance != nil {
		// Verify it's still running
		socketPath := m.kittyInstance.socketPath
		if m.testSocket(socketPath) {
			return socketPath, nil
		}
		// Stale instance, clear it
		m.kittyInstance = nil
	}

	// Method 1: Check if we're already inside Kitty (has window ID)
	if kittyWindowID := os.Getenv("KITTY_WINDOW_ID"); kittyWindowID != "" {
		log.Printf("[kitty detection] Running inside Kitty window ID: %s", kittyWindowID)
		// When inside Kitty, try default socket first (no --to needed)
		testCmd := exec.Command("kitty", "@", "ls")
		if err := testCmd.Run(); err == nil {
			log.Printf("[kitty detection] Using default socket (no --to required)")
			// Use empty string to signal "use default"
			m.kittyInstance = &kittyInstance{
				socketPath: "", // Empty means use default
				pid:        0,
			}
			return "", nil
		}
	}

	// Method 2: Check KITTY_LISTEN_ON environment variable
	if listenOn := os.Getenv("KITTY_LISTEN_ON"); listenOn != "" {
		log.Printf("[kitty detection] Checking KITTY_LISTEN_ON: %s", listenOn)
		if m.testSocket(listenOn) {
			log.Printf("[kitty detection] Using KITTY_LISTEN_ON socket")
			m.kittyInstance = &kittyInstance{
				socketPath: listenOn,
				pid:        0,
			}
			return listenOn, nil
		}
	}

	// Method 3: Check for Kitty processes with PID-based sockets
	// Use "pgrep kitty" without -x to match Nix wrappers
	log.Printf("[kitty detection] Searching for Kitty processes...")
	cmd := exec.Command("pgrep", "kitty")
	output, err := cmd.Output()
	if err != nil {
		kittyWindowID := os.Getenv("KITTY_WINDOW_ID")
		return "", fmt.Errorf(`no kitty processes found

Troubleshooting:
1. Ensure Kitty is running
2. Enable remote control in Kitty:
   Add to ~/.config/kitty/kitty.conf:
     allow_remote_control yes
     listen_on unix:/tmp/@mykitty
3. Restart Kitty

Environment check:
  KITTY_WINDOW_ID: %s
  KITTY_LISTEN_ON: %s
  Running in Kitty: %v`, kittyWindowID, os.Getenv("KITTY_LISTEN_ON"), kittyWindowID != "")
	}

	pids := strings.Split(strings.TrimSpace(string(output)), "\n")
	log.Printf("[kitty detection] Found %d Kitty processes", len(pids))

	// Try each PID's socket (common pattern: /tmp/@mykitty-<PID>)
	for _, pidStr := range pids {
		pidStr = strings.TrimSpace(pidStr)
		if pidStr == "" {
			continue
		}
		socketPath := fmt.Sprintf("unix:/tmp/@mykitty-%s", pidStr)

		log.Printf("[kitty detection] Testing socket: %s", socketPath)
		if m.testSocket(socketPath) {
			log.Printf("[kitty detection] Successfully connected to socket")
			pid, _ := strconv.Atoi(pidStr)
			m.kittyInstance = &kittyInstance{
				socketPath: socketPath,
				pid:        pid,
			}
			return socketPath, nil
		}
	}

	kittyWindowID := os.Getenv("KITTY_WINDOW_ID")
	return "", fmt.Errorf(`no kitty instance with remote control enabled found

Checked %d Kitty processes but none accepted remote control connections.

To enable remote control, add to ~/.config/kitty/kitty.conf:
  allow_remote_control yes
  listen_on unix:/tmp/@mykitty

Then restart Kitty.

Environment:
  KITTY_WINDOW_ID: %s
  KITTY_LISTEN_ON: %s`, len(pids), kittyWindowID, os.Getenv("KITTY_LISTEN_ON"))
}

// LaunchViaRemoteControl launches a panel using Kitty's remote control API
func (m *Manager) LaunchViaRemoteControl(name string, config *Config, component string) (*Instance, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if _, exists := m.instances[name]; exists {
		return nil, fmt.Errorf("panel %s already running", name)
	}

	// Find Kitty socket
	socketPath, err := m.detectKittySocket()
	if err != nil {
		return nil, fmt.Errorf("failed to find kitty instance: %w (ensure Kitty is running with allow_remote_control=yes)", err)
	}

	// Set window title for tracking
	config.WindowTitle = fmt.Sprintf("shine-%s", name)

	// Build remote control args
	args := config.ToRemoteControlArgs(component)

	// Build full args
	var fullArgs []string
	if socketPath != "" {
		// Use specific socket
		fullArgs = append([]string{"--to", socketPath}, args...)
	} else {
		// Use default socket (when running inside Kitty)
		fullArgs = args
	}

	// Launch via remote control
	cmd := exec.Command("kitty", fullArgs...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to launch via remote control: %w", err)
	}

	// Parse window ID from output (kitty @ launch returns window ID)
	windowID := strings.TrimSpace(string(output))

	// Create remote control client
	remote := NewRemoteControl(strings.TrimPrefix(socketPath, "unix:"))

	// Create instance
	instance := &Instance{
		Name:        name,
		Command:     nil, // No command to track
		Config:      config,
		Remote:      remote,
		WindowID:    windowID,
		WindowTitle: config.WindowTitle,
	}

	// Store instance
	m.instances[name] = instance

	return instance, nil
}

// Launch starts a panel using Kitty's remote control API
func (m *Manager) Launch(name string, config *Config, component string) (*Instance, error) {
	return m.LaunchViaRemoteControl(name, config, component)
}

// Get retrieves an instance by name
func (m *Manager) Get(name string) (*Instance, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	instance, exists := m.instances[name]
	return instance, exists
}

// Stop closes a panel window (in shared instance mode) or kills process
func (m *Manager) Stop(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	instance, exists := m.instances[name]
	if !exists {
		return fmt.Errorf("panel %s not found", name)
	}

	// If launched via remote control (no command), close window
	if instance.WindowID != "" && instance.Remote != nil {
		if err := instance.Remote.CloseWindow(instance.WindowTitle); err != nil {
			return fmt.Errorf("failed to close window: %w", err)
		}
	} else if instance.Command != nil {
		// Otherwise, kill process (old kitten panel method or multi-instance mode)
		if instance.Command.Process != nil {
			if err := instance.Command.Process.Kill(); err != nil {
				return fmt.Errorf("failed to kill panel %s: %w", name, err)
			}
			_ = instance.Command.Wait()
		}
	}

	// Remove from instances
	delete(m.instances, name)

	return nil
}

// List returns all running panel names
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.instances))
	for name := range m.instances {
		names = append(names, name)
	}
	return names
}

// Wait waits for all panels to exit
func (m *Manager) Wait() {
	m.mu.RLock()
	instances := make([]*Instance, 0, len(m.instances))
	for _, instance := range m.instances {
		instances = append(instances, instance)
	}
	m.mu.RUnlock()

	// Wait for each instance
	for _, instance := range instances {
		// Skip if launched via remote control (no Command to wait on)
		if instance.Command != nil {
			_ = instance.Command.Wait()
		}
	}
}
