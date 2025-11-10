package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// TestSIGWINCH verifies terminal resize propagation
// Note: This test is limited because we can't programmatically resize the terminal
// It mainly verifies that SIGWINCH doesn't crash prismctl
func TestSIGWINCH(t *testing.T) {
	// Build paths
	prismctlBin := filepath.Join("..", "..", "bin", "prismctl")
	testPrismBin := filepath.Join("..", "fixtures", "test-prism")

	// Verify binaries exist
	if _, err := os.Stat(prismctlBin); err != nil {
		t.Fatalf("prismctl binary not found: %v", err)
	}
	if _, err := os.Stat(testPrismBin); err != nil {
		t.Fatalf("test-prism binary not found: %v", err)
	}

	// Add test-prism to PATH
	testPath := filepath.Dir(testPrismBin)
	os.Setenv("PATH", testPath+":"+os.Getenv("PATH"))

	// Start prismctl
	cmd := exec.Command(prismctlBin, "panel-test", testPrismBin)
	cmd.Env = append(os.Environ(), "TERM=xterm-kitty")

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start prismctl: %v", err)
	}
	defer cmd.Process.Kill()

	time.Sleep(500 * time.Millisecond)

	// Find IPC socket
	sockPath, err := findIPCSocket(cmd.Process.Pid)
	if err != nil {
		t.Fatalf("Failed to find IPC socket: %v", err)
	}

	// Launch multiple prisms
	for _, name := range []string{"prism-A", "prism-B", "prism-C"} {
		resp := sendIPCCommand(t, sockPath, map[string]string{
			"action": "start",
			"prism":  "test-prism " + name,
		})
		if !resp["success"].(bool) {
			t.Fatalf("Failed to start %s: %v", name, resp["error"])
		}
		time.Sleep(200 * time.Millisecond)
	}

	t.Logf("✓ Launched 3 prisms")

	// Send SIGWINCH to prismctl process
	if err := cmd.Process.Signal(syscall.SIGWINCH); err != nil {
		t.Fatalf("Failed to send SIGWINCH: %v", err)
	}

	t.Logf("✓ Sent SIGWINCH to prismctl")

	// Wait and verify prismctl is still running
	time.Sleep(500 * time.Millisecond)

	// Try to get status (verifies prismctl didn't crash)
	resp := sendIPCCommand(t, sockPath, map[string]string{
		"action": "status",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Status command failed after SIGWINCH: %v", resp["error"])
	}

	t.Logf("✓ prismctl handled SIGWINCH without crashing")
	t.Logf("  Foreground: %s", resp["foreground"])
	t.Logf("  Background: %v", resp["background"])

	// Send multiple SIGWINCH signals
	for i := 0; i < 5; i++ {
		if err := cmd.Process.Signal(syscall.SIGWINCH); err != nil {
			t.Fatalf("Failed to send SIGWINCH %d: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("✓ Sent 5 SIGWINCH signals")

	// Verify still running
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "status",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Status command failed after multiple SIGWINCH: %v", resp["error"])
	}

	t.Logf("✓ prismctl handled multiple SIGWINCH signals successfully")

	// Cleanup
	sendIPCCommand(t, sockPath, map[string]string{"action": "stop"})
	cmd.Wait()
}
