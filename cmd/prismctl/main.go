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


func main() {
	// Handle help requests before anything else
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		if arg == "-h" || arg == "--help" {
			showHelp("")
			os.Exit(0)
		}
		if arg == "help" {
			topic := ""
			if len(os.Args) >= 3 {
				topic = os.Args[2]
			}
			showHelp(topic)
			os.Exit(0)
		}
		if arg == "--json" && len(os.Args) >= 3 && os.Args[2] == "help" {
			topic := ""
			if len(os.Args) >= 4 {
				topic = os.Args[3]
			}
			if err := helpJSON(topic); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
			os.Exit(0)
		}
	}

	// Setup logging to file
	if err := setupLogging(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to setup logging: %v\n", err)
		os.Exit(1)
	}

	// Parse arguments
	if len(os.Args) < 2 {
		showHelp("")
		os.Exit(1)
	}

	prismName := os.Args[1]
	instanceName := prismName

	// Optional instance name for socket identification
	if len(os.Args) >= 3 {
		instanceName = os.Args[2]
	}

	log.Printf("prismctl starting (prism: %s, instance: %s)", prismName, instanceName)

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
	ipcServer, err := newIPCServer(instanceName, sup)
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
