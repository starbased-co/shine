package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestHotSwap verifies instant switching between prisms
func TestHotSwap(t *testing.T) {
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

	// Launch prism A (foreground)
	resp := sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-A",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to start prism-A: %v", resp["error"])
	}
	t.Logf("✓ Launched prism-A")

	time.Sleep(500 * time.Millisecond)

	// Launch prism B (goes to background)
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-B",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to start prism-B: %v", resp["error"])
	}
	t.Logf("✓ Launched prism-B (should be foreground now)")

	time.Sleep(500 * time.Millisecond)

	// Perform hot-swap: B → A
	t.Logf("Performing hot-swap: prism-B → prism-A")
	startTime := time.Now()

	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-A",
	})
	swapLatency := time.Since(startTime)

	if !resp["success"].(bool) {
		t.Fatalf("Failed to swap to prism-A: %v", resp["error"])
	}

	t.Logf("✓ Hot-swap completed in %v", swapLatency)

	// Verify swap latency is under 50ms target
	if swapLatency > 50*time.Millisecond {
		t.Errorf("Swap latency %v exceeds 50ms target", swapLatency)
	} else {
		t.Logf("✓ Swap latency %v is within 50ms target", swapLatency)
	}

	// Verify foreground is now prism-A
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "status",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Status command failed: %v", resp["error"])
	}

	foreground := resp["foreground"].(string)
	if !contains(foreground, "prism-A") {
		t.Fatalf("Expected foreground to be prism-A, got: %s", foreground)
	}

	t.Logf("✓ Verified foreground is prism-A")

	// Perform multiple rapid swaps
	t.Logf("Performing 10 rapid swaps...")
	totalLatency := time.Duration(0)
	maxLatency := time.Duration(0)

	for i := 0; i < 10; i++ {
		target := "prism-A"
		if i%2 == 0 {
			target = "prism-B"
		}

		startTime := time.Now()
		resp = sendIPCCommand(t, sockPath, map[string]string{
			"action": "start",
			"prism":  fmt.Sprintf("test-prism %s", target),
		})
		latency := time.Since(startTime)

		if !resp["success"].(bool) {
			t.Fatalf("Swap %d failed: %v", i, resp["error"])
		}

		totalLatency += latency
		if latency > maxLatency {
			maxLatency = latency
		}

		// Small delay between swaps
		time.Sleep(50 * time.Millisecond)
	}

	avgLatency := totalLatency / 10
	t.Logf("✓ 10 rapid swaps completed")
	t.Logf("  Average latency: %v", avgLatency)
	t.Logf("  Max latency: %v", maxLatency)

	if avgLatency > 50*time.Millisecond {
		t.Errorf("Average swap latency %v exceeds 50ms target", avgLatency)
	}

	// Cleanup
	sendIPCCommand(t, sockPath, map[string]string{"action": "stop"})
	cmd.Wait()
}

// Helper: Check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		 indexOf(s, substr) >= 0))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
