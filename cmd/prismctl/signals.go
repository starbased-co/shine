package main

import (
	"log"
	"os"
	"os/signal"

	"golang.org/x/sys/unix"
)

// signalHandler manages all signal handling for prismctl
type signalHandler struct {
	sigCh      chan os.Signal
	supervisor *supervisor
}

// newSignalHandler creates and configures signal handling
func newSignalHandler(sup *supervisor) *signalHandler {
	sh := &signalHandler{
		sigCh:      make(chan os.Signal, 10),
		supervisor: sup,
	}

	// Register for all signals we need to handle
	signal.Notify(sh.sigCh,
		unix.SIGCHLD,  // Child process state change
		unix.SIGTERM,  // Termination request
		unix.SIGINT,   // Interrupt (Ctrl+C)
		unix.SIGHUP,   // Hangup (Kitty panel close)
		unix.SIGWINCH, // Window resize
	)

	return sh
}

// run processes signals in a loop until shutdown
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

// handleSIGCHLD reaps zombie processes and handles child exits
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

// handleShutdown performs graceful shutdown
func (sh *signalHandler) handleShutdown(sig os.Signal) {
	log.Printf("Received %s, shutting down gracefully", sig)
	sh.supervisor.shutdown()
}

// handleSIGWINCH forwards window resize to ALL child processes
func (sh *signalHandler) handleSIGWINCH() {
	sh.supervisor.propagateResize()
}

// stop stops signal handling
func (sh *signalHandler) stop() {
	signal.Stop(sh.sigCh)
	close(sh.sigCh)
}
