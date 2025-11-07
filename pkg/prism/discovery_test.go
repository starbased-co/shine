package prism

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/starbased-co/shine/pkg/config"
)

// setupTestDir creates a temporary directory with test binaries
func setupTestDir(t *testing.T) (string, func()) {
	tmpDir, err := os.MkdirTemp("", "prism-prism-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// createTestBinary creates an executable file in the given directory
func createTestBinary(t *testing.T, dir, name string) string {
	path := filepath.Join(dir, name)
	f, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}
	f.Close()

	// Make it executable
	if err := os.Chmod(path, 0755); err != nil {
		t.Fatalf("Failed to make binary executable: %v", err)
	}

	return path
}

func TestNewManager(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	searchPaths := []string{tmpDir}
	pm := NewManager(searchPaths)

	if pm == nil {
		t.Fatal("NewManager returned nil")
	}

	if pm.cache == nil {
		t.Error("Cache should be initialized")
	}
}

func TestAugmentPATH(t *testing.T) {
	tmpDir1, cleanup1 := setupTestDir(t)
	defer cleanup1()
	tmpDir2, cleanup2 := setupTestDir(t)
	defer cleanup2()

	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	searchPaths := []string{tmpDir1, tmpDir2}
	_ = NewManager(searchPaths)

	// Verify PATH was augmented
	newPath := os.Getenv("PATH")

	if !strings.Contains(newPath, tmpDir1) {
		t.Errorf("PATH doesn't contain %s", tmpDir1)
	}

	if !strings.Contains(newPath, tmpDir2) {
		t.Errorf("PATH doesn't contain %s", tmpDir2)
	}

	// Verify original PATH is preserved
	if !strings.Contains(newPath, origPath) {
		t.Error("Original PATH not preserved")
	}

	// Verify prism dirs come before original PATH
	pathParts := strings.Split(newPath, string(os.PathListSeparator))
	if len(pathParts) < 3 {
		t.Fatal("PATH doesn't have enough parts")
	}

	if pathParts[0] != tmpDir1 {
		t.Errorf("Expected first PATH entry to be %s, got %s", tmpDir1, pathParts[0])
	}

	if pathParts[1] != tmpDir2 {
		t.Errorf("Expected second PATH entry to be %s, got %s", tmpDir2, pathParts[1])
	}
}

func TestFindPrism_InCache(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	binaryPath := createTestBinary(t, tmpDir, "prism-bar")

	pm := NewManager([]string{tmpDir})

	// Pre-populate cache
	pm.cache["bar"] = binaryPath

	// Should find in cache
	path, err := pm.FindPrism("bar", nil)
	if err != nil {
		t.Fatalf("Failed to find prism: %v", err)
	}

	if path != binaryPath {
		t.Errorf("Expected path %s, got %s", binaryPath, path)
	}
}

func TestFindPrism_ExplicitBinaryPath(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	binaryPath := createTestBinary(t, tmpDir, "custom-binary")

	pm := NewManager([]string{})

	cfg := &config.PrismConfig{
		Binary: binaryPath,
	}

	path, err := pm.FindPrism("test", cfg)
	if err != nil {
		t.Fatalf("Failed to find prism: %v", err)
	}

	if path != binaryPath {
		t.Errorf("Expected path %s, got %s", binaryPath, path)
	}
}

func TestFindPrism_InPATH(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createTestBinary(t, tmpDir, "prism-bar")

	// Add to PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)
	os.Setenv("PATH", tmpDir+string(os.PathListSeparator)+origPath)

	pm := NewManager([]string{})

	path, err := pm.FindPrism("bar", nil)
	if err != nil {
		t.Fatalf("Failed to find prism: %v", err)
	}

	if !strings.Contains(path, "prism-bar") {
		t.Errorf("Expected path to contain prism-bar, got %s", path)
	}
}

func TestFindPrism_InSearchPaths(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	createTestBinary(t, tmpDir, "prism-clock")

	pm := NewManager([]string{tmpDir})

	path, err := pm.FindPrism("clock", nil)
	if err != nil {
		t.Fatalf("Failed to find prism: %v", err)
	}

	if !strings.Contains(path, "prism-clock") {
		t.Errorf("Expected path to contain prism-clock, got %s", path)
	}

	// Verify it was cached
	if cachedPath, ok := pm.cache["clock"]; !ok || cachedPath != path {
		t.Error("Prism not properly cached")
	}
}

func TestFindPrism_NotFound(t *testing.T) {
	pm := NewManager([]string{})

	_, err := pm.FindPrism("nonexistent", nil)
	if err == nil {
		t.Error("Expected error for nonexistent prism")
	}

	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("Expected 'not found' error, got: %v", err)
	}
}

func TestFindPrism_CustomBinaryName(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Create binary with custom name
	createTestBinary(t, tmpDir, "custom-widget")

	pm := NewManager([]string{tmpDir})

	cfg := &config.PrismConfig{
		Binary: "custom-widget",
	}

	path, err := pm.FindPrism("weather", cfg)
	if err != nil {
		t.Fatalf("Failed to find prism: %v", err)
	}

	if !strings.Contains(path, "custom-widget") {
		t.Errorf("Expected path to contain custom-widget, got %s", path)
	}
}

func TestIsExecutable(t *testing.T) {
	tmpDir, cleanup := setupTestDir(t)
	defer cleanup()

	// Executable file
	execPath := createTestBinary(t, tmpDir, "test-exec")
	if !isExecutable(execPath) {
		t.Error("Expected file to be executable")
	}

	// Non-executable file
	nonExecPath := filepath.Join(tmpDir, "test-nonexec")
	f, err := os.Create(nonExecPath)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	f.Close()

	if isExecutable(nonExecPath) {
		t.Error("Expected file to be non-executable")
	}

	// Non-existent file
	if isExecutable("/nonexistent/path") {
		t.Error("Expected non-existent file to return false")
	}

	// Directory
	if isExecutable(tmpDir) {
		t.Error("Expected directory to return false")
	}
}

func TestExpandPaths(t *testing.T) {
	// Save original HOME
	origHome := os.Getenv("HOME")
	defer os.Setenv("HOME", origHome)

	testHome := "/test/home"
	os.Setenv("HOME", testHome)

	tests := []struct {
		input    []string
		expected []string
	}{
		{
			[]string{"~/config", "/usr/lib"},
			[]string{filepath.Join(testHome, "config"), "/usr/lib"},
		},
		{
			[]string{"/usr/lib", "/opt/bin"},
			[]string{"/usr/lib", "/opt/bin"},
		},
	}

	for _, tt := range tests {
		result := expandPaths(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("expandPaths(%v) length mismatch: got %d, want %d",
				tt.input, len(result), len(tt.expected))
			continue
		}

		for i, expected := range tt.expected {
			if result[i] != expected {
				t.Errorf("expandPaths(%v)[%d] = %q, want %q",
					tt.input, i, result[i], expected)
			}
		}
	}
}

func TestExpandPaths_EnvVars(t *testing.T) {
	// Set test environment variable
	os.Setenv("TEST_VAR", "/test/path")
	defer os.Unsetenv("TEST_VAR")

	paths := []string{"$TEST_VAR/subdir", "/usr/lib"}
	result := expandPaths(paths)

	expected := "/test/path/subdir"
	if result[0] != expected {
		t.Errorf("Expected %s, got %s", expected, result[0])
	}
}

func TestFindPrism_PriorityOrder(t *testing.T) {
	tmpDir1, cleanup1 := setupTestDir(t)
	defer cleanup1()
	tmpDir2, cleanup2 := setupTestDir(t)
	defer cleanup2()

	// Create same binary in both directories
	path1 := createTestBinary(t, tmpDir1, "prism-bar")
	createTestBinary(t, tmpDir2, "prism-bar")

	// tmpDir1 should have priority (listed first)
	pm := NewManager([]string{tmpDir1, tmpDir2})

	foundPath, err := pm.FindPrism("bar", nil)
	if err != nil {
		t.Fatalf("Failed to find prism: %v", err)
	}

	if foundPath != path1 {
		t.Errorf("Expected priority path %s, got %s", path1, foundPath)
	}
}
