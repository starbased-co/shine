package main

import (
	"os"
	"testing"

	"golang.org/x/sys/unix"
)

func TestAllocatePTY_Success(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}

	if master == nil {
		t.Fatal("allocatePTY() returned nil master")
	}

	if slave == nil {
		t.Fatal("allocatePTY() returned nil slave")
	}

	// Verify master and slave have valid file descriptors
	if master.Fd() == 0 {
		t.Error("allocatePTY() master has invalid FD")
	}

	if slave.Fd() == 0 {
		t.Error("allocatePTY() slave has invalid FD")
	}

	// Verify FDs are different
	if master.Fd() == slave.Fd() {
		t.Error("allocatePTY() master and slave have same FD")
	}

	// Clean up
	master.Close()
	slave.Close()
}

func TestAllocatePTY_MasterWriteSlaveRead(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	// Write to master, read from slave
	testData := []byte("test data\n")
	n, err := master.Write(testData)
	if err != nil {
		t.Fatalf("failed to write to master: %v", err)
	}

	if n != len(testData) {
		t.Errorf("wrote %d bytes to master, want %d", n, len(testData))
	}

	buf := make([]byte, len(testData))
	n, err = slave.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from slave: %v", err)
	}

	if n != len(testData) {
		t.Errorf("read %d bytes from slave, want %d", n, len(testData))
	}

	if string(buf) != string(testData) {
		t.Errorf("read %q from slave, want %q", string(buf), string(testData))
	}
}

func TestAllocatePTY_SlaveWriteMasterRead(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}
	defer master.Close()
	defer slave.Close()

	// Write to slave, read from master
	// Note: PTY may perform line ending conversion (\n -> \r)
	testData := []byte("reverse test")
	n, err := slave.Write(testData)
	if err != nil {
		t.Fatalf("failed to write to slave: %v", err)
	}

	if n != len(testData) {
		t.Errorf("wrote %d bytes to slave, want %d", n, len(testData))
	}

	buf := make([]byte, len(testData)+4) // Extra space for line ending conversion
	n, err = master.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from master: %v", err)
	}

	// Verify we read the expected data (allow for line ending conversion)
	if n < len(testData) {
		t.Errorf("read %d bytes from master, want at least %d", n, len(testData))
	}
}

func TestSyncTerminalSize_ValidFDs(t *testing.T) {
	// Create two PTY pairs
	source, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate source PTY: %v", err)
	}
	defer source.Close()

	target, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate target PTY: %v", err)
	}
	defer target.Close()

	// Set a specific size on source
	sourceWinsize := &unix.Winsize{
		Row: 24,
		Col: 80,
	}
	if err := unix.IoctlSetWinsize(int(source.Fd()), unix.TIOCSWINSZ, sourceWinsize); err != nil {
		t.Fatalf("failed to set source winsize: %v", err)
	}

	// Sync from source to target
	if err := syncTerminalSize(int(source.Fd()), int(target.Fd())); err != nil {
		t.Fatalf("syncTerminalSize() unexpected error: %v", err)
	}

	// Verify target has same size as source
	targetWinsize, err := unix.IoctlGetWinsize(int(target.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		t.Fatalf("failed to get target winsize: %v", err)
	}

	if targetWinsize.Row != sourceWinsize.Row {
		t.Errorf("target rows = %d, want %d", targetWinsize.Row, sourceWinsize.Row)
	}

	if targetWinsize.Col != sourceWinsize.Col {
		t.Errorf("target cols = %d, want %d", targetWinsize.Col, sourceWinsize.Col)
	}
}

func TestSyncTerminalSize_DifferentSizes(t *testing.T) {
	// Create two PTY pairs
	source, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate source PTY: %v", err)
	}
	defer source.Close()

	target, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate target PTY: %v", err)
	}
	defer target.Close()

	// Set different sizes
	sourceWinsize := &unix.Winsize{
		Row: 50,
		Col: 120,
	}
	if err := unix.IoctlSetWinsize(int(source.Fd()), unix.TIOCSWINSZ, sourceWinsize); err != nil {
		t.Fatalf("failed to set source winsize: %v", err)
	}

	targetWinsize := &unix.Winsize{
		Row: 24,
		Col: 80,
	}
	if err := unix.IoctlSetWinsize(int(target.Fd()), unix.TIOCSWINSZ, targetWinsize); err != nil {
		t.Fatalf("failed to set target winsize: %v", err)
	}

	// Sync should overwrite target with source size
	if err := syncTerminalSize(int(source.Fd()), int(target.Fd())); err != nil {
		t.Fatalf("syncTerminalSize() unexpected error: %v", err)
	}

	// Verify target now matches source
	newTargetWinsize, err := unix.IoctlGetWinsize(int(target.Fd()), unix.TIOCGWINSZ)
	if err != nil {
		t.Fatalf("failed to get target winsize: %v", err)
	}

	if newTargetWinsize.Row != sourceWinsize.Row {
		t.Errorf("target rows = %d, want %d", newTargetWinsize.Row, sourceWinsize.Row)
	}

	if newTargetWinsize.Col != sourceWinsize.Col {
		t.Errorf("target cols = %d, want %d", newTargetWinsize.Col, sourceWinsize.Col)
	}
}

func TestSyncTerminalSize_InvalidSourceFD(t *testing.T) {
	target, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate target PTY: %v", err)
	}
	defer target.Close()

	// Use invalid source FD
	err = syncTerminalSize(-1, int(target.Fd()))
	if err == nil {
		t.Error("syncTerminalSize() with invalid source FD should return error")
	}

	if err != nil && err.Error() == "" {
		t.Error("syncTerminalSize() should return non-empty error message")
	}
}

func TestSyncTerminalSize_InvalidTargetFD(t *testing.T) {
	source, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate source PTY: %v", err)
	}
	defer source.Close()

	// Use invalid target FD
	err = syncTerminalSize(int(source.Fd()), -1)
	if err == nil {
		t.Error("syncTerminalSize() with invalid target FD should return error")
	}

	if err != nil && err.Error() == "" {
		t.Error("syncTerminalSize() should return non-empty error message")
	}
}

func TestSyncTerminalSize_BothInvalidFDs(t *testing.T) {
	err := syncTerminalSize(-1, -1)
	if err == nil {
		t.Error("syncTerminalSize() with both invalid FDs should return error")
	}
}

func TestSyncTerminalSize_RegularFileFD(t *testing.T) {
	// Create a regular file (not a TTY)
	tmpFile, err := os.CreateTemp("", "test-regular-file")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	target, _, err := allocatePTY()
	if err != nil {
		t.Fatalf("failed to allocate target PTY: %v", err)
	}
	defer target.Close()

	// Regular file FD should fail ioctl operations
	err = syncTerminalSize(int(tmpFile.Fd()), int(target.Fd()))
	if err == nil {
		t.Error("syncTerminalSize() with regular file FD should return error")
	}
}

func TestClosePTY_NilMaster(t *testing.T) {
	// Should not panic or error
	err := closePTY(nil)
	if err != nil {
		t.Errorf("closePTY(nil) returned error: %v, want nil", err)
	}
}

func TestClosePTY_ValidMaster(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}
	defer slave.Close()

	err = closePTY(master)
	if err != nil {
		t.Errorf("closePTY() unexpected error: %v", err)
	}

	// Verify master is closed (operations should fail)
	buf := make([]byte, 1)
	_, err = master.Read(buf)
	if err == nil {
		t.Error("reading from closed PTY should fail")
	}
}

func TestClosePTY_DoubleClose(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}
	defer slave.Close()

	// First close
	err = closePTY(master)
	if err != nil {
		t.Errorf("first closePTY() unexpected error: %v", err)
	}

	// Second close should return error (file already closed)
	err = closePTY(master)
	if err == nil {
		t.Error("second closePTY() should return error for already-closed file")
	}
}

func TestClosePTY_SlaveStillOpen(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}
	defer slave.Close()

	// Close master while slave is still open
	err = closePTY(master)
	if err != nil {
		t.Errorf("closePTY() with open slave unexpected error: %v", err)
	}

	// When master closes, slave can still write but will get EOF on read
	// This test just verifies master close doesn't crash slave operations
	_, writeErr := slave.Write([]byte("x"))
	if writeErr != nil {
		t.Logf("slave write after master close: %v (expected)", writeErr)
	}
}

func TestPTYPair_IndependentLifecycle(t *testing.T) {
	master, slave, err := allocatePTY()
	if err != nil {
		t.Fatalf("allocatePTY() unexpected error: %v", err)
	}

	// Close slave first
	slave.Close()

	// Master should still be valid
	buf := make([]byte, 1)
	_, err = master.Read(buf)
	// Read might fail with EOF or other error, but FD should be valid
	if master.Fd() == 0 {
		t.Error("master FD became invalid after slave close")
	}

	// Clean up
	master.Close()
}

func TestAllocatePTY_MultiplePairs(t *testing.T) {
	// Allocate multiple PTY pairs to verify resource cleanup
	pairs := make([]*os.File, 0)

	for i := 0; i < 5; i++ {
		master, slave, err := allocatePTY()
		if err != nil {
			t.Fatalf("allocatePTY() iteration %d failed: %v", i, err)
		}
		pairs = append(pairs, master, slave)
	}

	// Clean up all PTYs
	for _, pty := range pairs {
		pty.Close()
	}
}
