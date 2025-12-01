package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/starbased-co/shine/pkg/paths"
	"github.com/starbased-co/shine/pkg/rpc"
	"github.com/starbased-co/shine/pkg/state"
)

func connectShined() (*rpc.ShinedClient, error) {
	sockPath := paths.ShinedSocket()
	return rpc.NewShinedClient(sockPath, rpc.WithTimeout(3*time.Second))
}

func isShinedRunning() bool {
	_, err := os.Stat(paths.ShinedSocket())
	return err == nil
}

// discovers running prismctl instances by scanning for runtime files (sockets/mmap)
func discoverPrismInstances() ([]string, error) {
	socketsDir := paths.RuntimeDir()

	// Try sockets first (they're authoritative for running instances)
	pattern := filepath.Join(socketsDir, "prism-*.sock")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to search for sockets: %w", err)
	}

	instances := make([]string, 0, len(matches))
	for _, socket := range matches {
		instances = append(instances, extractInstanceName(socket))
	}

	return instances, nil
}

func cmdStart() error {
	if isShinedRunning() {
		Success("shined is already running")
		return nil
	}

	Info("Starting shined service...")

	shinedBin, err := exec.LookPath("shined")
	if err != nil {
		return fmt.Errorf("shined not found in PATH: %w", err)
	}

	// Start shined in background
	cmd := exec.Command(shinedBin)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start shined: %w", err)
	}

	// Wait for socket to appear
	for i := 0; i < 50; i++ {
		if isShinedRunning() {
			Success(fmt.Sprintf("shined started (PID: %d)", cmd.Process.Pid))
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	return fmt.Errorf("shined started but socket not created within timeout")
}

func cmdStop() error {
	Info("Stopping shine service...")

	ctx := context.Background()

	if isShinedRunning() {
		client, err := connectShined()
		if err != nil {
			Warning(fmt.Sprintf("Failed to connect to shined: %v, falling back to direct shutdown", err))
		} else {
			defer client.Close()

			result, err := client.Status(ctx)
			if err != nil {
				Warning(fmt.Sprintf("Failed to query shined status: %v", err))
			} else {
				for _, panel := range result.Panels {
					Muted(fmt.Sprintf("Stopping %s...", panel.Instance))
					_, err := client.KillPanel(ctx, panel.Instance)
					if err != nil {
						Warning(fmt.Sprintf("Failed to stop %s: %v", panel.Instance, err))
					}
				}
				Success(fmt.Sprintf("Stopped %d panel(s)", len(result.Panels)))
				return nil
			}
		}
	}

	// Fallback: discover and stop prismctl instances directly
	instances, err := discoverPrismInstances()
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		Warning("No panels running")
		return nil
	}

	var stopped int
	for _, instance := range instances {
		Muted(fmt.Sprintf("Stopping %s...", instance))

		client, err := rpc.NewPrismClient(paths.PrismSocket(instance))
		if err != nil {
			Warning(fmt.Sprintf("Failed to connect to %s: %v", instance, err))
			continue
		}

		_, err = client.Shutdown(ctx, true)
		client.Close()

		if err != nil {
			Warning(fmt.Sprintf("Failed to stop %s: %v", instance, err))
		} else {
			stopped++
		}
	}

	Success(fmt.Sprintf("Stopped %d panel(s)", stopped))
	return nil
}

func cmdReload() error {
	Info("Reloading configuration...")

	if !isShinedRunning() {
		return fmt.Errorf("shined is not running")
	}

	ctx := context.Background()
	client, err := connectShined()
	if err != nil {
		return fmt.Errorf("failed to connect to shined: %w", err)
	}
	defer client.Close()

	result, err := client.Reload(ctx)
	if err != nil {
		return fmt.Errorf("reload request failed: %w", err)
	}

	if !result.Reloaded {
		if len(result.Errors) > 0 {
			Error("Configuration reload failed:")
			for _, errMsg := range result.Errors {
				Muted(fmt.Sprintf("  - %s", errMsg))
			}
			return fmt.Errorf("reload completed with errors")
		}
		return fmt.Errorf("reload failed with no error details")
	}

	Success("Configuration reloaded successfully")
	return nil
}

func displayStateFromMmap(instance string, s *state.PrismRuntimeState) {
	fmt.Println()
	fmt.Printf("%s %s\n", styleBold.Render("Panel:"), instance)
	fmt.Printf("%s %s\n", styleMuted.Render("Source:"), "mmap")

	fgName := s.GetFgPrism()
	activePrisms := s.ActivePrisms()
	bgCount := len(activePrisms) - 1
	if fgName == "" {
		bgCount = len(activePrisms)
	}

	fmt.Println(StatusBox(fgName, bgCount, len(activePrisms)))

	if len(activePrisms) > 0 {
		table := NewTable("Prism", "PID", "State", "Uptime")
		for _, prism := range activePrisms {
			name := prism.GetName()
			stateStr := "background"
			if name == fgName {
				stateStr = styleSuccess.Render("foreground")
			} else {
				stateStr = styleMuted.Render("background")
			}
			uptime := prism.Uptime()
			uptimeStr := fmt.Sprintf("%v", uptime.Truncate(time.Second))
			table.AddRow(name, fmt.Sprintf("%d", prism.PID), stateStr, uptimeStr)
		}
		fmt.Println()
		table.Print()
	}
}

func displayStateFromRPC(instance string, prisms []rpc.PrismInfo) {
	fmt.Println()
	fmt.Printf("%s %s\n", styleBold.Render("Panel:"), instance)
	fmt.Printf("%s %s\n", styleMuted.Render("Source:"), "rpc")

	fgName := ""
	bgCount := 0
	for _, p := range prisms {
		if p.State == "fg" {
			fgName = p.Name
		} else {
			bgCount++
		}
	}

	fmt.Println(StatusBox(fgName, bgCount, len(prisms)))

	if len(prisms) > 0 {
		table := NewTable("Prism", "PID", "State", "Uptime")
		for _, prism := range prisms {
			stateStr := prism.State
			if prism.State == "fg" {
				stateStr = styleSuccess.Render("foreground")
			} else {
				stateStr = styleMuted.Render("background")
			}
			uptime := time.Duration(prism.UptimeMs) * time.Millisecond
			uptimeStr := fmt.Sprintf("%v", uptime.Truncate(time.Second))
			table.AddRow(prism.Name, fmt.Sprintf("%d", prism.PID), stateStr, uptimeStr)
		}
		fmt.Println()
		table.Print()
	}
}

func cmdStatus() error {
	ctx := context.Background()

	// Try shined first for aggregated status
	if isShinedRunning() {
		client, err := connectShined()
		if err == nil {
			defer client.Close()

			result, err := client.Status(ctx)
			if err == nil {
				// Display shined-level status
				uptime := time.Duration(result.Uptime) * time.Millisecond
				uptimeStr := uptime.Truncate(time.Second).String()

				Header(fmt.Sprintf("Shine Status (v%s, uptime: %s)", result.Version, uptimeStr))

				if len(result.Panels) == 0 {
					Warning("No panels running")
					Info("Start panels with: shine start")
					return nil
				}

				// Query each panel for detailed status
				for _, panel := range result.Panels {
					displayPanelStatus(ctx, panel.Instance)
				}
				return nil
			}
			// If shined query fails, fall through to discovery
			Warning(fmt.Sprintf("Failed to query shined: %v, falling back to discovery", err))
		}
	}

	// Fallback: discover all running prism instances directly
	instances, err := discoverPrismInstances()
	if err != nil {
		return err
	}

	if len(instances) == 0 {
		Warning("No panels running")
		Info("Start panels with: shine start")
		return nil
	}

	Header(fmt.Sprintf("Shine Status (%d panel(s))", len(instances)))

	for _, instance := range instances {
		displayPanelStatus(ctx, instance)
	}

	return nil
}

func displayPanelStatus(ctx context.Context, instance string) {
	// Try mmap first (instant, no connection needed)
	reader, err := state.OpenPrismStateReader(paths.PrismState(instance))
	if err == nil {
		s, readErr := reader.Read()
		reader.Close()
		if readErr == nil {
			displayStateFromMmap(instance, s)
			return
		}
	}

	// Fallback to RPC
	client, err := rpc.NewPrismClient(paths.PrismSocket(instance))
	if err != nil {
		fmt.Println()
		fmt.Printf("%s %s\n", styleBold.Render("Panel:"), instance)
		Error(fmt.Sprintf("Failed to connect: %v", err))
		return
	}

	result, err := client.List(ctx)
	client.Close()

	if err != nil {
		fmt.Println()
		fmt.Printf("%s %s\n", styleBold.Render("Panel:"), instance)
		Error(fmt.Sprintf("Failed to query: %v", err))
		return
	}

	displayStateFromRPC(instance, result.Prisms)
}

func cmdLogs(panelID string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	logDir := filepath.Join(home, ".local", "share", "shine", "logs")

	if panelID == "" {
		Info(fmt.Sprintf("Log directory: %s", logDir))

		files, err := os.ReadDir(logDir)
		if err != nil {
			return fmt.Errorf("failed to read log directory: %w", err)
		}

		if len(files) == 0 {
			Warning("No log files found")
			return nil
		}

		table := NewTable("Log File", "Size")
		for _, file := range files {
			if file.IsDir() {
				continue
			}
			info, _ := file.Info()
			size := "?"
			if info != nil {
				size = fmt.Sprintf("%d bytes", info.Size())
			}
			table.AddRow(file.Name(), size)
		}

		table.Print()
		fmt.Println()
		Info("View a log with: shine logs <filename>")
		return nil
	}

	logPath := filepath.Join(logDir, panelID)
	if !strings.HasSuffix(logPath, ".log") {
		logPath += ".log"
	}

	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		return fmt.Errorf("log file not found: %s", logPath)
	}

	// last 50 lines for now. TODO: enrich log output later
	cmd := exec.Command("tail", "-n", "50", logPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to read log: %w", err)
	}

	return nil
}

// TODO: remove/redo this way of getting instance name.
func extractInstanceName(socketPath string) string {
	base := filepath.Base(socketPath)
	// Remove "prism-" prefix and ".sock" suffix
	name := strings.TrimPrefix(base, "prism-")
	name = strings.TrimSuffix(name, ".sock")
	return name
}
