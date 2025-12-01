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

type Panel struct {
	Name       string
	Instance   string
	WindowID   string
	PID        int
	SocketPath string
	RPCClient  *rpc.PrismClient
	Config     *PrismEntry
	CrashCount int
	LastCrash  time.Time
}

type PrismRestartState struct {
	RestartCount      int
	RestartTimestamps []time.Time
	ExplicitlyStopped bool
}

type PanelManager struct {
	mu       sync.Mutex
	panels   map[string]*Panel
	logDir   string
	prismctlBin string
	restartState map[string]map[string]*PrismRestartState
}

func getPIDFromWindowID(windowID string) (int, error) {
	cmd := exec.Command("kitten", "@", "ls")
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("failed to list kitty windows: %w", err)
	}

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

func NewPanelManager() (*PanelManager, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	logDir := filepath.Join(home, ".local", "share", "shine", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	prismctlBin, err := exec.LookPath("prismctl")
	if err != nil {
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

func (pm *PanelManager) SpawnPanel(config *PrismEntry, instanceName string) (*Panel, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if existing, ok := pm.panels[instanceName]; ok {
		return existing, nil
	}

	panelCfg := config.ToPanelConfig()
	prismctlArgs := []string{instanceName}
	kittenArgs := panelCfg.ToPanelArgs(pm.prismctlBin)
	kittenArgs = append(kittenArgs, prismctlArgs...)
	cmd := exec.Command("kitten", kittenArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to spawn panel: %w\nOutput: %s", err, string(output))
	}

	windowID := strings.TrimSpace(string(output))
	if windowID == "" {
		return nil, fmt.Errorf("failed to get window ID from Kitty")
	}

	log.Printf("Spawned panel %s (window ID: %s)", instanceName, windowID)

	pid, err := getPIDFromWindowID(windowID)
	if err != nil {
		log.Printf("Warning: failed to get PID for window %s: %v", windowID, err)
		pid = 0
	}

	socketPath := paths.PrismSocket(instanceName)

	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if _, err := os.Stat(socketPath); err != nil {
		return nil, fmt.Errorf("prismctl socket not created within timeout")
	}

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

	if err := pm.configureApps(panel, config); err != nil {
		return nil, fmt.Errorf("failed to configure apps: %w", err)
	}

	return panel, nil
}

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

func (pm *PanelManager) KillPanel(instanceName string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[instanceName]
	if !ok {
		return fmt.Errorf("panel %s not found", instanceName)
	}

	cmd := exec.Command("kitten", "@", "close-window", "--match", fmt.Sprintf("id:%s", panel.WindowID))
	if err := cmd.Run(); err != nil {
		log.Printf("Warning: failed to close window %s: %v", panel.WindowID, err)
	}

	delete(pm.panels, instanceName)
	log.Printf("Killed panel %s (window ID: %s)", instanceName, panel.WindowID)
	return nil
}

func (pm *PanelManager) GetPanel(instanceName string) (*Panel, bool) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[instanceName]
	return panel, ok
}

func (pm *PanelManager) ListPanels() []*Panel {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panels := make([]*Panel, 0, len(pm.panels))
	for _, panel := range pm.panels {
		panels = append(panels, panel)
	}
	return panels
}

func (pm *PanelManager) CheckHealth(panel *Panel) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := panel.RPCClient.Health(ctx)
	return err == nil
}

func (pm *PanelManager) MonitorPanels() {
	panels := pm.ListPanels()

	for _, panel := range panels {
		if !pm.CheckHealth(panel) {
			log.Printf("Panel %s is not responsive", panel.Instance)
			pm.handlePanelCrash(panel)
		}
	}
}

func (pm *PanelManager) handlePanelCrash(panel *Panel) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.panels, panel.Instance)

	now := time.Now()
	if now.Sub(panel.LastCrash) > time.Hour {
		panel.CrashCount = 0
	}
	panel.CrashCount++
	panel.LastCrash = now

	log.Printf("Panel %s crashed (crash count: %d)", panel.Instance, panel.CrashCount)

	policy := panel.Config.GetRestartPolicy()
	shouldRestart := false

	switch policy {
	case RestartAlways:
		shouldRestart = true
	case RestartOnFailure:
		shouldRestart = true
	case RestartUnlessStopped:
		shouldRestart = true
	case RestartNo:
		shouldRestart = false
	}

	if shouldRestart && panel.Config.MaxRestarts > 0 && panel.CrashCount > panel.Config.MaxRestarts {
		log.Printf("Panel %s exceeded max_restarts (%d), not restarting", panel.Instance, panel.Config.MaxRestarts)
		shouldRestart = false
	}

	if shouldRestart {
		delay := panel.Config.GetRestartDelay()
		log.Printf("Restarting panel %s after %v delay", panel.Instance, delay)

		go func() {
			time.Sleep(delay)
			pm.mu.Lock()
			defer pm.mu.Unlock()

			newPanel, err := pm.spawnPanelUnlocked(panel.Config, panel.Instance)
			if err != nil {
				log.Printf("Failed to restart panel %s: %v", panel.Instance, err)
				return
			}

			newPanel.CrashCount = panel.CrashCount
			newPanel.LastCrash = panel.LastCrash

			log.Printf("Successfully restarted panel %s", panel.Instance)
		}()
	}
}

func (pm *PanelManager) spawnPanelUnlocked(config *PrismEntry, instanceName string) (*Panel, error) {
	panelCfg := config.ToPanelConfig()
	prismctlArgs := []string{instanceName}
	kittenArgs := panelCfg.ToPanelArgs(pm.prismctlBin)
	kittenArgs = append(kittenArgs, prismctlArgs...)
	cmd := exec.Command("kitten", kittenArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to spawn panel: %w\nOutput: %s", err, string(output))
	}

	windowID := strings.TrimSpace(string(output))
	if windowID == "" {
		return nil, fmt.Errorf("failed to get window ID from Kitty")
	}

	pid, err := getPIDFromWindowID(windowID)
	if err != nil {
		log.Printf("Warning: failed to get PID for window %s: %v", windowID, err)
		pid = 0
	}

	socketPath := paths.PrismSocket(instanceName)

	for i := 0; i < 50; i++ {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	if _, err := os.Stat(socketPath); err != nil {
		return nil, fmt.Errorf("prismctl socket not created within timeout")
	}

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

	if err := pm.configureApps(panel, config); err != nil {
		return nil, fmt.Errorf("failed to configure apps: %w", err)
	}

	return panel, nil
}

func (pm *PanelManager) Shutdown() {
	panels := pm.ListPanels()

	for _, panel := range panels {
		log.Printf("Stopping panel %s", panel.Instance)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, _ = panel.RPCClient.Shutdown(ctx, true)
		cancel()

		pm.KillPanel(panel.Instance)
	}
}

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

func (pm *PanelManager) MarkPrismStopped(panelInstance, prismName string, exitCode int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	state := pm.getRestartState(panelInstance, prismName)

	if exitCode == 0 {
		state.ExplicitlyStopped = true
		log.Printf("[%s] Prism %s marked as explicitly stopped", panelInstance, prismName)
	}
}

func (pm *PanelManager) TriggerRestartPolicy(panelInstance, prismName string, exitCode int) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	panel, ok := pm.panels[panelInstance]
	if !ok {
		log.Printf("Panel %s not found, cannot restart prism %s", panelInstance, prismName)
		return
	}

	state := pm.getRestartState(panelInstance, prismName)
	state.RestartTimestamps = pruneRestartTimestamps(state.RestartTimestamps)
	state.RestartCount = len(state.RestartTimestamps)

	policy := panel.Config.GetRestartPolicy()
	maxRestarts := panel.Config.MaxRestarts
	restartDelay := panel.Config.GetRestartDelay()

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

	if maxRestarts > 0 && state.RestartCount >= maxRestarts {
		log.Printf("[%s] Prism %s exceeded max_restarts (%d/%d), not restarting",
			panelInstance, prismName, state.RestartCount, maxRestarts)
		return
	}

	log.Printf("[%s] Will restart prism %s: %s (restart count: %d, delay: %v)",
		panelInstance, prismName, reason, state.RestartCount, restartDelay)

	state.RestartTimestamps = append(state.RestartTimestamps, time.Now())
	state.RestartCount = len(state.RestartTimestamps)
	state.ExplicitlyStopped = false

	go pm.restartPrismAsync(panel, prismName, restartDelay, state.RestartCount)
}

func (pm *PanelManager) restartPrismAsync(panel *Panel, prismName string, delay time.Duration, restartCount int) {
	time.Sleep(delay)

	log.Printf("[%s] Attempting restart #%d of prism %s", panel.Instance, restartCount, prismName)

	client, err := rpc.NewPrismClient(panel.SocketPath)
	if err != nil {
		log.Printf("[%s] Failed to create RPC client for restart: %v", panel.Instance, err)
		return
	}
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, err = client.Up(ctx, prismName)
	if err != nil {
		log.Printf("[%s] Failed to restart prism %s: %v", panel.Instance, prismName, err)
		return
	}

	log.Printf("[%s] Successfully restarted prism %s", panel.Instance, prismName)
}
