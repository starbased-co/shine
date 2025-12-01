package main

import (
	"log"
	"time"

	"github.com/starbased-co/shine/pkg/state"
)

// StateManager wraps state.PrismStateWriter and provides methods
// that the supervisor calls on state changes
type StateManager struct {
	writer   *state.PrismStateWriter
	instance string
}

func newStateManager(statePath, instance string) (*StateManager, error) {
	writer, err := state.NewPrismStateWriter(statePath)
	if err != nil {
		return nil, err
	}

	writer.SetInstance(instance)

	return &StateManager{
		writer:   writer,
		instance: instance,
	}, nil
}

func (s *StateManager) OnPrismStarted(name string, pid int, fg bool) {
	log.Printf("State: prism started %s (PID %d, fg=%v)", name, pid, fg)

	if _, err := s.writer.AddPrism(name, int32(pid), fg); err != nil {
		log.Printf("Warning: failed to add prism to state: %v", err)
	}
}

func (s *StateManager) OnPrismStopped(name string) {
	log.Printf("State: prism stopped %s", name)
	s.writer.RemovePrism(name)
}

func (s *StateManager) OnForegroundChanged(name string) {
	log.Printf("State: foreground changed to %s", name)
	s.writer.SetForeground(name)
}

// OnPrismResumed is called when a suspended prism receives SIGCONT
func (s *StateManager) OnPrismResumed(name string) {
	s.OnForegroundChanged(name)
}

func (s *StateManager) UpdatePrism(index int, name string, pid int, fg bool, restarts uint8) {
	stateVal := state.PrismStateBg
	if fg {
		stateVal = state.PrismStateFg
	}

	if err := s.writer.SetPrism(index, name, int32(pid), stateVal, restarts, time.Now().UnixMilli()); err != nil {
		log.Printf("Warning: failed to update prism in state: %v", err)
	}
}

func (s *StateManager) Sync() error {
	return s.writer.Sync()
}

func (s *StateManager) Close() error {
	return s.writer.Close()
}

func (s *StateManager) Remove() error {
	log.Printf("State: removing state file at %s", s.writer.Path())
	return s.writer.Remove()
}

func (s *StateManager) Path() string {
	return s.writer.Path()
}
