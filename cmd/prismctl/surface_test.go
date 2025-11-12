package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestActivateSurface_NilRealPTY(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for child PTY
	tmpFile, err := os.CreateTemp("", "test-pty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = activateSurface(ctx, nil, tmpFile)
	if err == nil {
		t.Error("activateSurface() with nil realPTY should return error")
	}

	expectedErr := "cannot activate surface with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("activateSurface() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestActivateSurface_NilChildPTY(t *testing.T) {
	ctx := context.Background()

	// Create a temporary file for real PTY
	tmpFile, err := os.CreateTemp("", "test-pty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = activateSurface(ctx, tmpFile, nil)
	if err == nil {
		t.Error("activateSurface() with nil childPTY should return error")
	}

	expectedErr := "cannot activate surface with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("activateSurface() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestActivateSurface_BothNil(t *testing.T) {
	ctx := context.Background()

	_, err := activateSurface(ctx, nil, nil)
	if err == nil {
		t.Error("activateSurface() with both nil PTYs should return error")
	}

	expectedErr := "cannot activate surface with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("activateSurface() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestActivateSurface_ValidPTYs(t *testing.T) {
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
	state, err := activateSurface(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateSurface() unexpected error: %v", err)
	}

	if state == nil {
		t.Fatal("activateSurface() returned nil state")
	}

	if !state.active {
		t.Error("activateSurface() state.active = false, want true")
	}

	if state.ctx == nil {
		t.Error("activateSurface() state.ctx is nil")
	}

	if state.cancel == nil {
		t.Error("activateSurface() state.cancel is nil")
	}

	// Close pipes to trigger EOF in goroutines
	realR.Close()
	childW.Close()

	// Clean up - deactivateSurface will wait for goroutines to exit
	deactivateSurface(state)
}

func TestActivateSurface_Cancellation(t *testing.T) {
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
		t.Fatalf("activateSurface() unexpected error: %v", err)
	}

	// Cancel parent context
	cancel()

	// Give goroutines time to detect cancellation
	time.Sleep(50 * time.Millisecond)

	// State should still be active (only deactivateSurface sets it to false)
	if !state.active {
		t.Error("cancellation should not directly set state.active to false")
	}

	// Close pipes before cleanup
	realR.Close()
	childW.Close()

	// Clean up
	deactivateSurface(state)
}

func TestActivateSurface_DataFlow(t *testing.T) {
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
	state, err := activateSurface(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateSurface() unexpected error: %v", err)
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
	deactivateSurface(state)
}

func TestDeactivateSurface_NilState(t *testing.T) {
	// Should not panic
	deactivateSurface(nil)
}

func TestDeactivateSurface_InactiveState(t *testing.T) {
	state := &surfaceState{
		active: false,
	}

	// Should not panic
	deactivateSurface(state)
}

func TestDeactivateSurface_ActiveState(t *testing.T) {
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

	state, err := activateSurface(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateSurface() unexpected error: %v", err)
	}

	if !state.active {
		t.Fatal("state should be active before stopping")
	}

	// Close pipes to unblock goroutines
	realR.Close()
	childW.Close()

	deactivateSurface(state)

	if state.active {
		t.Error("deactivateSurface() should set active to false")
	}
}

func TestDeactivateSurface_DoubleStop(t *testing.T) {
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

	state, err := activateSurface(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateSurface() unexpected error: %v", err)
	}

	// Close pipes to unblock goroutines
	realR.Close()
	childW.Close()

	// First stop
	deactivateSurface(state)

	// Second stop should be safe (no-op)
	deactivateSurface(state)
}

func TestDeactivateSurface_WaitsForGoroutines(t *testing.T) {
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

	state, err := activateSurface(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateSurface() unexpected error: %v", err)
	}

	// Close pipes to trigger EOF in goroutines
	realR.Close()
	childW.Close()

	// deactivateSurface should wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		deactivateSurface(state)
		close(done)
	}()

	// Wait with timeout
	select {
	case <-done:
		// Success - deactivateSurface completed
	case <-time.After(2 * time.Second):
		t.Error("deactivateSurface() did not complete within timeout (goroutines may not have exited)")
	}
}

func TestSurfaceState_StructFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &surfaceState{
		ctx:    ctx,
		cancel: cancel,
		active: true,
	}

	if state.ctx != ctx {
		t.Error("surfaceState.ctx not set correctly")
	}

	if state.cancel == nil {
		t.Error("surfaceState.cancel is nil")
	}

	if !state.active {
		t.Error("surfaceState.active should be true")
	}

	// Verify wg is zero-initialized
	// If we can add to it, it's properly initialized
	state.wg.Add(1)
	state.wg.Done()
}
