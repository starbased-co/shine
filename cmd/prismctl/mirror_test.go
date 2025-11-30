package main

import (
	"context"
	"io"
	"os"
	"testing"
	"time"
)

func TestActivateMirror_NilRealPTY(t *testing.T) {
	ctx := context.Background()

	tmpFile, err := os.CreateTemp("", "test-pty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = activateMirror(ctx, nil, tmpFile)
	if err == nil {
		t.Error("activateMirror() with nil realPTY should return error")
	}

	expectedErr := "cannot activate mirror with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("activateMirror() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestActivateMirror_NilChildPTY(t *testing.T) {
	ctx := context.Background()

	tmpFile, err := os.CreateTemp("", "test-pty")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	_, err = activateMirror(ctx, tmpFile, nil)
	if err == nil {
		t.Error("activateMirror() with nil childPTY should return error")
	}

	expectedErr := "cannot activate mirror with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("activateMirror() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestActivateMirror_BothNil(t *testing.T) {
	ctx := context.Background()

	_, err := activateMirror(ctx, nil, nil)
	if err == nil {
		t.Error("activateMirror() with both nil PTYs should return error")
	}

	expectedErr := "cannot activate mirror with nil PTY"
	if err.Error() != expectedErr {
		t.Errorf("activateMirror() error = %q, want %q", err.Error(), expectedErr)
	}
}

func TestActivateMirror_ValidPTYs(t *testing.T) {
	ctx := context.Background()

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

	state, err := activateMirror(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateMirror() unexpected error: %v", err)
	}

	if state == nil {
		t.Fatal("activateMirror() returned nil state")
	}

	if !state.active {
		t.Error("activateMirror() state.active = false, want true")
	}

	if state.ctx == nil {
		t.Error("activateMirror() state.ctx is nil")
	}

	if state.cancel == nil {
		t.Error("activateMirror() state.cancel is nil")
	}

	realR.Close()
	childW.Close()

	deactivateMirror(state)
}

func TestActivateMirror_Cancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

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

	state, err := activateMirror(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateMirror() unexpected error: %v", err)
	}

	cancel()

	time.Sleep(50 * time.Millisecond)

	// State should still be active (only deactivateMirror sets it to false)
	if !state.active {
		t.Error("cancellation should not directly set state.active to false")
	}

	realR.Close()
	childW.Close()

	deactivateMirror(state)
}

func TestActivateMirror_DataFlow(t *testing.T) {
	ctx := context.Background()

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

	state, err := activateMirror(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateMirror() unexpected error: %v", err)
	}

	testData := []byte("test input\n")
	go func() {
		realW.Write(testData)
		realW.Close()
	}()

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

	realR.Close()
	childW.Close()

	deactivateMirror(state)
}

func TestDeactivateMirror_NilState(t *testing.T) {
	deactivateMirror(nil)
}

func TestDeactivateMirror_InactiveState(t *testing.T) {
	state := &mirrorState{
		active: false,
	}

	deactivateMirror(state)
}

func TestDeactivateMirror_ActiveState(t *testing.T) {
	ctx := context.Background()

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

	state, err := activateMirror(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateMirror() unexpected error: %v", err)
	}

	if !state.active {
		t.Fatal("state should be active before stopping")
	}

	realR.Close()
	childW.Close()

	deactivateMirror(state)

	if state.active {
		t.Error("deactivateMirror() should set active to false")
	}
}

func TestDeactivateMirror_DoubleStop(t *testing.T) {
	ctx := context.Background()

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

	state, err := activateMirror(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateMirror() unexpected error: %v", err)
	}

	realR.Close()
	childW.Close()

	deactivateMirror(state)
	deactivateMirror(state)
}

func TestDeactivateMirror_WaitsForGoroutines(t *testing.T) {
	ctx := context.Background()

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

	state, err := activateMirror(ctx, realR, childW)
	if err != nil {
		t.Fatalf("activateMirror() unexpected error: %v", err)
	}

	realR.Close()
	childW.Close()

	done := make(chan struct{})
	go func() {
		deactivateMirror(state)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Error("deactivateMirror() did not complete within timeout")
	}
}

func TestMirrorState_StructFields(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	state := &mirrorState{
		ctx:    ctx,
		cancel: cancel,
		active: true,
	}

	if state.ctx != ctx {
		t.Error("mirrorState.ctx not set correctly")
	}

	if state.cancel == nil {
		t.Error("mirrorState.cancel is nil")
	}

	if !state.active {
		t.Error("mirrorState.active should be true")
	}

	state.wg.Add(1)
	state.wg.Done()
}
