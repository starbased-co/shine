package integration

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBackgroundProcessing ensures background prisms keep running
func TestBackgroundProcessing(t *testing.T) {
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

	// Start prismctl and capture output
	cmd := exec.Command(prismctlBin, "panel-test", testPrismBin)
	cmd.Env = append(os.Environ(), "TERM=xterm-kitty")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}

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

	// Launch prism A (prints timestamp every 1s)
	resp := sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-A",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to start prism-A: %v", resp["error"])
	}
	t.Logf("✓ Launched prism-A (foreground)")

	time.Sleep(2 * time.Second)

	// Record tick count from prism-A
	initialTicks := captureTickCount(t, stdout, "prism-A")
	t.Logf("✓ prism-A initial ticks: %d", initialTicks)

	// Launch prism B (moves A to background)
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-B",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to start prism-B: %v", resp["error"])
	}
	t.Logf("✓ Launched prism-B (foreground, prism-A now background)")

	// Wait 5 seconds with prism-A in background
	t.Logf("Waiting 5s with prism-A in background...")
	time.Sleep(5 * time.Second)

	// Swap back to prism-A
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-A",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to swap back to prism-A: %v", resp["error"])
	}
	t.Logf("✓ Swapped back to prism-A (foreground)")

	time.Sleep(1 * time.Second)

	// Record new tick count
	finalTicks := captureTickCount(t, stdout, "prism-A")
	t.Logf("✓ prism-A final ticks: %d", finalTicks)

	// Verify ticks continued (should have ~5-7 more ticks)
	tickDelta := finalTicks - initialTicks
	t.Logf("Tick delta: %d (expected ~5-7)", tickDelta)

	if tickDelta < 4 {
		t.Fatalf("Background processing failed: prism-A only gained %d ticks (expected ~5-7)", tickDelta)
	}

	t.Logf("✓ Background processing verified: prism-A continued running while in background")

	// Cleanup
	sendIPCCommand(t, sockPath, map[string]string{"action": "stop"})
	cmd.Wait()
}

// Helper: Capture tick count from prism output
func captureTickCount(t *testing.T, stdout interface{}, prismName string) int {
	scanner := bufio.NewScanner(stdout.(interface{ Read([]byte) (int, error) }))
	maxTick := 0

	// Scan recent output for tick messages
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if scanner.Scan() {
			line := scanner.Text()
			// Look for: "[prism-A] tick N at HH:MM:SS"
			if strings.Contains(line, fmt.Sprintf("[%s] tick", prismName)) {
				var tick int
				if _, err := fmt.Sscanf(line, fmt.Sprintf("[%s] tick %%d", prismName), &tick); err == nil {
					if tick > maxTick {
						maxTick = tick
					}
				}
			}
		}
	}

	return maxTick
}
