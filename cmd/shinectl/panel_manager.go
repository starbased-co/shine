package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/starbased-co/shine/pkg/ipc"
	"github.com/starbased-co/shine/pkg/paths"
)

// Panel represents a spawned Kitty panel running prismctl
type Panel struct {
	Name       string       // Prism name (e.g., "shine-clock")
	Instance   string       // Instance name for socket (e.g., "clock", "bar")
	WindowID   string       // Kitty window ID
	SocketPath string       // Path to prismctl Unix socket
	IPCClient  *ipc.Client  // IPC client for communication
	Config     *PrismEntry  // Configuration from prism.toml
	CrashCount int          // Crash counter for restart policy
	LastCrash  time.Time    // Last crash timestamp
}

// PanelManager manages the lifecycle of Kitty panels running prismctl
type PanelManager struct {
	mu          sync.Mutex
	panels      map[string]*Panel // Map: instance name -> Panel
	logDir      string
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
func (pm *PanelManager) SpawnPanel(config *PrismEntry, instanceName string) (*Panel, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Check if panel already exists
	if existing, ok := pm.panels[instanceName]; ok {
		return existing, nil
	}

	// Convert PrismEntry to panel.Config for positioning
	panelCfg := config.ToPanelConfig()

	// Build prismctl command path with arguments
	prismctlArgs := []string{config.Name, instanceName}

	// Generate kitten @ launch arguments with positioning
	kittenArgs := panelCfg.ToRemoteControlArgs(pm.prismctlBin)
	kittenArgs = append(kittenArgs, prismctlArgs...)

	// Launch Kitty panel using kitten @ launch with os-panel positioning
	cmd := exec.Command("kitten", kittenArgs...)

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

	log.Printf("Spawned panel %s (window ID: %s) for prism %s", instanceName, windowID, config.Name)

	// Build socket path (will be created by prismctl)
	socketPath := paths.PrismSocket(instanceName)

	// Wait for prismctl to create socket (up to 5 seconds)
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Verify socket was created
	if _, err := os.Stat(socketPath); err != nil {
		return nil, fmt.Errorf("prismctl socket not created within timeout")
	}

	panel := &Panel{
		Name:       config.Name,
		Instance:   instanceName,
		WindowID:   windowID,
		SocketPath: socketPath,
		IPCClient:  ipc.NewClient(socketPath),
		Config:     config,
		CrashCount: 0,
	}

	pm.panels[instanceName] = panel
	return panel, nil
}

// KillPanel terminates a panel by closing the Kitty window
func (pm *PanelManager) KillPanel(instanceName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[instanceName]
	if !ok {
		return fmt.Errorf("panel %s not found", instanceName)
	}

	// Close Kitty window (this will also kill prismctl)
	cmd := exec.Command("kitten", "@", "close-window", "--match", fmt.Sprintf("id:%s", panel.WindowID))
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to close window %s: %v", panel.WindowID, err)
	}

	delete(pm.panels, instanceName)
	log.Printf("Killed panel %s (window ID: %s)", instanceName, panel.WindowID)
	return nil
}

// GetPanel retrieves a panel by instance name
func (pm *PanelManager) GetPanel(instanceName string) (*Panel, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[instanceName]
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
			log.Printf("Panel %s is not responsive", panel.Instance)
			pm.handlePanelCrash(panel)
		}
	}
}

// handlePanelCrash handles a crashed panel according to restart policy
func (pm *PanelManager) handlePanelCrash(panel *Panel) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Remove from active panels
	delete(pm.panels, panel.Instance)

	// Update crash tracking
	now := time.Now()
	if now.Sub(panel.LastCrash) > time.Hour {
		// Reset counter if last crash was over an hour ago
		panel.CrashCount = 0
	}
	panel.CrashCount++
	panel.LastCrash = now

	log.Printf("Panel %s crashed (crash count: %d)", panel.Instance, panel.CrashCount)

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
		log.Printf("Panel %s exceeded max_restarts (%d), not restarting", panel.Instance, panel.Config.MaxRestarts)
		shouldRestart = false
	}

	if shouldRestart {
		delay := panel.Config.GetRestartDelay()
		log.Printf("Restarting panel %s after %v delay", panel.Instance, delay)

		// Restart after delay (in goroutine to not block)
		go func() {
			time.Sleep(delay)
			pm.mu.Lock()
			defer pm.mu.Unlock()

			// Re-spawn panel
			newPanel, err := pm.spawnPanelUnlocked(panel.Config, panel.Instance)
			if err != nil {
				log.Printf("Failed to restart panel %s: %v", panel.Instance, err)
				return
			}

			// Preserve crash tracking
			newPanel.CrashCount = panel.CrashCount
			newPanel.LastCrash = panel.LastCrash

			log.Printf("Successfully restarted panel %s", panel.Instance)
		}()
	}
}

// spawnPanelUnlocked is the internal spawn function (caller must hold lock)
func (pm *PanelManager) spawnPanelUnlocked(config *PrismEntry, instanceName string) (*Panel, error) {
	// Convert PrismEntry to panel.Config for positioning
	panelCfg := config.ToPanelConfig()

	// Build prismctl command path with arguments
	prismctlArgs := []string{config.Name, instanceName}

	// Generate kitten @ launch arguments with positioning
	kittenArgs := panelCfg.ToRemoteControlArgs(pm.prismctlBin)
	kittenArgs = append(kittenArgs, prismctlArgs...)

	// Launch Kitty panel using kitten @ launch with os-panel positioning
	cmd := exec.Command("kitten", kittenArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to spawn panel: %w\nOutput: %s", err, string(output))
	}

	windowID := strings.TrimSpace(string(output))
	if windowID == "" {
		return nil, fmt.Errorf("failed to get window ID from Kitty")
	}

	// Wait for socket
	socketPath := paths.PrismSocket(instanceName)

	// Wait for prismctl to create socket (up to 5 seconds)
	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Verify socket was created
	if _, err := os.Stat(socketPath); err != nil {
		return nil, fmt.Errorf("prismctl socket not created within timeout")
	}

	panel := &Panel{
		Name:       config.Name,
		Instance:   instanceName,
		WindowID:   windowID,
		SocketPath: socketPath,
		IPCClient:  ipc.NewClient(socketPath),
		Config:     config,
		CrashCount: 0,
	}

	pm.panels[instanceName] = panel
	return panel, nil
}

// Shutdown gracefully stops all panels
func (pm *PanelManager) Shutdown() {
	panels := pm.ListPanels()

	for _, panel := range panels {
		log.Printf("Stopping panel %s", panel.Instance)
		_ = panel.IPCClient.Stop()
		pm.KillPanel(panel.Instance)
	}
}
