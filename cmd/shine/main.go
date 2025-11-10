package main

import (
	"fmt"
	"os"
)

const version = "0.2.0"

func usage() {
	fmt.Printf("✨ shine v%s - Prism TUI Manager\n\n", version)
	fmt.Println("Usage: shine <command> [arguments]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  start              Start/resume the shine service")
	fmt.Println("  stop               Gracefully stop all panels")
	fmt.Println("  reload             Reload configuration and update panels")
	fmt.Println("  status             Show status of all panels")
	fmt.Println("  logs [panel-id]    View logs (all or specific panel)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, --help         Show this help message")
	fmt.Println("  -v, --version      Show version information")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shine start        # Start shinectl service")
	fmt.Println("  shine status       # Show panel status")
	fmt.Println("  shine logs         # List all log files")
	fmt.Println("  shine logs shinectl # View shinectl.log")
	fmt.Println("  shine stop         # Stop all panels")
	fmt.Println()
	fmt.Println("Configuration:")
	fmt.Println("  Config file: ~/.config/shine/prism.toml")
	fmt.Println("  Log files:   ~/.local/share/shine/logs/")
	fmt.Println()
	fmt.Println("For more information, see the documentation.")
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Handle flags
	switch command {
	case "-h", "--help":
		usage()
		return
	case "-v", "--version":
		fmt.Printf("shine v%s\n", version)
		return
	}

	// Execute command
	var err error
	switch command {
	case "start":
		err = cmdStart()

	case "stop":
		err = cmdStop()

	case "reload":
		err = cmdReload()

	case "status":
		err = cmdStatus()

	case "logs":
		panelID := ""
		if len(os.Args) > 2 {
			panelID = os.Args[2]
		}
		err = cmdLogs(panelID)

	default:
		Error(fmt.Sprintf("Unknown command: %s", command))
		fmt.Println()
		usage()
		os.Exit(1)
	}

	if err != nil {
		Error(err.Error())
		os.Exit(1)
	}
}
