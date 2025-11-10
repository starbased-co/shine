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

	// Create panel manager
	panelMgr := panel.NewManager()

	// Create prism manager
	prismMgr := prism.NewManager(panelMgr)

	// Launch all enabled prisms
	for name, prismCfg := range cfg.Prisms {
		if prismCfg == nil || !prismCfg.Enabled {
			continue
		}

		fmt.Printf("Launching %s", name)
		if prismCfg.ResolvedPath != "" {
			fmt.Printf(" (%s)", prismCfg.ResolvedPath)
		}
		fmt.Println("...")

		if err := prismMgr.Launch(name, prismCfg); err != nil {
			log.Printf("Failed to launch prism %s: %v", name, err)
			continue
		}

		// Report launch status
		if instance, ok := prismMgr.Get(name); ok {
			if instance.Panel.WindowID != "" {
				fmt.Printf("  ✓ %s launched (Window ID: %s)\n", name, instance.Panel.WindowID)
			} else if instance.Panel.Command != nil && instance.Panel.Command.Process != nil {
				fmt.Printf("  ✓ %s launched (PID: %d)\n", name, instance.Panel.Command.Process.Pid)
			} else {
				fmt.Printf("  ✓ %s launched\n", name)
			}

			panelCfg := prismCfg.ToPanelConfig()
			if panelCfg.ListenSocket != "" {
				fmt.Printf("  ✓ Remote control: %s\n", panelCfg.ListenSocket)
			}
		}

		// Wait for each prism to fully start before launching the next
		// This ensures single-instance mode can detect existing instances
		time.Sleep(500 * time.Millisecond)
	}

	// List running prisms
	running := prismMgr.List()
	if len(running) == 0 {
		fmt.Println("\nNo prisms enabled. Edit your config to enable prisms.")
		fmt.Printf("Config location: %s\n", configPath)
		os.Exit(0)
	}

	fmt.Printf("\nRunning %d prism(s): %v\n", len(running), running)
	fmt.Println("Press Ctrl+C to stop all prisms")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal or panels to exit
	go panelMgr.Wait()

	<-sigChan
	fmt.Println("\n\nShutting down...")

	// Stop all prisms
	for _, name := range prismMgr.List() {
		fmt.Printf("Stopping %s...\n", name)
		if err := prismMgr.Stop(name); err != nil {
			log.Printf("Error stopping %s: %v", name, err)
		}
	}

	fmt.Println("Goodbye!")
}
