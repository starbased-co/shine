package main

import (
	"log"
	"time"

	"github.com/starbased-co/shine/pkg/paths"
	"github.com/starbased-co/shine/pkg/state"
)

type StateManager struct {
	writer    *state.ShinedStateWriter
	startTime time.Time
}

func newStateManager() (*StateManager, error) {
	writer, err := state.NewShinedStateWriter(paths.ShinedState())
	if err != nil {
		return nil, err
	}

	return &StateManager{
		writer:    writer,
		startTime: time.Now(),
	}, nil
}

func (sm *StateManager) OnPanelSpawned(instance, name string, pid int, healthy bool) {
	_, err := sm.writer.AddPanel(instance, name, int32(pid), healthy)
	if err != nil {
		log.Printf("Failed to add panel to state: %v", err)
	}
}

func (sm *StateManager) OnPanelKilled(instance string) {
	sm.writer.RemovePanel(instance)
}

func (sm *StateManager) OnPanelHealthChanged(instance string, healthy bool) {
	sm.writer.SetPanelHealth(instance, healthy)
}

func (sm *StateManager) OnPanelPrismStarted(panel, name string, pid int) {
	log.Printf("State: panel %s - prism started: %s (PID %d)", panel, name, pid)
	// Future: could track prism state in panel metadata
}

func (sm *StateManager) OnPanelPrismStopped(panel, name string, exitCode int) {
	log.Printf("State: panel %s - prism stopped: %s (exit=%d)", panel, name, exitCode)
	// Future: could update prism state in panel metadata
}

func (sm *StateManager) OnPanelPrismCrashed(panel, name string, exitCode, signal int) {
	log.Printf("State: panel %s - prism crashed: %s (exit=%d, signal=%d)", panel, name, exitCode, signal)
	// Future: could trigger restart policy or mark panel unhealthy
}

func (sm *StateManager) OnPanelSurfaceSwitched(panel, from, to string) {
	log.Printf("State: panel %s - surface switched: %s → %s", panel, from, to)
	// Future: could track current foreground prism in panel metadata
}

func (sm *StateManager) Uptime() time.Duration {
	return time.Since(sm.startTime)
}

func (sm *StateManager) Close() error {
	return sm.writer.Close()
}

func (sm *StateManager) Remove() error {
	return sm.writer.Remove()
}
