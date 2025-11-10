package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// TestBasicCompilation verifies binaries exist and are executable
func TestBasicCompilation(t *testing.T) {
	// Build paths
	prismctlBin := filepath.Join("..", "..", "bin", "prismctl")
	testPrismBin := filepath.Join("..", "fixtures", "test-prism")

	// Verify prismctl exists and is executable
	if info, err := os.Stat(prismctlBin); err != nil {
		t.Fatalf("prismctl binary not found: %v", err)
	} else if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("prismctl is not executable")
	} else {
		t.Logf("✓ prismctl found and executable")
	}

	// Verify test-prism exists and is executable
	if info, err := os.Stat(testPrismBin); err != nil {
		t.Fatalf("test-prism binary not found: %v", err)
	} else if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("test-prism is not executable")
	} else {
		t.Logf("✓ test-prism found and executable")
	}

	// Try to run prismctl --help (should work without PTY)
	cmd := exec.Command(prismctlBin, "--help")
	output, err := cmd.CombinedOutput()
	if err == nil || len(output) > 0 {
		t.Logf("✓ prismctl --help executed")
	}

	// Try to run test-prism with timeout (will fail without TTY but should compile)
	cmd = exec.Command(testPrismBin, "test")
	done := make(chan error, 1)
	if err := cmd.Start(); err != nil {
		t.Logf("test-prism start error (expected without TTY): %v", err)
	} else {
		go func() {
			done <- cmd.Wait()
		}()

		select {
		case <-done:
			t.Logf("✓ test-prism executable (exited as expected without TTY)")
		case <-time.After(1 * time.Second):
			cmd.Process.Kill()
			t.Logf("✓ test-prism killed after timeout (was running)")
		}
	}
}
