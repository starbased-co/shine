// signals.go handles signal propagation with prism-aware routing.
// Unlike typical process managers, prismctl routes signals differently based
// on application state:
//
// SIGWINCH (window size change)
//   - Propagated to ALL child PTYs, not just foreground
//   - Background prisms need correct size for when they become foreground
//
// SIGCHLD (child state change)
//   - Triggered when any child exits/stops
//   - Reaps ALL exited children in a loop (Wait4 with WNOHANG)
//   - Exit code: normal exit → status code, signal death → 128 + signal
//
// SIGINT (Ctrl+C)
//   - Kills ONLY the foreground prism (first in MRU list)
//   - If no prisms running, shuts down prismctl itself
//   - User can press Ctrl+C repeatedly to kill prisms one by one
//
// SIGTERM/SIGHUP
//   - Full graceful shutdown of all prisms

package main

import (
	"log"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

type signalHandler struct {
	sigCh      chan os.Signal
	supervisor *supervisor
}

func newSignalHandler(sup *supervisor) *signalHandler {
	sh := &signalHandler{
		sigCh:      make(chan os.Signal, 10),
		supervisor: sup,
	}

	signal.Notify(sh.sigCh,
		unix.SIGCHLD,
		unix.SIGTERM,
		unix.SIGINT,
		unix.SIGHUP,
		unix.SIGWINCH,
	)

	return sh
}

func (sh *signalHandler) run() {
	for sig := range sh.sigCh {
		switch sig {
		case unix.SIGCHLD:
			sh.handleSIGCHLD()
		case unix.SIGINT:
			// Ctrl+C: kill foreground prism if exists, otherwise shutdown
			if sh.handleSIGINT() {
				return // Shutdown requested
			}
		case unix.SIGTERM, unix.SIGHUP:
			sh.handleShutdown(sig)
			return
		case unix.SIGWINCH:
			sh.handleSIGWINCH()
		}
	}
}

func (sh *signalHandler) handleSIGCHLD() {
	// Reap all exited children
	for {
		var status unix.WaitStatus
		pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
		if err != nil || pid <= 0 {
			// No more children to reap
			break
		}

		// Notify supervisor of child exit
		exitCode := 0
		if status.Exited() {
			exitCode = status.ExitStatus()
			log.Printf("Child %d exited with code %d", pid, exitCode)
		} else if status.Signaled() {
			exitCode = 128 + int(status.Signal())
			log.Printf("Child %d terminated by signal %s", pid, status.Signal())
		}

		sh.supervisor.handleChildExit(pid, exitCode)
	}
}

// handleSIGINT handles Ctrl+C intelligently
// Returns true if prismctl should shutdown (exit signal loop)
func (sh *signalHandler) handleSIGINT() bool {
	sh.supervisor.mu.Lock()
	hasForeground := len(sh.supervisor.prismList) > 0
	var foregroundName string
	if hasForeground {
		foregroundName = sh.supervisor.prismList[0].name
	}
	sh.supervisor.mu.Unlock()

	if hasForeground {
		// Kill foreground prism only
		log.Printf("Ctrl+C: killing foreground prism: %s", foregroundName)
		if err := sh.supervisor.killPrism(foregroundName); err != nil {
			log.Printf("Failed to kill foreground prism: %v", err)
		}

		// Note: killPrism is async - handleChildExit will clean up
		// User can press Ctrl+C again to exit if no more prisms
		return false // Keep running, let signal loop process SIGCHLD
	} else {
		// No prisms running, shutdown prismctl
		log.Printf("Ctrl+C: no prisms running, shutting down")
		sh.handleShutdown(unix.SIGINT)
		return true // Exit signal loop
	}
}

func (sh *signalHandler) handleShutdown(sig os.Signal) {
	log.Printf("Received %s, shutting down gracefully", sig)
	sh.supervisor.shutdown()
}

func (sh *signalHandler) handleSIGWINCH() {
	sh.supervisor.propagateResize()
}

func (sh *signalHandler) stop() {
	signal.Stop(sh.sigCh)
	close(sh.sigCh)
}
