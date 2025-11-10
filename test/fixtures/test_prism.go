package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func main() {
	// Get prism name from args or use default
	name := "test-prism"
	if len(os.Args) > 1 {
		name = os.Args[1]
	}

	// Setup signal handlers
	sigCh := make(chan os.Signal, 10)
	signal.Notify(sigCh, unix.SIGTERM, unix.SIGINT, unix.SIGWINCH)

	// Print identifying info
	fmt.Printf("=== %s started (PID %d) ===\n", name, os.Getpid())

	// Verify we have a TTY
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Fprintf(os.Stderr, "ERROR: stdin is not a TTY\n")
		os.Exit(1)
	}

	// Print TTY info
	ttyName, err := os.Readlink("/proc/self/fd/0")
	if err != nil {
		ttyName = "unknown"
	}
	fmt.Printf("TTY: %s\n", ttyName)
	fmt.Printf("TERM: %s\n", os.Getenv("TERM"))

	// Print initial terminal size
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err == nil {
		fmt.Printf("SIZE: %dx%d\n", width, height)
	}

	// Ticker for periodic output
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	counter := 0

	// Main loop
	for {
		select {
		case <-ticker.C:
			// Print timestamp every second (shows background processing)
			counter++
			fmt.Printf("[%s] tick %d at %s\n", name, counter, time.Now().Format("15:04:05"))

		case sig := <-sigCh:
			switch sig {
			case unix.SIGWINCH:
				// Print new terminal size on resize
				width, height, err := term.GetSize(int(os.Stdin.Fd()))
				if err == nil {
					fmt.Printf("[%s] SIGWINCH: new size %dx%d\n", name, width, height)
				}

			case unix.SIGTERM, unix.SIGINT:
				// Clean exit
				fmt.Printf("[%s] received %s, exiting cleanly\n", name, sig)
				os.Exit(0)
			}
		}
	}
}
