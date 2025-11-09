package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

// relayState tracks the state of an active relay
type relayState struct {
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	active    bool
	childPTY  *os.File // Track PTY so we can interrupt it
}

// startRelay launches bidirectional copy between Real PTY and child PTY
// Real PTY (stdin/stdout) ↔ child PTY master (foreground prism)
func startRelay(ctx context.Context, realPTY *os.File, childPTY *os.File) (*relayState, error) {
	if realPTY == nil || childPTY == nil {
		return nil, fmt.Errorf("cannot start relay with nil PTY")
	}

	// Clear any previous read deadline (from stopRelay)
	if err := childPTY.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("Warning: failed to clear read deadline: %v", err)
	}

	// Create cancellable context for this relay
	relayCtx, cancel := context.WithCancel(ctx)

	state := &relayState{
		ctx:      relayCtx,
		cancel:   cancel,
		active:   true,
		childPTY: childPTY,
	}

	// Start bidirectional relay goroutines
	state.wg.Add(2)

	// Real PTY → child PTY (user input to prism)
	go func() {
		defer state.wg.Done()
		if _, err := io.Copy(childPTY, realPTY); err != nil {
			// These errors are normal during shutdown/swap:
			// - EOF: clean close
			// - ErrClosedPipe: pipe closed
			// - "input/output error": PTY closed (ENXIO/EIO)
			if err != io.EOF && err != io.ErrClosedPipe && !isExpectedPTYError(err) {
				log.Printf("Relay (real→child) error: %v", err)
			}
		}
	}()

	// child PTY → Real PTY (prism output to terminal)
	go func() {
		defer state.wg.Done()
		if _, err := io.Copy(os.Stdout, childPTY); err != nil {
			// These errors are normal during shutdown/swap:
			// - EOF: clean close
			// - ErrClosedPipe: pipe closed
			// - "input/output error": PTY closed (ENXIO/EIO)
			if err != io.EOF && err != io.ErrClosedPipe && !isExpectedPTYError(err) {
				log.Printf("Relay (child→real) error: %v", err)
			}
		}
	}()

	log.Printf("Relay started: Real PTY ↔ child PTY (fd %d)", childPTY.Fd())

	return state, nil
}

// stopRelay cancels relay goroutines (non-blocking)
func stopRelay(state *relayState) {
	if state == nil || !state.active {
		return
	}

	// Cancel context
	state.cancel()

	// Force child PTY io.Copy to return by setting deadline
	// This interrupts one goroutine (child→real)
	if state.childPTY != nil {
		state.childPTY.SetReadDeadline(time.Unix(0, 0))
	}

	// Don't wait for goroutines - the stdin reader will be blocked until next input
	// Just abandon them and let them die naturally
	state.active = false
}

// isExpectedPTYError checks if an error is expected during PTY shutdown
func isExpectedPTYError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// Common PTY closure errors on Linux
	return strings.Contains(errStr, "input/output error") || // EIO
		strings.Contains(errStr, "no such device") || // ENXIO
		strings.Contains(errStr, "i/o timeout") // Deadline set during stopRelay
}
