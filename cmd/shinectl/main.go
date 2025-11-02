package main

import (
	"fmt"
	"os"

	"github.com/starbased-co/shine/pkg/panel"
)

func usage() {
	fmt.Println("Usage: shinectl <command> <panel>")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  toggle <panel>  Toggle panel visibility")
	fmt.Println("  show <panel>    Show panel")
	fmt.Println("  hide <panel>    Hide panel")
	fmt.Println()
	fmt.Println("Panels:")
	fmt.Println("  chat           Chat component")
	fmt.Println("  bar            Status bar component")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shinectl toggle chat")
	fmt.Println("  shinectl show chat")
	fmt.Println("  shinectl hide chat")
	fmt.Println("  shinectl toggle bar")
}

func getSocketPath(panelName string) string {
	return fmt.Sprintf("/tmp/shine-%s.sock", panelName)
}

func main() {
	if len(os.Args) < 3 {
		usage()
		os.Exit(1)
	}

	command := os.Args[1]
	panelName := os.Args[2]

	// Get socket path for panel
	socketPath := getSocketPath(panelName)

	// Create remote control client
	rc := panel.NewRemoteControl(socketPath)

	// Execute command
	var err error
	switch command {
	case "toggle":
		fmt.Printf("Toggling %s panel...\n", panelName)
		err = rc.ToggleVisibility()
	case "show":
		fmt.Printf("Showing %s panel...\n", panelName)
		err = rc.Show()
	case "hide":
		fmt.Printf("Hiding %s panel...\n", panelName)
		err = rc.Hide()
	default:
		fmt.Printf("Error: unknown command '%s'\n\n", command)
		usage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		fmt.Printf("\nTroubleshooting:\n")
		fmt.Printf("  - Is the %s panel running? (check with: ps aux | grep shine-%s)\n", panelName, panelName)
		fmt.Printf("  - Is the socket accessible? (check: ls -la %s)\n", socketPath)
		os.Exit(1)
	}

	fmt.Printf("✓ Command sent successfully\n")
}
