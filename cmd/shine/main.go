package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/starbased-co/shine/pkg/config"
	"github.com/starbased-co/shine/pkg/panel"
	"github.com/starbased-co/shine/pkg/prism"
)

func main() {
	// Load configuration
	configPath := config.DefaultConfigPath()
	cfg := config.LoadOrDefault(configPath)

	fmt.Printf("✨ Shine - Hyprland Layer Shell TUI Toolkit\n")
	fmt.Printf("Configuration: %s\n\n", configPath)

	// Initialize prism manager with configured search paths
	prismMgr := prism.NewManager(cfg.Core.GetPaths())

	// Create panel manager
	panelMgr := panel.NewManager()

	// Launch all enabled prisms (unified treatment)
	for name, prismCfg := range cfg.Prisms {
		if prismCfg == nil || !prismCfg.Enabled {
			continue
		}

		if err := launchPrism(prismMgr, panelMgr, name, prismCfg); err != nil {
			log.Printf("Failed to launch prism %s: %v", name, err)
			continue
		}

		// Wait for each prism to fully start before launching the next
		// This ensures single-instance mode can detect existing instances
		time.Sleep(500 * time.Millisecond)
	}

	// List running panels
	panels := panelMgr.List()
	if len(panels) == 0 {
		fmt.Println("\nNo prisms enabled. Edit your config to enable prisms.")
		fmt.Printf("Config location: %s\n", configPath)
		os.Exit(0)
	}

	fmt.Printf("\nRunning %d prism(s): %v\n", len(panels), panels)
	fmt.Println("Press Ctrl+C to stop all prisms")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal or panels to exit
	go panelMgr.Wait()

	<-sigChan
	fmt.Println("\n\nShutting down...")

	// Stop all panels
	for _, name := range panelMgr.List() {
		fmt.Printf("Stopping %s...\n", name)
		if err := panelMgr.Stop(name); err != nil {
			log.Printf("Error stopping %s: %v", name, err)
		}
	}

	fmt.Println("Goodbye!")
}

// launchPrism discovers and launches a prism with the given configuration
func launchPrism(
	prismMgr *prism.Manager,
	panelMgr *panel.Manager,
	name string,
	cfg *config.PrismConfig,
) error {
	// Find prism binary using discovery manager
	prismPath, err := prismMgr.FindPrism(name, cfg)
	if err != nil {
		return fmt.Errorf("failed to find prism binary: %w", err)
	}

	fmt.Printf("Launching %s (%s)...\n", name, prismPath)

	// Convert prism config to panel config
	panelCfg := cfg.ToPanelConfig()

	// Launch via panel manager
	instance, err := panelMgr.Launch(name, panelCfg, prismPath)
	if err != nil {
		return fmt.Errorf("failed to launch: %w", err)
	}

	// Report launch status
	if instance.WindowID != "" {
		fmt.Printf("  ✓ %s launched (Window ID: %s)\n", name, instance.WindowID)
	} else if instance.Command != nil && instance.Command.Process != nil {
		fmt.Printf("  ✓ %s launched (PID: %d)\n", name, instance.Command.Process.Pid)
	} else {
		fmt.Printf("  ✓ %s launched\n", name)
	}

	if panelCfg.ListenSocket != "" {
		fmt.Printf("  ✓ Remote control: %s\n", panelCfg.ListenSocket)
	}

	return nil
}
