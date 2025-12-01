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

type prismState int

const (
	prismForeground prismState = iota
	prismBackground
)

type prismInstance struct {
	name      string
	pid       int
	state     prismState
	ptyMaster *os.File
}

type supervisor struct {
	mu           sync.Mutex
	termState    *terminalState
	prismList    []prismInstance // MRU list: [0] = foreground, [1] = most recent background, etc.
	shutdownCh   chan struct{}
	childExitCh  chan childExit
	mirror       *mirrorState
	mirrorCtx    context.Context
	mirrorCancel context.CancelFunc
	shuttingDown bool
	stateManager *StateManager
	notifyMgr    *NotificationManager
	appPaths     map[string]string // App name → resolved binary path
}

type childExit struct {
	pid      int
	exitCode int
}

func newSupervisor(termState *terminalState, stateMgr *StateManager, notifyMgr *NotificationManager) *supervisor {
	ctx, cancel := context.WithCancel(context.Background())
	return &supervisor{
		termState:     termState,
		prismList:     make([]prismInstance, 0),
		shutdownCh:    make(chan struct{}),
		childExitCh:   make(chan childExit, 1),
		mirror:       nil,
		mirrorCtx:    ctx,
		mirrorCancel: cancel,
		stateManager:  stateMgr,
		notifyMgr:     notifyMgr,
		appPaths:      make(map[string]string),
	}
}

func (s *supervisor) findPrism(name string) int {
	for i, p := range s.prismList {
		if p.name == name {
			return i
		}
	}
	return -1
}

func (s *supervisor) registerApp(name, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.appPaths[name] = path
}

func (s *supervisor) startPrism(prismName string) error {
	return s.start(prismName)
}

// start implements idempotent launch/resume with three cases
func (s *supervisor) start(prismName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	targetIdx := s.findPrism(prismName)
	if targetIdx == -1 {
		return s.launchAndForeground(prismName)
	}

	if targetIdx == 0 {
		log.Printf("Prism %s already in foreground", prismName)
		return nil
	}

	return s.resumeToForeground(targetIdx)
}

// launchAndForeground launches a new prism and brings it to foreground
// Assumes caller holds s.mu lock
func (s *supervisor) launchAndForeground(prismName string) error {
	var binaryPath string
	var err error

	if path, ok := s.appPaths[prismName]; ok && path != "" {
		binaryPath = path
	}

	if binaryPath == "" {
		binaryPath, err = exec.LookPath(prismName)
		if err != nil {
			return fmt.Errorf("prism not found in PATH: %s (%w)", prismName, err)
		}
	}

	log.Printf("Launching new prism: %s (resolved to %s)", prismName, binaryPath)

	if len(s.prismList) > 0 {
		old := s.prismList[0]
		log.Printf("Suspending current foreground %s (PID %d)", old.name, old.pid)
		if err := unix.Kill(old.pid, unix.SIGSTOP); err != nil {
			log.Printf("Warning: failed to SIGSTOP %s: %v", old.name, err)
		}
		s.prismList[0].state = prismBackground
	}

	ptyMaster, ptySlave, err := allocatePTY()
	if err != nil {
		return fmt.Errorf("failed to allocate PTY: %w", err)
	}

	if err := syncTerminalSize(int(os.Stdin.Fd()), int(ptyMaster.Fd())); err != nil {
		closePTY(ptyMaster)
		ptySlave.Close()
		return fmt.Errorf("failed to sync terminal size: %w", err)
	}

	// CRITICAL: Reset terminal state
	log.Printf("Resetting terminal state")
	if err := s.termState.resetTerminalState(); err != nil {
		log.Printf("Warning: failed to reset terminal state: %v", err)
	}

	// Stabilization delay
	time.Sleep(10 * time.Millisecond)

	cmd := exec.Command(binaryPath)
	cmd.Stdin = ptySlave
	cmd.Stdout = ptySlave
	cmd.Stderr = ptySlave
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
		Ctty:    0,
	}

	if err := cmd.Start(); err != nil {
		closePTY(ptyMaster)
		ptySlave.Close()
		return fmt.Errorf("failed to start prism: %w", err)
	}

	ptySlave.Close()

	pid := cmd.Process.Pid
	log.Printf("Prism started: %s (PID %d) with PTY", prismName, pid)

	newInstance := prismInstance{
		name:      prismName,
		pid:       pid,
		state:     prismForeground,
		ptyMaster: ptyMaster,
	}
	s.prismList = append([]prismInstance{newInstance}, s.prismList...)

	if err := s.activateMirrorToForeground(); err != nil {
		log.Printf("Warning: failed to start mirror: %v", err)
	}

	if s.stateManager != nil {
		s.stateManager.OnPrismStarted(prismName, pid, true)
	}

	if s.notifyMgr != nil {
		s.notifyMgr.OnPrismStarted(prismName, pid)
	}

	return nil
}

func (s *supervisor) resumeToForeground(targetIdx int) error {
	target := s.prismList[targetIdx]
	log.Printf("Resuming prism %s (PID %d) to foreground", target.name, target.pid)

	if len(s.prismList) > 0 && targetIdx != 0 {
		old := s.prismList[0]
		log.Printf("Suspending current foreground %s (PID %d)", old.name, old.pid)
		if err := unix.Kill(old.pid, unix.SIGSTOP); err != nil {
			log.Printf("Warning: failed to SIGSTOP %s: %v", old.name, err)
		}
		s.prismList[0].state = prismBackground
	}

	// Resume the target prism
	if err := unix.Kill(target.pid, unix.SIGCONT); err != nil {
		log.Printf("Warning: failed to SIGCONT %s: %v", target.name, err)
	}

	log.Printf("Resetting terminal state")
	if err := s.termState.resetTerminalState(); err != nil {
		log.Printf("Warning: failed to reset terminal state: %v", err)
	}

	time.Sleep(10 * time.Millisecond)

	if err := syncTerminalSize(int(os.Stdin.Fd()), int(target.ptyMaster.Fd())); err != nil {
		log.Printf("Warning: failed to sync terminal size: %v", err)
	}

	s.prismList = append(s.prismList[:targetIdx], s.prismList[targetIdx+1:]...)

	target.state = prismForeground
	s.prismList = append([]prismInstance{target}, s.prismList...)

	log.Printf("Prism %s brought to foreground", target.name)

	if err := s.swapMirror(); err != nil {
		log.Printf("Warning: failed to swap mirror: %v", err)
	}

	if err := unix.Kill(target.pid, unix.SIGWINCH); err != nil {
		log.Printf("Warning: failed to send SIGWINCH for redraw: %v", err)
	}

	if s.stateManager != nil {
		s.stateManager.OnForegroundChanged(target.name)
	}

	if s.notifyMgr != nil && len(s.prismList) > 1 {
		previousFg := s.prismList[1].name
		s.notifyMgr.OnForegroundChanged(previousFg, target.name)
	}

	return nil
}

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

	// Resume first - suspended processes ignore SIGTERM
	unix.Kill(pid, unix.SIGCONT)

	if err := unix.Kill(pid, unix.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Don't wait - handleChildExit will clean up when SIGCHLD arrives
	return nil
}

func (s *supervisor) handleChildExit(pid, exitCode int) {
	s.mu.Lock()
	defer s.mu.Unlock()

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

	if err := closePTY(exited.ptyMaster); err != nil {
		log.Printf("Warning: failed to close PTY master: %v", err)
	}

	select {
	case s.childExitCh <- childExit{pid: pid, exitCode: exitCode}:
		log.Printf("Sent exit event to childExitCh for PID %d", pid)
	default:
		log.Printf("WARNING: Failed to send exit event - channel full or no listener for PID %d", pid)
	}

	if exitedIdx == 0 {
		if s.mirror != nil {
			deactivateMirror(s.mirror)
			s.mirror = nil
		}

		if err := s.termState.resetTerminalState(); err != nil {
			log.Printf("Error resetting terminal state after child exit: %v", err)
		}
	}

	s.prismList = append(s.prismList[:exitedIdx], s.prismList[exitedIdx+1:]...)

	if s.stateManager != nil {
		s.stateManager.OnPrismStopped(exited.name)
	}

	if s.notifyMgr != nil {
		if exitCode == 0 {
			s.notifyMgr.OnPrismStopped(exited.name, exitCode)
		} else {
			s.notifyMgr.OnPrismCrashed(exited.name, exitCode, 0)
		}
	}

	if len(s.prismList) == 0 {
		log.Printf("Last prism exited, initiating shutdown")
		go s.shutdown()
		return
	}

	// Auto-bring next to foreground if foreground exited
	if exitedIdx == 0 && len(s.prismList) > 0 {
		time.Sleep(10 * time.Millisecond)

		next := s.prismList[0]

		// Resume the suspended background prism
		if err := unix.Kill(next.pid, unix.SIGCONT); err != nil {
			log.Printf("Warning: failed to SIGCONT %s: %v", next.name, err)
		}

		if err := syncTerminalSize(int(os.Stdin.Fd()), int(next.ptyMaster.Fd())); err != nil {
			log.Printf("Warning: failed to sync terminal size: %v", err)
		}

		unix.Kill(next.pid, unix.SIGWINCH)

		s.prismList[0].state = prismForeground

		log.Printf("Auto-resumed to foreground: %s (PID %d)", next.name, next.pid)
	}
}

// propagateResize propagates SIGWINCH to ALL child PTYs (not just foreground)
func (s *supervisor) propagateResize() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.prismList) == 0 {
		return
	}

	realWinsize, err := unix.IoctlGetWinsize(int(os.Stdin.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		log.Printf("Warning: failed to get Real PTY size: %v", err)
		return
	}

	log.Printf("Propagating resize to %d prisms: %dx%d", len(s.prismList), realWinsize.Col, realWinsize.Row)

	for _, prism := range s.prismList {
		if err := unix.IoctlSetWinsize(int(prism.ptyMaster.Fd()), unix.TIOCSWINSZ, realWinsize); err != nil {
			log.Printf("Warning: failed to sync size to %s (PID %d): %v", prism.name, prism.pid, err)
			continue
		}

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

	if s.mirror != nil {
		deactivateMirror(s.mirror)
		s.mirror = nil
	}

	if s.mirrorCancel != nil {
		s.mirrorCancel()
	}

	close(s.shutdownCh)

	// Resume all suspended prisms first - they ignore SIGTERM while suspended
	for _, prism := range s.prismList {
		unix.Kill(prism.pid, unix.SIGCONT)
	}

	for _, prism := range s.prismList {
		log.Printf("Terminating prism %s (PID %d)", prism.name, prism.pid)

		if err := unix.Kill(prism.pid, unix.SIGTERM); err != nil {
			log.Printf("Warning: failed to send SIGTERM to %s: %v", prism.name, err)
		}
	}

	// Wait briefly for graceful exits (match Kitty's signal delivery timing)
	time.Sleep(20 * time.Millisecond)

	for _, prism := range s.prismList {
		if err := unix.Kill(prism.pid, unix.SIGKILL); err == nil {
			log.Printf("Sent SIGKILL to %s (PID %d)", prism.name, prism.pid)
		}

		if err := closePTY(prism.ptyMaster); err != nil {
			log.Printf("Warning: failed to close PTY master for %s: %v", prism.name, err)
		}
	}

	if err := s.termState.restoreTerminalState(); err != nil {
		log.Printf("Warning: failed to restore terminal state: %v", err)
	}

	log.Printf("Supervisor shutdown complete")

	fmt.Println("[ ] Exiting... ")
}

func (s *supervisor) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shuttingDown
}

func (s *supervisor) activateMirrorToForeground() error {
	if len(s.prismList) == 0 {
		return fmt.Errorf("no prisms to connect to")
	}

	foreground := s.prismList[0]
	if foreground.state != prismForeground {
		return fmt.Errorf("internal error: position [0] not foreground")
	}

	if s.mirror != nil {
		deactivateMirror(s.mirror)
		s.mirror = nil
		log.Printf("Stopped previous mirror before starting new one")
	}

	// os.Stdin (Real PTY slave) ↔ foreground.ptyMaster
	mirror, err := activateMirror(s.mirrorCtx, os.Stdin, foreground.ptyMaster)
	if err != nil {
		return fmt.Errorf("failed to start mirror: %w", err)
	}

	s.mirror = mirror
	log.Printf("Mirror started to foreground prism: %s (PID %d)", foreground.name, foreground.pid)

	return nil
}

func (s *supervisor) swapMirror() error {
	startTime := time.Now()

	if s.mirror != nil {
		deactivateMirror(s.mirror)
		s.mirror = nil
	}

	// Clear screen AFTER stopping old mirror but BEFORE starting new one
	// This ensures no race between clear and buffered output from background prism
	// CSI 2 J = clear screen, CSI H = cursor home, CSI 0 m = reset all attributes
	os.Stdout.WriteString("\x1b[2J\x1b[H\x1b[0m")

	if err := s.activateMirrorToForeground(); err != nil {
		return err
	}

	swapLatency := time.Since(startTime)
	log.Printf("Mirror swap completed in %v", swapLatency)

	if swapLatency > 50*time.Millisecond {
		log.Printf("Warning: swap latency exceeded 50ms target: %v", swapLatency)
	}

	return nil
}
