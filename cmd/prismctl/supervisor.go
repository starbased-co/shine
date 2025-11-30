package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

// prismState represents the current state of a prism instance
type prismState int

const (
	prismForeground prismState = iota // Currently visible
	prismBackground                   // Running in background
)

// prismInstance represents a single prism process
type prismInstance struct {
	name      string
	pid       int
	state     prismState
	ptyMaster *os.File // Child's PTY master FD
}

// supervisor manages the lifecycle of child prism processes
type supervisor struct {
	mu           sync.Mutex
	termState    *terminalState
	prismList    []prismInstance // MRU list: [0] = foreground, [1] = most recent background, etc.
	shutdownCh   chan struct{}
	childExitCh  chan childExit
	surface       *surfaceState // Current active surface (Real PTY ↔ foreground child PTY)
	surfaceCtx    context.Context
	surfaceCancel context.CancelFunc
	shuttingDown bool // Flag to prevent double-shutdown
	stateManager *StateManager // State management for mmap file
	notifyMgr    *NotificationManager // Notification manager for shined
	appPaths     map[string]string // App name → resolved binary path (multi-app mode)
}

// childExit represents a child process exit event
type childExit struct {
	pid      int
	exitCode int
}

// newSupervisor creates a new supervisor instance
func newSupervisor(termState *terminalState, stateMgr *StateManager, notifyMgr *NotificationManager) *supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &supervisor{
		termState:     termState,
		prismList:     make([]prismInstance, 0),
		shutdownCh:    make(chan struct{}),
		childExitCh:   make(chan childExit, 1),
		surface:       nil,
		surfaceCtx:    ctx,
		surfaceCancel: cancel,
		stateManager:  stateMgr,
		notifyMgr:     notifyMgr,
		appPaths:      make(map[string]string),
	}
}

// findPrism returns the index of a prism by name, or -1 if not found
func (s *supervisor) findPrism(name string) int {
	for i, p := range s.prismList {
		if p.name == name {
			return i
		}
	}
	return -1
}

// registerApp stores the resolved binary path for an app (used in multi-app mode)
func (s *supervisor) registerApp(name, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appPaths[name] = path
}

// startPrism launches the initial prism (wrapper for compatibility)
func (s *supervisor) startPrism(prismName string) error {
	return s.start(prismName)
}

// start implements idempotent launch/resume with three cases
func (s *supervisor) start(prismName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Case 1: Prism doesn't exist → launch fresh
	targetIdx := s.findPrism(prismName)
	if targetIdx == -1 {
		return s.launchAndForeground(prismName)
	}

	// Case 2: Already foreground → no-op
	if targetIdx == 0 {
		log.Printf("Prism %s already in foreground", prismName)
		return nil
	}

	// Case 3: In background → resume to foreground
	return s.resumeToForeground(targetIdx)
}

// launchAndForeground launches a new prism and brings it to foreground
// Assumes caller holds s.mu lock
func (s *supervisor) launchAndForeground(prismName string) error {
	var binaryPath string
	var err error

	// Check if we have a registered path for this app (multi-app mode)
	if path, ok := s.appPaths[prismName]; ok && path != "" {
		binaryPath = path
	}

	if binaryPath == "" {
		// Fall back to LookPath (legacy single-app mode)
		binaryPath, err = exec.LookPath(prismName)
		if err != nil {
			return fmt.Errorf("prism not found in PATH: %s (%w)", prismName, err)
		}
	}

	log.Printf("Launching new prism: %s (resolved to %s)", prismName, binaryPath)

	// Move current foreground to background if exists
	if len(s.prismList) > 0 {
		log.Printf("Moving current foreground to background (PID %d)", s.prismList[0].pid)
		s.prismList[0].state = prismBackground
	}

	// Allocate PTY pair for new prism
	ptyMaster, ptySlave, err := allocatePTY()
	if err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	// Sync terminal size from real terminal to child PTY
	if err := syncTerminalSize(int(os.Stdin.Fd()), int(ptyMaster.Fd())); err != nil {
		closePTY(ptyMaster)
		ptySlave.Close()
		return fmt.Errorf("failed to sync terminal size: %w", err)
	}

	// Reset terminal state (CRITICAL!)
	log.Printf("Resetting terminal state")
	if err := s.termState.resetTerminalState(); err != nil {
		log.Printf("Warning: failed to reset terminal state: %v", err)
	}

	// Stabilization delay
	time.Sleep(10 * time.Millisecond)

	// Fork/exec new prism with PTY as controlling terminal
	cmd := exec.Command(binaryPath)
	cmd.Stdin = ptySlave
	cmd.Stdout = ptySlave
	cmd.Stderr = ptySlave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true, // Create new session
		Setctty: true, // Make PTY controlling terminal
		Ctty:    0,    // FD 0 (stdin) in child process
	}

	if err := cmd.Start(); err != nil {
		closePTY(ptyMaster)
		ptySlave.Close()
		return fmt.Errorf("failed to start prism: %w", err)
	}

	// Close slave in parent - child keeps it open
	ptySlave.Close()

	pid := cmd.Process.Pid
	log.Printf("Prism started: %s (PID %d) with PTY", prismName, pid)

	// Add to front of MRU list
	newInstance := prismInstance{
		name:      prismName,
		pid:       pid,
		state:     prismForeground,
		ptyMaster: ptyMaster,
	}
	s.prismList = append([]prismInstance{newInstance}, s.prismList...)

	// Start surface to new foreground prism
	if err := s.activateSurfaceToForeground(); err != nil {
		log.Printf("Warning: failed to start surface: %v", err)
	}

	// Update state: new prism started in foreground
	if s.stateManager != nil {
		s.stateManager.OnPrismStarted(prismName, pid, true)
	}

	// Notify shined of prism start
	if s.notifyMgr != nil {
		s.notifyMgr.OnPrismStarted(prismName, pid)
	}

	return nil
}

// resumeToForeground brings a background prism to foreground
func (s *supervisor) resumeToForeground(targetIdx int) error {
	target := s.prismList[targetIdx]
	log.Printf("Bringing prism %s (PID %d) to foreground", target.name, target.pid)

	// Move current foreground to background
	if len(s.prismList) > 0 && targetIdx != 0 {
		log.Printf("Moving current foreground to background (PID %d)", s.prismList[0].pid)
		s.prismList[0].state = prismBackground
	}

	// Reset terminal state (CRITICAL!)
	log.Printf("Resetting terminal state")
	if err := s.termState.resetTerminalState(); err != nil {
		log.Printf("Warning: failed to reset terminal state: %v", err)
	}

	// Stabilization delay
	time.Sleep(10 * time.Millisecond)

	// Sync terminal size to target PTY
	if err := syncTerminalSize(int(os.Stdin.Fd()), int(target.ptyMaster.Fd())); err != nil {
		log.Printf("Warning: failed to sync terminal size: %v", err)
	}

	// Move target to position [0] and reorder MRU list
	// Remove from current position
	s.prismList = append(s.prismList[:targetIdx], s.prismList[targetIdx+1:]...)

	// Update state and add to front
	target.state = prismForeground
	s.prismList = append([]prismInstance{target}, s.prismList...)

	log.Printf("Prism %s brought to foreground", target.name)

	// Hot-swap surface to new foreground prism (includes screen clear)
	if err := s.swapSurface(); err != nil {
		log.Printf("Warning: failed to swap surface: %v", err)
	}

	// Send SIGWINCH after surface is connected to trigger redraw
	if err := unix.Kill(target.pid, unix.SIGWINCH); err != nil {
		log.Printf("Warning: failed to send SIGWINCH for redraw: %v", err)
	}

	// Update state: foreground changed
	if s.stateManager != nil {
		s.stateManager.OnForegroundChanged(target.name)
	}

	// Notify shined of surface switch
	if s.notifyMgr != nil && len(s.prismList) > 1 {
		// Previous foreground is now at index [1]
		previousFg := s.prismList[1].name
		s.notifyMgr.OnSurfaceSwitched(previousFg, target.name)
	}

	return nil
}

// killPrism terminates a prism by name with auto-resume
func (s *supervisor) killPrism(prismName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetIdx := s.findPrism(prismName)
	if targetIdx == -1 {
		return fmt.Errorf("prism not found: %s", prismName)
	}

	target := s.prismList[targetIdx]
	pid := target.pid

	log.Printf("Killing prism %s (PID %d)", prismName, pid)

	// Send SIGTERM - handleChildExit will do cleanup when SIGCHLD arrives
	if err := unix.Kill(pid, unix.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Don't wait for exit - that blocks the signal handler!
	// handleChildExit will clean up when SIGCHLD arrives
	return nil
}

// handleChildExit processes child exit events
func (s *supervisor) handleChildExit(pid, exitCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Find the exited prism in MRU list
	exitedIdx := -1
	for i, p := range s.prismList {
		if p.pid == pid {
			exitedIdx = i
			break
		}
	}

	if exitedIdx == -1 {
		log.Printf("Warning: received exit for unknown PID %d", pid)
		return
	}

	exited := s.prismList[exitedIdx]
	log.Printf("Child exited: %s (PID %d, code %d)", exited.name, pid, exitCode)

	// Close PTY master
	if err := closePTY(exited.ptyMaster); err != nil {
		log.Printf("Warning: failed to close PTY master: %v", err)
	}

	// Notify any waiting kill operation
	select {
	case s.childExitCh <- childExit{pid: pid, exitCode: exitCode}:
		log.Printf("Sent exit event to childExitCh for PID %d", pid)
	default:
		log.Printf("WARNING: Failed to send exit event - channel full or no listener for PID %d", pid)
	}

	// If foreground exited, clean up surface and reset terminal state
	if exitedIdx == 0 {
		// Stop surface after foreground exit
		if s.surface != nil {
			deactivateSurface(s.surface)
			s.surface = nil
		}

		if err := s.termState.resetTerminalState(); err != nil {
			log.Printf("Error resetting terminal state after child exit: %v", err)
		}
	}

	// Remove from MRU list
	s.prismList = append(s.prismList[:exitedIdx], s.prismList[exitedIdx+1:]...)

	// Update state: prism stopped
	if s.stateManager != nil {
		s.stateManager.OnPrismStopped(exited.name)
	}

	// Notify shined of prism exit (stopped or crashed)
	if s.notifyMgr != nil {
		if exitCode == 0 {
			s.notifyMgr.OnPrismStopped(exited.name, exitCode)
		} else {
			// Non-zero exit is a crash
			s.notifyMgr.OnPrismCrashed(exited.name, exitCode, 0)
		}
	}

	// Check if prismList is now empty - auto-shutdown
	if len(s.prismList) == 0 {
		log.Printf("Last prism exited, initiating shutdown")
		go s.shutdown() // Async to avoid deadlock with mutex
		return
	}

	// If foreground crashed AND others exist, bring next to foreground
	if exitedIdx == 0 && len(s.prismList) > 0 {
		time.Sleep(10 * time.Millisecond)

		next := s.prismList[0]

		// Sync terminal size to next prism
		if err := syncTerminalSize(int(os.Stdin.Fd()), int(next.ptyMaster.Fd())); err != nil {
			log.Printf("Warning: failed to sync terminal size: %v", err)
		}

		// Send SIGWINCH to trigger redraw
		unix.Kill(next.pid, unix.SIGWINCH)

		s.prismList[0].state = prismForeground

		log.Printf("Auto-brought to foreground after crash: %s (PID %d)", next.name, next.pid)
	}
}

// propagateResize propagates SIGWINCH to ALL child PTYs (not just foreground)
func (s *supervisor) propagateResize() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.prismList) == 0 {
		return
	}

	// Get current terminal size from Real PTY
	realWinsize, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		log.Printf("Warning: failed to get Real PTY size: %v", err)
		return
	}

	log.Printf("Propagating resize to %d prisms: %dx%d", len(s.prismList), realWinsize.Col, realWinsize.Row)

	// Sync terminal size to ALL child PTYs and send SIGWINCH
	for _, prism := range s.prismList {
		// Sync terminal size to child PTY
		if err := unix.IoctlSetWinsize(int(prism.ptyMaster.Fd()), unix.TIOCSWINSZ, realWinsize); err != nil {
			log.Printf("Warning: failed to sync size to %s (PID %d): %v", prism.name, prism.pid, err)
			continue
		}

		// Send SIGWINCH to child process
		if err := unix.Kill(prism.pid, unix.SIGWINCH); err != nil {
			log.Printf("Warning: failed to send SIGWINCH to %s (PID %d): %v", prism.name, prism.pid, err)
		}
	}
}

// shutdown performs graceful shutdown
func (s *supervisor) shutdown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Idempotency: prevent double-shutdown
	if s.shuttingDown {
		log.Printf("Shutdown already in progress, ignoring")
		return
	}
	s.shuttingDown = true

	log.Printf("Supervisor shutdown initiated")

	// Stop surface during shutdown
	if s.surface != nil {
		deactivateSurface(s.surface)
		s.surface = nil
	}

	// Cancel surface context
	if s.surfaceCancel != nil {
		s.surfaceCancel()
	}

	// Signal shutdown
	close(s.shutdownCh)

	// Kill all prisms in the list
	for _, prism := range s.prismList {
		log.Printf("Terminating prism %s (PID %d)", prism.name, prism.pid)

		// Try graceful shutdown first
		if err := unix.Kill(prism.pid, unix.SIGTERM); err != nil {
			log.Printf("Warning: failed to send SIGTERM to %s: %v", prism.name, err)
		}
	}

	// Wait briefly for graceful exits (match Kitty's signal delivery timing)
	time.Sleep(20 * time.Millisecond)

	// Force kill any remaining prisms and close PTYs
	for _, prism := range s.prismList {
		if err := unix.Kill(prism.pid, unix.SIGKILL); err == nil {
			log.Printf("Sent SIGKILL to %s (PID %d)", prism.name, prism.pid)
		}

		// Close PTY master
		if err := closePTY(prism.ptyMaster); err != nil {
			log.Printf("Warning: failed to close PTY master for %s: %v", prism.name, err)
		}
	}

	// Restore terminal to original state
	if err := s.termState.restoreTerminalState(); err != nil {
		log.Printf("Warning: failed to restore terminal state: %v", err)
	}

	log.Printf("Supervisor shutdown complete")

	// Print friendly goodbye to user
	fmt.Println("[ ] Exiting... ")
}

// isShuttingDown checks if shutdown is in progress
func (s *supervisor) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shuttingDown
}

// activateSurfaceToForeground starts surface to current foreground prism
func (s *supervisor) activateSurfaceToForeground() error {
	if len(s.prismList) == 0 {
		return fmt.Errorf("no prisms to connect to")
	}

	foreground := s.prismList[0]
	if foreground.state != prismForeground {
		return fmt.Errorf("internal error: position [0] not foreground")
	}

	// Stop existing surface if running
	if s.surface != nil {
		// Force the surface to stop by closing the read side
		// This makes io.Copy return immediately
		deactivateSurface(s.surface)
		s.surface = nil
		log.Printf("Stopped previous surface before starting new one")
	}

	// Start new surface: os.Stdin (Real PTY slave) ↔ foreground.ptyMaster
	surface, err := activateSurface(s.surfaceCtx, os.Stdin, foreground.ptyMaster)
	if err != nil {
		return fmt.Errorf("failed to start surface: %w", err)
	}

	s.surface = surface
	log.Printf("Surface started to foreground prism: %s (PID %d)", foreground.name, foreground.pid)

	return nil
}

// swapSurface stops current surface and starts new one to foreground
func (s *supervisor) swapSurface() error {
	startTime := time.Now()

	// Stop current surface
	if s.surface != nil {
		deactivateSurface(s.surface)
		s.surface = nil
	}

	// Clear screen AFTER stopping old surface but BEFORE starting new one
	// This ensures no race between clear and buffered output from background prism
	// CSI 2 J = clear screen, CSI H = cursor home, CSI 0 m = reset all attributes
	os.Stdout.WriteString("\x1b[2J\x1b[H\x1b[0m")

	// Start new surface to foreground
	if err := s.activateSurfaceToForeground(); err != nil {
		return err
	}

	swapLatency := time.Since(startTime)
	log.Printf("Surface swap completed in %v", swapLatency)

	if swapLatency > 50*time.Millisecond {
		log.Printf("Warning: swap latency exceeded 50ms target: %v", swapLatency)
	}

	return nil
}
