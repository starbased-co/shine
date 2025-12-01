package main

import (
	"fmt"
	"os"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

func allocatePTY() (master, slave *os.File, err error) {
	master, slave, err = pty.Open()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to allocate PTY: %w", err)
	}
	return master, slave, nil
}

// syncTerminalSize copies terminal size from source FD to target FD
// NOTE: will be important function for more complex multiplexing i.e. multi-app/split window
func syncTerminalSize(sourceFd, targetFd int) error {
	sourceWinsize, err := unix.IoctlGetWinsize(sourceFd, unix.TIOCGWINSZ)
	if err != nil {
		return fmt.Errorf("failed to get source terminal size: %w", err)
	}

	if err := unix.IoctlSetWinsize(targetFd, unix.TIOCSWINSZ, sourceWinsize); err != nil {
		return fmt.Errorf("failed to set target terminal size: %w", err)
	}

	return nil
}

func closePTY(master *os.File) error {
	if master == nil {
		return nil
	}

	if err := master.Close(); err != nil {
		return fmt.Errorf("failed to close PTY master: %w", err)
	}

	return nil
}
