package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Panel represents a spawned Kitty panel running prismctl
type Panel struct {
	Name       string       // Prism name (e.g., "shine-clock")
	Component  string       // Component name for socket (e.g., "panel-1")
	WindowID   string       // Kitty window ID
	SocketPath string       // Path to prismctl Unix socket
	IPCClient  *IPCClient   // IPC client for communication
	Config     *PrismEntry  // Configuration from prism.toml
	PID        int          // prismctl process PID
	CrashCount int          // Crash counter for restart policy
	LastCrash  time.Time    // Last crash timestamp
}

// PanelManager manages the lifecycle of Kitty panels running prismctl
type PanelManager struct {
	mu         sync.Mutex
	panels     map[string]*Panel // Map: component name -> Panel
	logDir     string
	prismctlBin string
}

// NewPanelManager creates a new panel manager
func NewPanelManager() (*PanelManager, error) {
	// Ensure log directory exists
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	logDir := filepath.Join(home, ".local", "share", "shine", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Find prismctl binary
	prismctlBin, err := exec.LookPath("prismctl")
	if err != nil {
		// Try relative to shinectl binary
		exePath, _ := os.Executable()
		if exePath != "" {
			prismctlBin = filepath.Join(filepath.Dir(exePath), "prismctl")
			if _, err := os.Stat(prismctlBin); err != nil {
				return nil, fmt.Errorf("prismctl not found in PATH or binary directory")
			}
		} else {
			return nil, fmt.Errorf("prismctl not found in PATH: %w", err)
		}
	}

	return &PanelManager{
		panels:      make(map[string]*Panel),
		logDir:      logDir,
		prismctlBin: prismctlBin,
	}, nil
}

// SpawnPanel spawns a new Kitty panel running prismctl
func (pm *PanelManager) SpawnPanel(config *PrismEntry, componentName string) (*Panel, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if panel already exists
	if existing, ok := pm.panels[componentName]; ok {
		return existing, nil
	}

	// Build prismctl command: prismctl <prism-name> <component-name>
	title := fmt.Sprintf("shine-%s", componentName)

	// Launch Kitty panel using kitten @ launch
	// Use --type=window to create a new window (panel) in current tab
	cmd := exec.Command(
		"kitten", "@", "launch",
		"--type=window",
		"--title", title,
		pm.prismctlBin, config.Name, componentName,
	)

	// Capture output to parse window ID
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to spawn panel: %w\nOutput: %s", err, string(output))
	}

	// Parse window ID from output (Kitty returns the window ID)
	windowID := strings.TrimSpace(string(output))
	if windowID == "" {
		return nil, fmt.Errorf("failed to get window ID from Kitty")
	}

	log.Printf("Spawned panel %s (window ID: %s) for prism %s", componentName, windowID, config.Name)

	// Build socket path (will be created by prismctl)
	uid := os.Getuid()
	socketPattern := fmt.Sprintf("/run/user/%d/shine/prism-%s.*.sock", uid, componentName)

	// Wait for prismctl to create socket (up to 5 seconds)
	var socketPath string
	for i := 0; i < 50; i++ {
		matches, _ := filepath.Glob(socketPattern)
		if len(matches) > 0 {
			socketPath = matches[0]
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if socketPath == "" {
		return nil, fmt.Errorf("prismctl socket not created within timeout")
	}

	// Extract PID from socket name (prism-{component}.{pid}.sock)
	base := filepath.Base(socketPath)
	parts := strings.Split(base, ".")
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid socket name format: %s", base)
	}
	pidStr := parts[len(parts)-2]
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse PID from socket name: %w", err)
	}

	panel := &Panel{
		Name:       config.Name,
		Component:  componentName,
		WindowID:   windowID,
		SocketPath: socketPath,
		IPCClient:  NewIPCClient(socketPath),
		Config:     config,
		PID:        pid,
		CrashCount: 0,
	}

	pm.panels[componentName] = panel
	return panel, nil
}

// KillPanel terminates a panel by closing the Kitty window
func (pm *PanelManager) KillPanel(componentName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[componentName]
	if !ok {
		return fmt.Errorf("panel %s not found", componentName)
	}

	// Close Kitty window (this will also kill prismctl)
	cmd := exec.Command("kitten", "@", "close-window", "--match", fmt.Sprintf("id:%s", panel.WindowID))
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to close window %s: %v", panel.WindowID, err)
	}

	delete(pm.panels, componentName)
	log.Printf("Killed panel %s (window ID: %s)", componentName, panel.WindowID)
	return nil
}

// GetPanel retrieves a panel by component name
func (pm *PanelManager) GetPanel(componentName string) (*Panel, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[componentName]
	return panel, ok
}

// ListPanels returns all active panels
func (pm *PanelManager) ListPanels() []*Panel {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panels := make([]*Panel, 0, len(pm.panels))
	for _, panel := range pm.panels {
		panels = append(panels, panel)
	}
	return panels
}

// CheckHealth checks if a panel's prismctl is still running
func (pm *PanelManager) CheckHealth(panel *Panel) bool {
	// Try to ping via IPC
	if err := panel.IPCClient.Ping(); err != nil {
		return false
	}
	return true
}

// MonitorPanels checks health of all panels and handles crashes
func (pm *PanelManager) MonitorPanels() {
	panels := pm.ListPanels()

	for _, panel := range panels {
		if !pm.CheckHealth(panel) {
			log.Printf("Panel %s (PID %d) is not responsive", panel.Component, panel.PID)
			pm.handlePanelCrash(panel)
		}
	}
}

// handlePanelCrash handles a crashed panel according to restart policy
func (pm *PanelManager) handlePanelCrash(panel *Panel) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Remove from active panels
	delete(pm.panels, panel.Component)

	// Update crash tracking
	now := time.Now()
	if now.Sub(panel.LastCrash) > time.Hour {
		// Reset counter if last crash was over an hour ago
		panel.CrashCount = 0
	}
	panel.CrashCount++
	panel.LastCrash = now

	log.Printf("Panel %s crashed (crash count: %d)", panel.Component, panel.CrashCount)

	// Check restart policy
	policy := panel.Config.GetRestartPolicy()
	shouldRestart := false

	switch policy {
	case RestartAlways:
		shouldRestart = true
	case RestartOnFailure:
		shouldRestart = true // Crash is a failure
	case RestartUnlessStopped:
		shouldRestart = true // Crash means it wasn't explicitly stopped
	case RestartNo:
		shouldRestart = false
	}

	// Check max_restarts limit
	if shouldRestart && panel.Config.MaxRestarts > 0 && panel.CrashCount > panel.Config.MaxRestarts {
		log.Printf("Panel %s exceeded max_restarts (%d), not restarting", panel.Component, panel.Config.MaxRestarts)
		shouldRestart = false
	}

	if shouldRestart {
		delay := panel.Config.GetRestartDelay()
		log.Printf("Restarting panel %s after %v delay", panel.Component, delay)

		// Restart after delay (in goroutine to not block)
		go func() {
			time.Sleep(delay)
			pm.mu.Lock()
			defer pm.mu.Unlock()

			// Re-spawn panel
			newPanel, err := pm.spawnPanelUnlocked(panel.Config, panel.Component)
			if err != nil {
				log.Printf("Failed to restart panel %s: %v", panel.Component, err)
				return
			}

			// Preserve crash tracking
			newPanel.CrashCount = panel.CrashCount
			newPanel.LastCrash = panel.LastCrash

			log.Printf("Successfully restarted panel %s (PID %d)", panel.Component, newPanel.PID)
		}()
	}
}

// spawnPanelUnlocked is the internal spawn function (caller must hold lock)
func (pm *PanelManager) spawnPanelUnlocked(config *PrismEntry, componentName string) (*Panel, error) {
	// Build prismctl command
	title := fmt.Sprintf("shine-%s", componentName)

	cmd := exec.Command(
		"kitten", "@", "launch",
		"--type=window",
		"--title", title,
		pm.prismctlBin, config.Name, componentName,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to spawn panel: %w\nOutput: %s", err, string(output))
	}

	windowID := strings.TrimSpace(string(output))
	if windowID == "" {
		return nil, fmt.Errorf("failed to get window ID from Kitty")
	}

	// Wait for socket
	uid := os.Getuid()
	socketPattern := fmt.Sprintf("/run/user/%d/shine/prism-%s.*.sock", uid, componentName)

	var socketPath string
	for i := 0; i < 50; i++ {
		matches, _ := filepath.Glob(socketPattern)
		if len(matches) > 0 {
			socketPath = matches[0]
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if socketPath == "" {
		return nil, fmt.Errorf("prismctl socket not created within timeout")
	}

	// Extract PID
	base := filepath.Base(socketPath)
	parts := strings.Split(base, ".")
	pidStr := parts[len(parts)-2]
	pid, _ := strconv.Atoi(pidStr)

	panel := &Panel{
		Name:       config.Name,
		Component:  componentName,
		WindowID:   windowID,
		SocketPath: socketPath,
		IPCClient:  NewIPCClient(socketPath),
		Config:     config,
		PID:        pid,
		CrashCount: 0,
	}

	pm.panels[componentName] = panel
	return panel, nil
}

// Shutdown gracefully stops all panels
func (pm *PanelManager) Shutdown() {
	panels := pm.ListPanels()

	for _, panel := range panels {
		log.Printf("Stopping panel %s", panel.Component)
		_ = panel.IPCClient.Stop()
		pm.KillPanel(panel.Component)
	}
}
