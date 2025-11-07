package prism

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create executable file
	execPath := filepath.Join(tmpDir, "test-binary")
	content := []byte("#!/bin/sh\necho test\n")
	if err := os.WriteFile(execPath, content, 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	result, err := Validate(execPath)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Valid {
		t.Errorf("Expected valid result, got invalid with errors: %v", result.Errors)
	}
}

func TestValidateNonExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	// Create non-executable file
	filePath := filepath.Join(tmpDir, "test-file")
	if err := os.WriteFile(filePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result, err := Validate(filePath)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result for non-executable file")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected error about non-executable file")
	}
}

func TestValidateNonExistent(t *testing.T) {
	result, err := Validate("/nonexistent/path/binary")
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if result.Valid {
		t.Error("Expected invalid result for nonexistent file")
	}

	if len(result.Errors) == 0 {
		t.Error("Expected error about nonexistent file")
	}
}

