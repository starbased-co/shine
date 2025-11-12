package main

import (
	"context"
	"os"
	"testing"
)

func TestPrismState_Constants(t *testing.T) {
	// Verify prismState constants are distinct
	if prismForeground == prismBackground {
		t.Error("prismForeground and prismBackground should be distinct")
	}

	// Verify zero value is prismForeground
	var state prismState
	if state != prismForeground {
		t.Errorf("zero value prismState = %v, want prismForeground (%v)", state, prismForeground)
	}
}

func TestPrismInstance_StructCreation(t *testing.T) {
	// Create a test PTY for the instance
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() failed: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	instance := prismInstance{
		name:      "test-prism",
		pid:       12345,
		state:     prismForeground,
		ptyMaster: master,
	}

	if instance.name != "test-prism" {
		t.Errorf("instance.name = %q, want %q", instance.name, "test-prism")
	}

	if instance.pid != 12345 {
		t.Errorf("instance.pid = %d, want 12345", instance.pid)
	}

	if instance.state != prismForeground {
		t.Errorf("instance.state = %v, want prismForeground", instance.state)
	}

	if instance.ptyMaster != master {
		t.Error("instance.ptyMaster not set correctly")
	}

	// Verify PTY master is valid
	if instance.ptyMaster.Fd() == 0 {
		t.Error("instance.ptyMaster has invalid FD")
	}
}

func TestPrismInstance_BackgroundState(t *testing.T) {
	instance := prismInstance{
		name:      "background-prism",
		pid:       67890,
		state:     prismBackground,
		ptyMaster: nil, // Can be nil in tests
	}

	if instance.state != prismBackground {
		t.Errorf("instance.state = %v, want prismBackground", instance.state)
	}
}

func TestNewSupervisor_Initialization(t *testing.T) {
	termState := &terminalState{}
	sup := newSupervisor(termState)

	if sup == nil {
		t.Fatal("newSupervisor() returned nil")
	}

	if sup.termState != termState {
		t.Error("supervisor.termState not set correctly")
	}

	if sup.prismList == nil {
		t.Error("supervisor.prismList is nil, want empty slice")
	}

	if len(sup.prismList) != 0 {
		t.Errorf("supervisor.prismList length = %d, want 0", len(sup.prismList))
	}

	if sup.shutdownCh == nil {
		t.Error("supervisor.shutdownCh is nil")
	}

	if sup.childExitCh == nil {
		t.Error("supervisor.childExitCh is nil")
	}

	if sup.surface != nil {
		t.Error("supervisor.relay should be nil initially")
	}

	if sup.surfaceCtx == nil {
		t.Error("supervisor.surfaceCtx is nil")
	}

	if sup.surfaceCancel == nil {
		t.Error("supervisor.relayCancel is nil")
	}

	// Verify shutdown channel is not closed
	select {
	case <-sup.shutdownCh:
		t.Error("shutdownCh should not be closed on initialization")
	default:
		// Expected
	}
}

func TestSupervisor_FindPrism_EmptyList(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	idx := sup.findPrism("nonexistent")
	if idx != -1 {
		t.Errorf("findPrism() on empty list = %d, want -1", idx)
	}
}

func TestSupervisor_FindPrism_SingleItem(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	sup.prismList = []prismInstance{
		{name: "test-prism", pid: 100},
	}

	idx := sup.findPrism("test-prism")
	if idx != 0 {
		t.Errorf("findPrism(\"test-prism\") = %d, want 0", idx)
	}

	idx = sup.findPrism("other-prism")
	if idx != -1 {
		t.Errorf("findPrism(\"other-prism\") = %d, want -1", idx)
	}
}

func TestSupervisor_FindPrism_MultipleItems(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	sup.prismList = []prismInstance{
		{name: "prism-a", pid: 100},
		{name: "prism-b", pid: 200},
		{name: "prism-c", pid: 300},
	}

	tests := []struct {
		name     string
		wantIdx  int
	}{
		{"prism-a", 0},
		{"prism-b", 1},
		{"prism-c", 2},
		{"nonexistent", -1},
	}

	for _, tt := range tests {
		idx := sup.findPrism(tt.name)
		if idx != tt.wantIdx {
			t.Errorf("findPrism(%q) = %d, want %d", tt.name, idx, tt.wantIdx)
		}
	}
}

func TestSupervisor_FindPrism_DuplicateNames(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// If duplicates exist (shouldn't happen in practice), return first match
	sup.prismList = []prismInstance{
		{name: "duplicate", pid: 100},
		{name: "duplicate", pid: 200},
	}

	idx := sup.findPrism("duplicate")
	if idx != 0 {
		t.Errorf("findPrism() with duplicates = %d, want 0 (first match)", idx)
	}
}

func TestSupervisor_IsShuttingDown_NotShutdown(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	if sup.isShuttingDown() {
		t.Error("isShuttingDown() = true on fresh supervisor, want false")
	}
}

func TestSupervisor_IsShuttingDown_AfterShutdown(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// Set shuttingDown flag (normally done by shutdown())
	sup.mu.Lock()
	sup.shuttingDown = true
	sup.mu.Unlock()

	if !sup.isShuttingDown() {
		t.Error("isShuttingDown() = false after setting shuttingDown, want true")
	}
}

func TestChildExit_StructCreation(t *testing.T) {
	exit := childExit{
		pid:      12345,
		exitCode: 0,
	}

	if exit.pid != 12345 {
		t.Errorf("childExit.pid = %d, want 12345", exit.pid)
	}

	if exit.exitCode != 0 {
		t.Errorf("childExit.exitCode = %d, want 0", exit.exitCode)
	}
}

func TestChildExit_NonZeroExitCode(t *testing.T) {
	exit := childExit{
		pid:      67890,
		exitCode: 1,
	}

	if exit.exitCode != 1 {
		t.Errorf("childExit.exitCode = %d, want 1", exit.exitCode)
	}
}

func TestSupervisor_RelayContextCancellation(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// Verify context is not cancelled initially
	select {
	case <-sup.surfaceCtx.Done():
		t.Error("surfaceCtx should not be cancelled on initialization")
	default:
		// Expected
	}

	// Cancel context
	sup.surfaceCancel()

	// Verify context is now cancelled
	select {
	case <-sup.surfaceCtx.Done():
		// Expected
	default:
		t.Error("surfaceCtx should be cancelled after calling relayCancel()")
	}
}

func TestSupervisor_PrismListOrdering(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// MRU list: [0] = foreground, [1+] = background in MRU order
	sup.prismList = []prismInstance{
		{name: "foreground", pid: 100, state: prismForeground},
		{name: "recent-bg", pid: 200, state: prismBackground},
		{name: "older-bg", pid: 300, state: prismBackground},
	}

	// Verify foreground is at position 0
	if sup.prismList[0].state != prismForeground {
		t.Error("prismList[0] should be foreground")
	}

	// Verify background prisms follow
	for i := 1; i < len(sup.prismList); i++ {
		if sup.prismList[i].state != prismBackground {
			t.Errorf("prismList[%d] should be background, got %v", i, sup.prismList[i].state)
		}
	}
}

func TestSupervisor_StartPrism_WrapperFunction(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// startPrism() should be a wrapper for start()
	// Both should fail with same error for nonexistent prism
	err := sup.startPrism("nonexistent-prism")
	if err == nil {
		t.Error("startPrism() with nonexistent prism should return error")
	}
}

func TestRelayState_ContextPropagation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create pipe pairs
	realR, realW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create real pipe: %v", err)
	}
	defer realW.Close()

	childR, childW, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create child pipe: %v", err)
	}
	defer childR.Close()

	state, err := activateSurface(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateSurface() failed: %v", err)
	}

	// Cancel parent context
	cancel()

	// Verify relay's context is cancelled
	select {
	case <-state.ctx.Done():
		// Expected - relay context inherits from parent
	default:
		t.Error("relay context should be cancelled when parent is cancelled")
	}

	// Close pipes before cleanup
	realR.Close()
	childW.Close()

	deactivateSurface(state)
}

func TestSupervisor_ChildExitChannelBuffered(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// Verify childExitCh is buffered (capacity 1)
	exit := childExit{pid: 100, exitCode: 0}

	// Should not block - channel has buffer
	select {
	case sup.childExitCh <- exit:
		// Success - channel is buffered
	default:
		t.Error("childExitCh should be buffered with capacity 1")
	}

	// Drain channel
	<-sup.childExitCh
}

func TestSupervisor_MutexProtection(t *testing.T) {
	sup := newSupervisor(&terminalState{})

	// Verify we can lock/unlock mutex (basic sanity check)
	sup.mu.Lock()
	sup.mu.Unlock()

	// Verify mutex protects concurrent access to prismList
	done := make(chan bool)
	go func() {
		sup.mu.Lock()
		sup.prismList = append(sup.prismList, prismInstance{name: "test", pid: 100})
		sup.mu.Unlock()
		done <- true
	}()

	<-done

	sup.mu.Lock()
	listLen := len(sup.prismList)
	sup.mu.Unlock()

	if listLen != 1 {
		t.Errorf("prismList length = %d after concurrent append, want 1", listLen)
	}
}
