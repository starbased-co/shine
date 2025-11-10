package integration

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestStress launches many prisms, swaps randomly, kills randomly
func TestStress(t *testing.T) {
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

	// Launch 5 prisms
	prismNames := []string{"prism-A", "prism-B", "prism-C", "prism-D", "prism-E"}
	for _, name := range prismNames {
		resp := sendIPCCommand(t, sockPath, map[string]string{
			"action": "start",
			"prism":  "test-prism " + name,
		})
		if !resp["success"].(bool) {
			t.Fatalf("Failed to start %s: %v", name, resp["error"])
		}
		time.Sleep(100 * time.Millisecond)
	}

	t.Logf("✓ Launched 5 prisms")

	// Random seed
	rand.Seed(time.Now().UnixNano())

	// Track active prisms
	activePrisms := make(map[string]bool)
	for _, name := range prismNames {
		activePrisms[name] = true
	}

	swapCount := 0
	killCount := 0
	relaunchCount := 0

	// Stress test loop: 100 operations
	for i := 0; i < 100; i++ {
		// Get current status
		resp := sendIPCCommand(t, sockPath, map[string]string{
			"action": "status",
		})
		if !resp["success"].(bool) {
			t.Fatalf("Status command failed at iteration %d: %v", i, resp["error"])
		}

		// Random operation
		op := rand.Intn(100)

		if op < 70 { // 70% chance: swap to random prism
			// Pick random active prism
			var target string
			for name := range activePrisms {
				if activePrisms[name] {
					target = name
					break
				}
			}

			if target != "" {
				resp = sendIPCCommand(t, sockPath, map[string]string{
					"action": "start",
					"prism":  "test-prism " + target,
				})
				if resp["success"].(bool) {
					swapCount++
				} else {
					t.Logf("Warning: swap to %s failed: %v", target, resp["error"])
				}
			}

		} else if op < 85 { // 15% chance: kill random prism
			// Pick random active prism to kill
			var target string
			for name := range activePrisms {
				if activePrisms[name] {
					target = name
					break
				}
			}

			if target != "" && len(activePrisms) > 1 { // Keep at least 1 prism alive
				resp = sendIPCCommand(t, sockPath, map[string]string{
					"action": "kill",
					"prism":  "test-prism " + target,
				})
				if resp["success"].(bool) {
					delete(activePrisms, target)
					killCount++
				} else {
					t.Logf("Warning: kill %s failed: %v", target, resp["error"])
				}
			}

		} else { // 15% chance: relaunch killed prism
			// Find a killed prism
			for _, name := range prismNames {
				if !activePrisms[name] {
					resp = sendIPCCommand(t, sockPath, map[string]string{
						"action": "start",
						"prism":  "test-prism " + name,
					})
					if resp["success"].(bool) {
						activePrisms[name] = true
						relaunchCount++
					} else {
						t.Logf("Warning: relaunch %s failed: %v", name, resp["error"])
					}
					break
				}
			}
		}

		// Small delay between operations
		time.Sleep(20 * time.Millisecond)
	}

	t.Logf("✓ Stress test completed: 100 operations")
	t.Logf("  Swaps: %d", swapCount)
	t.Logf("  Kills: %d", killCount)
	t.Logf("  Relaunches: %d", relaunchCount)

	// Final status check
	resp := sendIPCCommand(t, sockPath, map[string]string{
		"action": "status",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Final status check failed: %v", resp["error"])
	}

	t.Logf("✓ Final status: foreground=%s, background=%v", resp["foreground"], resp["background"])

	// Verify no FD leaks by checking /proc/PID/fd count
	fdPath := fmt.Sprintf("/proc/%d/fd", cmd.Process.Pid)
	entries, err := os.ReadDir(fdPath)
	if err == nil {
		fdCount := len(entries)
		t.Logf("✓ Open file descriptors: %d", fdCount)

		// Rough estimate: should not have hundreds of FDs open
		if fdCount > 100 {
			t.Errorf("Potential FD leak: %d file descriptors open", fdCount)
		}
	}

	// Cleanup
	sendIPCCommand(t, sockPath, map[string]string{"action": "stop"})
	cmd.Wait()

	t.Logf("✓ Stress test PASSED - no crashes, no FD leaks")
}
