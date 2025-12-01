package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/unix"
)

type terminalState struct {
	savedTermios *unix.Termios
	fd           int
}

func newTerminalState() (*terminalState, error) {
	fd := int(os.Stdin.Fd())

	termios, err := unix.IoctlGetTermios(fd, unix.TCGETS)
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal attributes: %w", err)
	}

	return &terminalState{
		savedTermios: termios,
		fd:           fd,
	}, nil
}

// resetTerminalState resets the terminal to canonical mode and clears visual state
// This MUST be called after EVERY child exit (clean or crash) to prevent terminal corruption
// - What does canonical mean? https://www.gnu.org/software/libc/manual/html_node/Canonical-or-Not.html
// - TCGETS: termios control get settings
// - TCSETS: termios control set settings (immediate)
//   - See also TCSETSW and TCSETSF, which respectively wait to apply settings until output drained or input flushed 
func (ts *terminalState) resetTerminalState() error {
	// 1. Reset termios to canonical mode
	termios, err := unix.IoctlGetTermios(ts.fd, unix.TCGETS)
	if err != nil {
		return fmt.Errorf("failed to get current terminal attributes: %w", err)
	}

	termios.Lflag |= unix.ICANON | unix.ECHO | unix.ISIG
	termios.Lflag &^= unix.IEXTEN
	termios.Iflag |= unix.ICRNL
	termios.Iflag &^= unix.INLCR

	if err := unix.IoctlSetTermios(ts.fd, unix.TCSETS, termios); err != nil {
		return fmt.Errorf("failed to set terminal attributes: %w", err)
	}

	// 2. Send visual reset sequences to clear terminal state
	resetSeq := []byte{
		0x1b, '[', '0', 'm',                     // SGR reset (colors, bold, etc.)
		0x1b, '[', '?', '1', '0', '4', '9', 'l', // Exit alt screen
		0x1b, '[', '?', '2', '5', 'h',           // Show cursor
		0x1b, '[', '?', '1', '0', '0', '0', 'l', // Disable mouse
		0x1b, '[', '?', '1', '0', '0', '6', 'l', // Disable SGR mouse
	}

	if _, err := unix.Write(ts.fd, resetSeq); err != nil {
		return fmt.Errorf("failed to write reset sequences: %w", err)
	}

	return nil
}

func (ts *terminalState) restoreTerminalState() error {
	if ts.savedTermios == nil {
		return fmt.Errorf("no saved terminal state to restore")
	}

	if err := unix.IoctlSetTermios(ts.fd, unix.TCSETS, ts.savedTermios); err != nil {
		return fmt.Errorf("failed to restore terminal attributes: %w", err)
	}

	return nil
}
