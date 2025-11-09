package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

// setupLogging configures logging to file instead of stderr
func setupLogging() error {
	// Create log directory
	logDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "shine", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open log file (append mode)
	logPath := filepath.Join(logDir, "prismctl.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	// Set log output to file
	log.SetOutput(logFile)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetPrefix("[prismctl] ")

	return nil
}

const usage = `prismctl - Supervisor for Shine prism processes

Usage:
  prismctl <prism-name> [component-name]

Arguments:
  prism-name      Name of the prism binary to run (e.g., shine-clock)
  component-name  Optional component identifier for IPC socket naming (default: same as prism-name)

Description:
  prismctl manages the lifecycle of a single prism process, providing:
  - Terminal state management and cleanup
  - Hot-swap capability via IPC
  - Signal handling (SIGCHLD, SIGTERM, SIGWINCH)
  - Crash recovery

Examples:
  prismctl shine-clock
  prismctl shine-spotify music-panel

IPC Socket:
  The IPC socket is created at:
  /run/user/<uid>/shine/prism-<component>.<<pid>.sock

IPC Commands:
  {"action":"start","prism":"shine-spotify"}  # Start/resume prism (idempotent)
  {"action":"kill","prism":"shine-clock"}     # Kill prism (auto-resumes next)
  {"action":"status"}                         # Query current status
  {"action":"stop"}                           # Graceful shutdown

For more information, see the Shine documentation.
`

func main() {
	// Setup logging to file
	if err := setupLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}

	// Parse arguments
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	prismName := os.Args[1]
	componentName := prismName

	// Optional component name for socket identification
	if len(os.Args) >= 3 {
		componentName = os.Args[2]
	}

	// Show help if requested
	if prismName == "-h" || prismName == "--help" || prismName == "help" {
		fmt.Print(usage)
		os.Exit(0)
	}

	log.Printf("prismctl starting (prism: %s, component: %s)", prismName, componentName)

	// Initialize terminal state management
	termState, err := newTerminalState()
	if err != nil {
		log.Fatalf("Failed to initialize terminal state: %v", err)
	}
	log.Printf("Terminal state saved")

	// Create supervisor
	sup := newSupervisor(termState)

	// Setup signal handling
	sigHandler := newSignalHandler(sup)
	defer sigHandler.stop()

	// Start IPC server
	ipcServer, err := newIPCServer(componentName, sup)
	if err != nil {
		log.Fatalf("Failed to start IPC server: %v", err)
	}
	defer ipcServer.stop()

	// Start IPC server in background
	go ipcServer.serve()

	// Start initial prism
	if err := sup.startPrism(prismName); err != nil {
		log.Fatalf("Failed to start prism: %v", err)
	}

	// Run signal handler (blocks until shutdown)
	log.Printf("prismctl running (PID %d)", os.Getpid())
	sigHandler.run()

	log.Printf("prismctl exiting")
}
