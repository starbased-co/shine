package prism

import (
	"os"
	"path/filepath"
	"testing"
)

func TestManifestParsing(t *testing.T) {
	// Create temporary directory for test
	tmpDir := t.TempDir()

	manifestContent := `
[prism]
name = "test-prism"
version = "1.0.0"
binary = "shine-test"

[metadata]
description = "Test prism"
author = "Test Author"
license = "MIT"
homepage = "https://example.com"
`

	manifestPath := filepath.Join(tmpDir, "prism.toml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write test manifest: %v", err)
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatalf("Failed to load manifest: %v", err)
	}

	if manifest.Prism.Name != "test-prism" {
		t.Errorf("Expected name 'test-prism', got '%s'", manifest.Prism.Name)
	}

	if manifest.Prism.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", manifest.Prism.Version)
	}

	if manifest.Prism.Binary != "shine-test" {
		t.Errorf("Expected binary 'shine-test', got '%s'", manifest.Prism.Binary)
	}
}

func TestManifestValidation(t *testing.T) {
	tests := []struct {
		name        string
		manifest    *Manifest
		expectError bool
	}{
		{
			name: "valid manifest",
			manifest: &Manifest{
				Prism: PrismInfo{
					Name:    "test",
					Version: "1.0.0",
					Binary:  "shine-test",
				},
			},
			expectError: false,
		},
		{
			name: "missing name",
			manifest: &Manifest{
				Prism: PrismInfo{
					Version: "1.0.0",
					Binary:  "shine-test",
				},
			},
			expectError: true,
		},
		{
			name: "missing version",
			manifest: &Manifest{
				Prism: PrismInfo{
					Name:   "test",
					Binary: "shine-test",
				},
			},
			expectError: false,
		},
		{
			name: "missing binary",
			manifest: &Manifest{
				Prism: PrismInfo{
					Name:    "test",
					Version: "1.0.0",
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected validation error, got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}
		})
	}
}

func TestDiscoveryByManifest(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test prism directory with manifest
	prismDir := filepath.Join(tmpDir, "weather")
	if err := os.Mkdir(prismDir, 0755); err != nil {
		t.Fatalf("Failed to create prism dir: %v", err)
	}

	manifestContent := `
[prism]
name = "weather"
version = "1.0.0"
binary = "shine-weather"

[metadata]
description = "Weather widget"
`
	manifestPath := filepath.Join(prismDir, "prism.toml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Create dummy binary
	binaryPath := filepath.Join(prismDir, "shine-weather")
	if err := os.WriteFile(binaryPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create binary: %v", err)
	}

	// Test discovery
	prisms, err := DiscoverByManifest([]string{tmpDir})
	if err != nil {
		t.Fatalf("Discovery failed: %v", err)
	}

	if len(prisms) != 1 {
		t.Fatalf("Expected 1 prism, found %d", len(prisms))
	}

	if path, ok := prisms["weather"]; !ok {
		t.Error("Expected to find 'weather' prism")
	} else if path != binaryPath {
		t.Errorf("Expected path '%s', got '%s'", binaryPath, path)
	}
}

func TestFindManifestDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test prism directory
	prismDir := filepath.Join(tmpDir, "test-prism")
	if err := os.Mkdir(prismDir, 0755); err != nil {
		t.Fatalf("Failed to create prism dir: %v", err)
	}

	manifestContent := `
[prism]
name = "test-prism"
version = "1.0.0"
binary = "shine-test-prism"
`
	manifestPath := filepath.Join(prismDir, "prism.toml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to write manifest: %v", err)
	}

	// Test finding manifest
	foundDir, manifest, err := FindManifestDir([]string{tmpDir}, "test-prism")
	if err != nil {
		t.Fatalf("Failed to find manifest: %v", err)
	}

	if foundDir != prismDir {
		t.Errorf("Expected dir '%s', got '%s'", prismDir, foundDir)
	}

	if manifest.Prism.Name != "test-prism" {
		t.Errorf("Expected name 'test-prism', got '%s'", manifest.Prism.Name)
	}

	// Test not found
	_, _, err = FindManifestDir([]string{tmpDir}, "nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent prism, got nil")
	}
}
