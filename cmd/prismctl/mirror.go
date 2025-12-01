// mirror.go implements bidirectional I/O mirroring between the real PTY and
// child PTYs. The mirror reflects user input to the foreground prism and
// prism output back to the terminal.

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

type mirrorState struct {
	ctx      context.Context
	cancel   context.CancelFunc
	wg       sync.WaitGroup
	active   bool
	childPTY *os.File
}

// activateMirror launches bidirectional copy between Real PTY and child PTY
// Real PTY (stdin/stdout) ↔ child PTY master (foreground prism)
func activateMirror(ctx context.Context, realPTY *os.File, childPTY *os.File) (*mirrorState, error) {
	if realPTY == nil || childPTY == nil {
		return nil, fmt.Errorf("cannot activate mirror with nil PTY")
	}

	// Clear any previous read deadline (from deactivateMirror)
	if err := childPTY.SetReadDeadline(time.Time{}); err != nil {
		log.Printf("Warning: failed to clear read deadline: %v", err)
	}

	mirrorCtx, cancel := context.WithCancel(ctx)

	state := &mirrorState{
		ctx:      mirrorCtx,
		cancel:   cancel,
		active:   true,
		childPTY: childPTY,
	}

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
				log.Printf("Mirror (real→child) error: %v", err)
			}
		}
	}()

	// child PTY → Real PTY (prism output to terminal)
	go func() {
		defer state.wg.Done()
		if _, err := io.Copy(os.Stdout, childPTY); err != nil {
			if err != io.EOF && err != io.ErrClosedPipe && !isExpectedPTYError(err) {
				log.Printf("Mirror (child→real) error: %v", err)
			}
		}
	}()

	log.Printf("Mirror activated: Real PTY ↔ child PTY (fd %d)", childPTY.Fd())

	return state, nil
}

func deactivateMirror(state *mirrorState) {
	if state == nil || !state.active {
		return
	}

	state.cancel()

	// Force child PTY io.Copy to return by setting deadline
	if state.childPTY != nil {
		state.childPTY.SetReadDeadline(time.Unix(0, 0))
	}

	// Don't wait for goroutines - the stdin reader will be blocked until next input
	state.active = false
}

func isExpectedPTYError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "input/output error") || // EIO
		strings.Contains(errStr, "no such device") || // ENXIO
		strings.Contains(errStr, "i/o timeout") // Deadline set during deactivateMirror
}
