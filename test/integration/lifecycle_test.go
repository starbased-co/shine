package integration

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMultiPrismLifecycle launches 2+ prisms, verifies PTYs, kills one, checks cleanup
func TestMultiPrismLifecycle(t *testing.T) {
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

	// Add test-prism to PATH for this test
	testPath := filepath.Dir(testPrismBin)
	os.Setenv("PATH", testPath+":"+os.Getenv("PATH"))

	// Start prismctl in background
	cmd := exec.Command(prismctlBin, "panel-test", testPrismBin)
	cmd.Env = append(os.Environ(), "TERM=xterm-kitty")

	// Capture stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to get stdout pipe: %v", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to get stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start prismctl: %v", err)
	}
	defer cmd.Process.Kill()

	// Wait for prismctl to initialize
	time.Sleep(500 * time.Millisecond)

	// Find IPC socket
	sockPath, err := findIPCSocket(cmd.Process.Pid)
	if err != nil {
		// Dump logs for debugging
		t.Logf("=== STDOUT ===")
		dumpPipe(t, stdout)
		t.Logf("=== STDERR ===")
		dumpPipe(t, stderr)
		t.Fatalf("Failed to find IPC socket: %v", err)
	}

	t.Logf("Found IPC socket: %s", sockPath)

	// Test 1: Launch first prism
	resp := sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-A",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to start prism-A: %v", resp["error"])
	}
	t.Logf("✓ Launched prism-A")

	time.Sleep(500 * time.Millisecond)

	// Test 2: Launch second prism
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "start",
		"prism":  "test-prism prism-B",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to start prism-B: %v", resp["error"])
	}
	t.Logf("✓ Launched prism-B")

	time.Sleep(500 * time.Millisecond)

	// Test 3: Verify status (should have 2 prisms)
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "status",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Status command failed: %v", resp["error"])
	}

	foreground := resp["foreground"].(string)
	background := resp["background"].([]interface{})

	if foreground == "" {
		t.Fatalf("No foreground prism")
	}
	if len(background) != 1 {
		t.Fatalf("Expected 1 background prism, got %d", len(background))
	}

	t.Logf("✓ Status verified: foreground=%s, background=%v", foreground, background)

	// Test 4: Kill one prism
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "kill",
		"prism":  foreground,
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to kill prism: %v", resp["error"])
	}
	t.Logf("✓ Killed foreground prism: %s", foreground)

	time.Sleep(500 * time.Millisecond)

	// Test 5: Verify only 1 prism remains
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "status",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Status command failed: %v", resp["error"])
	}

	foreground = resp["foreground"].(string)
	background = resp["background"].([]interface{})

	if foreground == "" {
		t.Fatalf("No foreground prism after kill")
	}
	if len(background) != 0 {
		t.Fatalf("Expected 0 background prisms, got %d", len(background))
	}

	t.Logf("✓ Cleanup verified: 1 prism remaining (%s)", foreground)

	// Test 6: Stop prismctl
	resp = sendIPCCommand(t, sockPath, map[string]string{
		"action": "stop",
	})
	if !resp["success"].(bool) {
		t.Fatalf("Failed to stop prismctl: %v", resp["error"])
	}
	t.Logf("✓ prismctl stopped cleanly")

	// Wait for process to exit
	cmd.Wait()
}

// Helper: Find IPC socket for prismctl PID
func findIPCSocket(pid int) (string, error) {
	pattern := fmt.Sprintf("/run/user/%d/shine/prism-*.%d.sock", os.Getuid(), pid)
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", err
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no socket found matching %s", pattern)
	}
	return matches[0], nil
}

// Helper: Send IPC command and parse response
func sendIPCCommand(t *testing.T, sockPath string, cmd map[string]string) map[string]interface{} {
	conn, err := net.Dial("unix", sockPath)
	if err != nil {
		t.Fatalf("Failed to connect to IPC socket: %v", err)
	}
	defer conn.Close()

	// Send command
	cmdBytes, _ := json.Marshal(cmd)
	conn.Write(cmdBytes)
	conn.Write([]byte("\n"))

	// Read response
	reader := bufio.NewReader(conn)
	respLine, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(respLine), &resp); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	return resp
}

// Helper: Dump pipe output
func dumpPipe(t *testing.T, pipe interface{}) {
	scanner := bufio.NewScanner(pipe.(interface{ Read([]byte) (int, error) }))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) != "" {
			t.Logf("%s", line)
		}
	}
}
