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

	"github.com/starbased-co/shine/pkg/config"
	"github.com/starbased-co/shine/pkg/paths"
)

const version = "0.1.0"

func usage() {
	showHelp("")
}

func main() {
	configPath := flag.String("config", "", "Path to prism.toml")
	showVersion := flag.Bool("version", false, "Print version and exit")
	helpTopic := flag.String("help", "", "Show help for a topic")
	flag.Usage = usage
	flag.Parse()

	if *showVersion {
		fmt.Printf("shined v%s\n", version)
		os.Exit(0)
	}

	if *helpTopic != "" || flag.NArg() > 0 && flag.Arg(0) == "help" {
		topic := *helpTopic
		if topic == "" && flag.NArg() > 1 {
			topic = flag.Arg(1)
		}

		showHelp(topic)
		os.Exit(0)
	}

	logFile := setupLogging()
	defer logFile.Close()

	log.Printf("shined v%s starting", version)

	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = config.DefaultConfigPath()
	}

	log.Printf("Loading configuration from: %s", cfgPath)

	// Load config using pkg/config (with prism discovery)
	pkgCfg, err := config.Load(cfgPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := pkgCfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Convert to PrismEntry slice for shined
	prismEntries := make([]*PrismEntry, 0)
	for name, pc := range pkgCfg.Prisms {
		if !pc.Enabled || pc.ResolvedPath == "" {
			log.Printf("Skipping prism %q: enabled=%v, resolved=%q", name, pc.Enabled, pc.ResolvedPath)
			continue
		}

		entry := &PrismEntry{
			PrismConfig: pc,
			// Restart policies default to "no"
			Restart:      "no",
			RestartDelay: "1s",
			MaxRestarts:  0,
		}

		if err := entry.ValidateRestartPolicy(); err != nil {
			log.Fatalf("Invalid restart policy for prism %q: %v", name, err)
		}

		prismEntries = append(prismEntries, entry)
	}

	log.Printf("Loaded configuration with %d prism(s)", len(prismEntries))

	stateMgr, err := newStateManager()
	if err != nil {
		log.Fatalf("Failed to create state manager: %v", err)
	}
	defer stateMgr.Close()

	pm, err := NewPanelManager()
	if err != nil {
		log.Fatalf("Failed to create panel manager: %v", err)
	}

	if err := startRPCServer(pm, stateMgr, cfgPath); err != nil {
		log.Fatalf("Failed to start RPC server: %v", err)
	}
	defer stopRPCServer()

	if err := spawnConfiguredPanels(pm, prismEntries, stateMgr); err != nil {
		log.Fatalf("Failed to spawn panels: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()

	log.Println("shined is running (Ctrl+C to stop)")

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
				stopRPCServer()
				pm.Shutdown()
				stateMgr.Remove() // Clean up state file on shutdown
				log.Println("shined stopped")
				return
			}

		case <-healthTicker.C:
			pm.MonitorPanels()
		}
	}
}

func setupLogging() *os.File {
	logDir := paths.LogDir()
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create log directory: %v", err)
	}

	logPath := filepath.Join(logDir, "shined.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file: %v", err)
	}

	// Log to both stdout and file
	log.SetOutput(logFile)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return logFile
}

func spawnConfiguredPanels(pm *PanelManager, entries []*PrismEntry, stateMgr *StateManager) error {
	for _, entry := range entries {
		instanceName := entry.Name

		log.Printf("Spawning panel for prism: %s (instance: %s, binary: %s)",
			entry.Name, instanceName, entry.ResolvedPath)

		panel, err := pm.SpawnPanel(entry, instanceName)
		if err != nil {
			return fmt.Errorf("failed to spawn panel for %s: %w", entry.Name, err)
		}

		// Update state with spawned panel
		healthy := pm.CheckHealth(panel)
		stateMgr.OnPanelSpawned(panel.Instance, panel.Name, panel.PID, healthy)

		log.Printf("Panel spawned successfully: %s (socket: %s)",
			panel.Instance, panel.SocketPath)
	}

	return nil
}

func reloadConfig(pm *PanelManager, configPath string) error {
	log.Println("Reloading configuration...")

	// Load new config using pkg/config
	pkgCfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := pkgCfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Convert to PrismEntry slice
	newEntries := make([]*PrismEntry, 0)
	for name, pc := range pkgCfg.Prisms {
		if !pc.Enabled || pc.ResolvedPath == "" {
			continue
		}

		entry := &PrismEntry{
			PrismConfig:  pc,
			Restart:      "no",
			RestartDelay: "1s",
			MaxRestarts:  0,
		}

		if err := entry.ValidateRestartPolicy(); err != nil {
			log.Printf("Invalid restart policy for prism %q: %v", name, err)
			continue
		}

		newEntries = append(newEntries, entry)
	}

	currentPanels := pm.ListPanels()

	// Build maps of current and new prism names and compare
	currentPrisms := make(map[string]*Panel)
	for _, panel := range currentPanels {
		currentPrisms[panel.Name] = panel
	}

	newPrisms := make(map[string]*PrismEntry)
	for _, entry := range newEntries {
		newPrisms[entry.Name] = entry
	}

  // kill   = {x ∈ current : x ∉ new}
	for name, panel := range currentPrisms {
		if _, exists := newPrisms[name]; !exists {
			log.Printf("Removing panel for prism %s (no longer in config)", name)
			if err := pm.KillPanel(panel.Instance); err != nil {
				log.Printf("Failed to kill panel %s: %v", panel.Instance, err)
			}
		}
	}

  // spawn  = {x ∈ new : x ∉ current}
	for name, entry := range newPrisms {
		if _, exists := currentPrisms[name]; !exists {
			instanceName := entry.Name

			log.Printf("Adding new panel for prism: %s (instance: %s)", name, instanceName)

			panel, err := pm.SpawnPanel(entry, instanceName)
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
