package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/starbased-co/shine/pkg/paths"
	"github.com/starbased-co/shine/pkg/rpc"
)

// Panel represents a spawned Kitty panel running prismctl
type Panel struct {
	Name       string            // Prism name (e.g., "shine-clock")
	Instance   string            // Instance name for socket (e.g., "clock", "bar")
	WindowID   string            // Kitty window ID
	PID        int               // prismctl process PID
	SocketPath string            // Path to prismctl Unix socket
	RPCClient  *rpc.PrismClient  // RPC client for communication
	Config     *PrismEntry       // Configuration from prism.toml
	CrashCount int               // Crash counter for restart policy
	LastCrash  time.Time         // Last crash timestamp
}

// PrismRestartState tracks restart state for a prism within a panel
type PrismRestartState struct {
	RestartCount     int       		// Number of restarts in current hour
	RestartTimestamps []time.Time // Rate limiter timestamp
	ExplicitlyStopped bool      	// Determines unless-stopped restart policy
}

// PanelManager manages the lifecycle of the panel
type PanelManager struct {
	mu          sync.Mutex
	panels      map[string]*Panel // Map: instance name -> Panel
	logDir      string
	prismctlBin string
	// Restart state tracking: panelInstance -> prismName -> RestartState
	restartState map[string]map[string]*PrismRestartState
}

// getPIDFromWindowID retrieves the PID of the process running in a Kitty window
func getPIDFromWindowID(windowID string) (int, error) {
	// Use kitten @ ls to get window information
	cmd := exec.Command("kitten", "@", "ls")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to list kitty windows: %w", err)
	}

	// Parse JSON output
	var osWindows []struct {
		ID   int `json:"id"`
		Tabs []struct {
			ID      int `json:"id"`
			Windows []struct {
				ID  int `json:"id"`
				PID int `json:"pid"`
			} `json:"windows"`
		} `json:"tabs"`
	}

	if err := json.Unmarshal(output, &osWindows); err != nil {
		return 0, fmt.Errorf("failed to parse kitty ls output: %w", err)
	}

	// Find the window with matching ID
	targetID := windowID
	for _, osWin := range osWindows {
		for _, tab := range osWin.Tabs {
			for _, win := range tab.Windows {
				if fmt.Sprintf("%d", win.ID) == targetID {
					return win.PID, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("window ID %s not found", windowID)
}

// NewPanelManager creates a new panel manager
func NewPanelManager() (*PanelManager, error) {
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
		// Try relative to shined binary
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
		panels:       make(map[string]*Panel),
		logDir:       logDir,
		prismctlBin:  prismctlBin,
		restartState: make(map[string]map[string]*PrismRestartState),
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

	// Pass only instance name to prismctl (apps sent via RPC)
	prismctlArgs := []string{instanceName}

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

	log.Printf("Spawned panel %s (window ID: %s)", instanceName, windowID)

	// Get PID from window ID
	pid, err := getPIDFromWindowID(windowID)
	if err != nil {
		log.Printf("Warning: failed to get PID for window %s: %v", windowID, err)
		pid = 0
	}

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

	// Create RPC client
	rpcClient, err := rpc.NewPrismClient(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}

	panel := &Panel{
		Name:       config.Name,
		Instance:   instanceName,
		WindowID:   windowID,
		PID:        pid,
		SocketPath: socketPath,
		RPCClient:  rpcClient,
		Config:     config,
		CrashCount: 0,
	}

	pm.panels[instanceName] = panel

	// Configure apps via RPC
	if err := pm.configureApps(panel, config); err != nil {
		return nil, fmt.Errorf("failed to configure apps: %w", err)
	}

	return panel, nil
}

// configureApps sends app configuration to prismctl via RPC
func (pm *PanelManager) configureApps(panel *Panel, config *PrismEntry) error {
	apps := make([]rpc.AppInfo, 0)

	for name, appCfg := range config.GetApps() {
		if appCfg == nil || !appCfg.Enabled || appCfg.ResolvedPath == "" {
			continue
		}
		apps = append(apps, rpc.AppInfo{
			Name:    name,
			Path:    appCfg.ResolvedPath,
			Enabled: appCfg.Enabled,
		})
	}

	if len(apps) == 0 {
		return fmt.Errorf("no enabled apps with resolved paths")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, err := panel.RPCClient.Configure(ctx, apps)
	if err != nil {
		return err
	}

	if len(result.Failed) > 0 {
		return fmt.Errorf("failed to start apps: %v", result.Failed)
	}

	log.Printf("Configured panel %s with %d apps: %v", panel.Instance, len(result.Started), result.Started)
	return nil
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
	// Try health check via RPC
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := panel.RPCClient.Health(ctx)
	return err == nil
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

	// Pass only instance name to prismctl (apps sent via RPC)
	prismctlArgs := []string{instanceName}

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

	// Get PID from window ID
	pid, err := getPIDFromWindowID(windowID)
	if err != nil {
		log.Printf("Warning: failed to get PID for window %s: %v", windowID, err)
		pid = 0
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

	// Create RPC client
	rpcClient, err := rpc.NewPrismClient(socketPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %w", err)
	}

	panel := &Panel{
		Name:       config.Name,
		Instance:   instanceName,
		WindowID:   windowID,
		PID:        pid,
		SocketPath: socketPath,
		RPCClient:  rpcClient,
		Config:     config,
		CrashCount: 0,
	}

	pm.panels[instanceName] = panel

	// Configure apps via RPC
	if err := pm.configureApps(panel, config); err != nil {
		return nil, fmt.Errorf("failed to configure apps: %w", err)
	}

	return panel, nil
}

// Shutdown gracefully stops all panels
func (pm *PanelManager) Shutdown() {
	panels := pm.ListPanels()

	for _, panel := range panels {
		log.Printf("Stopping panel %s", panel.Instance)

		// Request graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = panel.RPCClient.Shutdown(ctx, true)
		cancel()

		pm.KillPanel(panel.Instance)
	}
}

// getRestartState retrieves or creates restart state for a prism in a panel
func (pm *PanelManager) getRestartState(panelInstance, prismName string) *PrismRestartState {
	if pm.restartState[panelInstance] == nil {
		pm.restartState[panelInstance] = make(map[string]*PrismRestartState)
	}

	if pm.restartState[panelInstance][prismName] == nil {
		pm.restartState[panelInstance][prismName] = &PrismRestartState{
			RestartTimestamps: make([]time.Time, 0),
		}
	}

	return pm.restartState[panelInstance][prismName]
}

// pruneRestartTimestamps removes timestamps older than 1 hour
func pruneRestartTimestamps(timestamps []time.Time) []time.Time {
	now := time.Now()
	cutoff := now.Add(-1 * time.Hour)

	pruned := make([]time.Time, 0, len(timestamps))
	for _, t := range timestamps {
		if t.After(cutoff) {
			pruned = append(pruned, t)
		}
	}
	return pruned
}

// MarkPrismStopped marks a prism as explicitly stopped (for unless-stopped policy)
func (pm *PanelManager) MarkPrismStopped(panelInstance, prismName string, exitCode int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state := pm.getRestartState(panelInstance, prismName)

	// Only mark as explicitly stopped if exit code is 0 (clean exit)
	if exitCode == 0 {
		state.ExplicitlyStopped = true
		log.Printf("[%s] Prism %s marked as explicitly stopped", panelInstance, prismName)
	}
}

// TriggerRestartPolicy evaluates and executes restart policy for a crashed prism
func (pm *PanelManager) TriggerRestartPolicy(panelInstance, prismName string, exitCode int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Get panel
	panel, ok := pm.panels[panelInstance]
	if !ok {
		log.Printf("Panel %s not found, cannot restart prism %s", panelInstance, prismName)
		return
	}

	// Get restart state
	state := pm.getRestartState(panelInstance, prismName)

	// Prune old restart timestamps (older than 1 hour)
	state.RestartTimestamps = pruneRestartTimestamps(state.RestartTimestamps)
	state.RestartCount = len(state.RestartTimestamps)

	// Get restart policy from panel config
	policy := panel.Config.GetRestartPolicy()
	maxRestarts := panel.Config.MaxRestarts
	restartDelay := panel.Config.GetRestartDelay()

	// Determine if restart should happen
	shouldRestart := false
	reason := ""

	switch policy {
	case RestartNo:
		reason = "policy is 'no'"
		shouldRestart = false

	case RestartAlways:
		reason = "policy is 'always'"
		shouldRestart = true

	case RestartOnFailure:
		if exitCode != 0 {
			reason = fmt.Sprintf("policy is 'on-failure' and exit code is %d", exitCode)
			shouldRestart = true
		} else {
			reason = "policy is 'on-failure' but exit code is 0 (clean exit)"
			shouldRestart = false
		}

	case RestartUnlessStopped:
		if state.ExplicitlyStopped {
			reason = "policy is 'unless-stopped' and prism was explicitly stopped"
			shouldRestart = false
		} else {
			reason = "policy is 'unless-stopped' and prism was not explicitly stopped"
			shouldRestart = true
		}
	}

	if !shouldRestart {
		log.Printf("[%s] Not restarting prism %s: %s", panelInstance, prismName, reason)
		return
	}

	// Check max_restarts limit
	if maxRestarts > 0 && state.RestartCount >= maxRestarts {
		log.Printf("[%s] Prism %s exceeded max_restarts (%d/%d), not restarting",
			panelInstance, prismName, state.RestartCount, maxRestarts)
		return
	}

	log.Printf("[%s] Will restart prism %s: %s (restart count: %d, delay: %v)",
		panelInstance, prismName, reason, state.RestartCount, restartDelay)

	// Update restart state
	state.RestartTimestamps = append(state.RestartTimestamps, time.Now())
	state.RestartCount = len(state.RestartTimestamps)
	state.ExplicitlyStopped = false // Clear explicit stop flag on restart

	// Restart asynchronously after delay
	go pm.restartPrismAsync(panel, prismName, restartDelay, state.RestartCount)
}

// restartPrismAsync restarts a prism after a delay (runs in goroutine)
func (pm *PanelManager) restartPrismAsync(panel *Panel, prismName string, delay time.Duration, restartCount int) {
	// Sleep for restart delay
	time.Sleep(delay)

	log.Printf("[%s] Attempting restart #%d of prism %s", panel.Instance, restartCount, prismName)

	// Create RPC client to panel's prismctl
	client, err := rpc.NewPrismClient(panel.SocketPath)
	if err != nil {
		log.Printf("[%s] Failed to create RPC client for restart: %v", panel.Instance, err)
		return
	}
	defer client.Close()

	// Call up (which launches or resumes the prism)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.Up(ctx, prismName)
	if err != nil {
		log.Printf("[%s] Failed to restart prism %s: %v", panel.Instance, prismName, err)
		return
	}

	log.Printf("[%s] Successfully restarted prism %s", panel.Instance, prismName)
}
