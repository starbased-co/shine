package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const version = "0.1.0"

func usage() {
	showHelp("")
}

func main() {
	// Parse flags
	configPath := flag.String("config", "", "Path to prism.toml")
	showVersion := flag.Bool("version", false, "Print version and exit")
	helpTopic := flag.String("help", "", "Show help for a topic")
	jsonOutput := flag.Bool("json", false, "Output help in JSON format")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("shinectl v%s\n", version)
		os.Exit(0)
	}

	// Handle help command
	if *helpTopic != "" || flag.NArg() > 0 && flag.Arg(0) == "help" {
		topic := *helpTopic
		if topic == "" && flag.NArg() > 1 {
			topic = flag.Arg(1)
		}

		if *jsonOutput {
			if err := helpJSON(topic); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				os.Exit(1)
			}
		} else {
			showHelp(topic)
		}
		os.Exit(0)
	}

	// Setup logging
	logFile := setupLogging()
	defer logFile.Close()

	log.Printf("shinectl v%s starting", version)

	// Determine config path
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = DefaultConfigPath()
	}

	log.Printf("Loading configuration from: %s", cfgPath)

	// Load config
	config := LoadConfigOrDefault(cfgPath)
	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Loaded configuration with %d prism(s)", len(config.Prisms))

	// Create panel manager
	pm, err := NewPanelManager()
	if err != nil {
		log.Fatalf("Failed to create panel manager: %v", err)
	}

	// Spawn initial panels
	if err := spawnConfiguredPanels(pm, config); err != nil {
		log.Fatalf("Failed to spawn panels: %v", err)
	}

	// Setup signal handlers
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	// Health monitoring ticker
	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()

	log.Println("shinectl is running (Ctrl+C to stop)")

	// Main event loop
	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				log.Println("Received SIGHUP - reloading configuration")
				if err := reloadConfig(pm, cfgPath); err != nil {
					log.Printf("Failed to reload config: %v", err)
				}

			case syscall.SIGTERM, syscall.SIGINT:
				log.Println("Received shutdown signal - stopping all panels")
				pm.Shutdown()
				log.Println("shinectl stopped")
				return
			}

		case <-healthTicker.C:
			// Periodic health check
			pm.MonitorPanels()
		}
	}
}

// setupLogging configures logging to file
func setupLogging() *os.File {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	logDir := filepath.Join(home, ".local", "share", "shine", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logPath := filepath.Join(logDir, "shinectl.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Log to both stdout and file
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return logFile
}

// spawnConfiguredPanels spawns panels for all prisms in config
func spawnConfiguredPanels(pm *PanelManager, config *Config) error {
	for _, prism := range config.Prisms {
		// Use prism name as instance name for kitty --instance-group
		instanceName := prism.Name

		log.Printf("Spawning panel for prism: %s (instance: %s)", prism.Name, instanceName)

		panel, err := pm.SpawnPanel(&prism, instanceName)
		if err != nil {
			return fmt.Errorf("failed to spawn panel for %s: %w", prism.Name, err)
		}

		log.Printf("Panel spawned successfully: %s (socket: %s)",
			panel.Instance, panel.SocketPath)
	}

	return nil
}

// reloadConfig reloads configuration and updates panels accordingly
func reloadConfig(pm *PanelManager, configPath string) error {
	log.Println("Reloading configuration...")

	// Load new config
	newConfig := LoadConfigOrDefault(configPath)
	if err := newConfig.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Get current panels
	currentPanels := pm.ListPanels()

	// Build map of current prism names
	currentPrisms := make(map[string]*Panel)
	for _, panel := range currentPanels {
		currentPrisms[panel.Name] = panel
	}

	// Build map of new prism names
	newPrisms := make(map[string]*PrismEntry)
	for _, prism := range newConfig.Prisms {
		newPrisms[prism.Name] = &prism
	}

	// Remove panels that are no longer in config
	for name, panel := range currentPrisms {
		if _, exists := newPrisms[name]; !exists {
			log.Printf("Removing panel for prism %s (no longer in config)", name)
			if err := pm.KillPanel(panel.Instance); err != nil {
				log.Printf("Failed to kill panel %s: %v", panel.Instance, err)
			}
		}
	}

	// Add new panels
	for name, prism := range newPrisms {
		if _, exists := currentPrisms[name]; !exists {
			// Use prism name as instance name
			instanceName := prism.Name

			log.Printf("Adding new panel for prism: %s (instance: %s)", name, instanceName)

			panel, err := pm.SpawnPanel(prism, instanceName)
			if err != nil {
				log.Printf("Failed to spawn panel for %s: %v", name, err)
				continue
			}

			log.Printf("New panel spawned: %s", panel.Instance)
		}
	}

	log.Println("Configuration reloaded successfully")
	return nil
}
