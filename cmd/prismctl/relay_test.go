package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestStartRelay_NilRealPTY(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for child PTY
	tmpFile, err := os.CreateTemp("", "test-pty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = startRelay(ctx, nil, tmpFile)
	if err == nil {
		t.Error("startRelay() with nil realPTY should return error")
	}

	expectedErr := "cannot start relay with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("startRelay() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestStartRelay_NilChildPTY(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for real PTY
	tmpFile, err := os.CreateTemp("", "test-pty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = startRelay(ctx, tmpFile, nil)
	if err == nil {
		t.Error("startRelay() with nil childPTY should return error")
	}

	expectedErr := "cannot start relay with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("startRelay() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestStartRelay_BothNil(t *testing.T) {
	ctx := context.Background()

	_, err := startRelay(ctx, nil, nil)
	if err == nil {
		t.Error("startRelay() with both nil PTYs should return error")
	}

	expectedErr := "cannot start relay with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("startRelay() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestStartRelay_ValidPTYs(t *testing.T) {
	ctx := context.Background()

	// Create pipe pairs to simulate PTY communication
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

	// Start relay with read end of real pipe and write end of child pipe
	state, err := startRelay(ctx, realR, childW)
	if err != nil {
		t.Fatalf("startRelay() unexpected error: %v", err)
	}

	if state == nil {
		t.Fatal("startRelay() returned nil state")
	}

	if !state.active {
		t.Error("startRelay() state.active = false, want true")
	}

	if state.ctx == nil {
		t.Error("startRelay() state.ctx is nil")
	}

	if state.cancel == nil {
		t.Error("startRelay() state.cancel is nil")
	}

	// Close pipes to trigger EOF in goroutines
	realR.Close()
	childW.Close()

	// Clean up - stopRelay will wait for goroutines to exit
	stopRelay(state)
}

func TestStartRelay_Cancellation(t *testing.T) {
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

	state, err := startRelay(ctx, realR, childW)
	if err != nil {
		t.Fatalf("startRelay() unexpected error: %v", err)
	}

	// Cancel parent context
	cancel()

	// Give goroutines time to detect cancellation
	time.Sleep(50 * time.Millisecond)

	// State should still be active (only stopRelay sets it to false)
	if !state.active {
		t.Error("cancellation should not directly set state.active to false")
	}

	// Close pipes before cleanup
	realR.Close()
	childW.Close()

	// Clean up
	stopRelay(state)
}

func TestStartRelay_DataFlow(t *testing.T) {
	ctx := context.Background()

	// Create pipe pairs for bidirectional communication
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

	// Start relay
	state, err := startRelay(ctx, realR, childW)
	if err != nil {
		t.Fatalf("startRelay() unexpected error: %v", err)
	}

	// Write data to real PTY (simulating user input)
	testData := []byte("test input\n")
	go func() {
		realW.Write(testData)
		realW.Close() // Close to signal EOF
	}()

	// Read from child PTY (should receive user input)
	buf := make([]byte, len(testData))
	childR.SetReadDeadline(time.Now().Add(1 * time.Second))
	n, err := io.ReadFull(childR, buf)
	if err != nil {
		t.Errorf("failed to read from child PTY: %v", err)
	}

	if n != len(testData) {
		t.Errorf("read %d bytes, want %d", n, len(testData))
	}

	if string(buf) != string(testData) {
		t.Errorf("read %q, want %q", string(buf), string(testData))
	}

	// Close pipes before cleanup
	realR.Close()
	childW.Close()

	// Clean up
	stopRelay(state)
}

func TestStopRelay_NilState(t *testing.T) {
	// Should not panic
	stopRelay(nil)
}

func TestStopRelay_InactiveState(t *testing.T) {
	state := &relayState{
		active: false,
	}

	// Should not panic
	stopRelay(state)
}

func TestStopRelay_ActiveState(t *testing.T) {
	ctx := context.Background()

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

	state, err := startRelay(ctx, realR, childW)
	if err != nil {
		t.Fatalf("startRelay() unexpected error: %v", err)
	}

	if !state.active {
		t.Fatal("state should be active before stopping")
	}

	// Close pipes to unblock goroutines
	realR.Close()
	childW.Close()

	stopRelay(state)

	if state.active {
		t.Error("stopRelay() should set active to false")
	}
}

func TestStopRelay_DoubleStop(t *testing.T) {
	ctx := context.Background()

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

	state, err := startRelay(ctx, realR, childW)
	if err != nil {
		t.Fatalf("startRelay() unexpected error: %v", err)
	}

	// Close pipes to unblock goroutines
	realR.Close()
	childW.Close()

	// First stop
	stopRelay(state)

	// Second stop should be safe (no-op)
	stopRelay(state)
}

func TestStopRelay_WaitsForGoroutines(t *testing.T) {
	ctx := context.Background()

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

	state, err := startRelay(ctx, realR, childW)
	if err != nil {
		t.Fatalf("startRelay() unexpected error: %v", err)
	}

	// Close pipes to trigger EOF in goroutines
	realR.Close()
	childW.Close()

	// stopRelay should wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		stopRelay(state)
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		// Success - stopRelay completed
	case <-time.After(2 * time.Second):
		t.Error("stopRelay() did not complete within timeout (goroutines may not have exited)")
	}
}

func TestRelayState_StructFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &relayState{
		ctx:    ctx,
		cancel: cancel,
		active: true,
	}

	if state.ctx != ctx {
		t.Error("relayState.ctx not set correctly")
	}

	if state.cancel == nil {
		t.Error("relayState.cancel is nil")
	}

	if !state.active {
		t.Error("relayState.active should be true")
	}

	// Verify wg is zero-initialized
	// If we can add to it, it's properly initialized
	state.wg.Add(1)
	state.wg.Done()
}
