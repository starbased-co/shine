package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/starbased-co/shine/pkg/config"
	"github.com/starbased-co/shine/pkg/panel"
)

// findComponentBinary finds the full path to a component binary
func findComponentBinary(name string) (string, error) {
	// First, check if it's in PATH
	path, err := exec.LookPath(name)
	if err == nil {
		return path, nil
	}

	// If not in PATH, try relative to the shine binary
	exePath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	// Try in the same directory as shine
	binDir := filepath.Dir(exePath)
	componentPath := filepath.Join(binDir, name)
	if _, err := os.Stat(componentPath); err == nil {
		return componentPath, nil
	}

	return "", fmt.Errorf("component binary %s not found in PATH or %s", name, binDir)
}

func main() {
	// Load configuration
	configPath := config.DefaultConfigPath()
	cfg := config.LoadOrDefault(configPath)

	fmt.Printf("✨ Shine - Hyprland Layer Shell TUI Toolkit\n")
	fmt.Printf("Configuration: %s\n\n", configPath)

	// Create panel manager
	mgr := panel.NewManager()

	// Launch enabled components
	if cfg.Chat != nil && cfg.Chat.Enabled {
		fmt.Println("Launching chat panel...")
		panelCfg := cfg.Chat.ToPanelConfig()

		// Find the shine-chat binary
		chatBinary, err := findComponentBinary("shine-chat")
		if err != nil {
			log.Fatalf("Failed to find shine-chat binary: %v", err)
		}

		instance, err := mgr.Launch("chat", panelCfg, chatBinary)
		if err != nil {
			log.Fatalf("Failed to launch chat panel: %v", err)
		}

		fmt.Printf("  ✓ Chat panel launched (PID: %d)\n", instance.Command.Process.Pid)
		if panelCfg.ListenSocket != "" {
			fmt.Printf("  ✓ Remote control: %s\n", panelCfg.ListenSocket)
		}
	}

	if cfg.Bar != nil && cfg.Bar.Enabled {
		fmt.Println("Launching status bar...")
		panelCfg := cfg.Bar.ToPanelConfig()

		// Find the shine-bar binary
		barBinary, err := findComponentBinary("shine-bar")
		if err != nil {
			log.Fatalf("Failed to find shine-bar binary: %v", err)
		}

		instance, err := mgr.Launch("bar", panelCfg, barBinary)
		if err != nil {
			log.Fatalf("Failed to launch status bar: %v", err)
		}

		fmt.Printf("  ✓ Status bar launched (PID: %d)\n", instance.Command.Process.Pid)
		if panelCfg.ListenSocket != "" {
			fmt.Printf("  ✓ Remote control: %s\n", panelCfg.ListenSocket)
		}
	}

	// List running panels
	panels := mgr.List()
	if len(panels) == 0 {
		fmt.Println("\nNo panels enabled. Edit your config to enable components.")
		fmt.Printf("Config location: %s\n", configPath)
		os.Exit(0)
	}

	fmt.Printf("\nRunning %d panel(s): %v\n", len(panels), panels)
	fmt.Println("Press Ctrl+C to stop all panels")

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for signal or panels to exit
	go mgr.Wait()

	<-sigChan
	fmt.Println("\n\nShutting down...")

	// Stop all panels
	for _, name := range mgr.List() {
		fmt.Printf("Stopping %s...\n", name)
		if err := mgr.Stop(name); err != nil {
			log.Printf("Error stopping %s: %v", name, err)
		}
	}

	fmt.Println("Goodbye!")
}
