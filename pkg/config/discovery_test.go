package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/starbased-co/shine/pkg/paths"
)

func TestDiscoverDirectoryPrism(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	prismDir := filepath.Join(tmpDir, "weather")
	if err := os.Mkdir(prismDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create prism.toml
	manifestContent := `name = "weather"
version = "1.0.0"
path = "shine-weather"
enabled = true
origin = "top-right"
width = "400px"
height = "30px"

[metadata]
description = "Weather widget"
author = "Test Author"
license = "MIT"
`
	manifestPath := filepath.Join(prismDir, "prism.toml")
	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test discovery
	prism, err := discoverDirectoryPrism(tmpDir, "weather")
	if err != nil {
		t.Fatalf("Failed to discover directory prism: %v", err)
	}

	if prism.Config.Name != "weather" {
		t.Errorf("Expected name 'weather', got '%s'", prism.Config.Name)
	}

	if prism.Config.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", prism.Config.Version)
	}

	if prism.Config.Origin != "top-right" {
		t.Errorf("Expected origin 'top-right', got '%s'", prism.Config.Origin)
	}

	// Verify metadata was loaded
	if prism.Config.Metadata == nil {
		t.Error("Metadata should not be nil")
	} else {
		if desc, ok := prism.Config.Metadata["description"].(string); !ok || desc != "Weather widget" {
			t.Errorf("Expected metadata description 'Weather widget', got '%v'", prism.Config.Metadata["description"])
		}
	}

	if prism.Source != SourcePrismDir {
		t.Errorf("Expected source SourcePrismDir, got %v", prism.Source)
	}
}

func TestDiscoverStandalonePrism(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create standalone clock.toml
	configContent := `name = "clock"
version = "1.0.0"
path = "shine-clock"
enabled = true
origin = "top-right"
width = "150px"
height = "30px"

[metadata]
description = "Simple clock"
author = "Clock Author"
`
	configPath := filepath.Join(tmpDir, "clock.toml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test discovery
	prism, err := discoverStandalonePrism(tmpDir, "clock.toml")
	if err != nil {
		t.Fatalf("Failed to discover standalone prism: %v", err)
	}

	if prism.Config.Name != "clock" {
		t.Errorf("Expected name 'clock', got '%s'", prism.Config.Name)
	}

	if prism.Config.Width.(string) != "150px" {
		t.Errorf("Expected width '150px', got '%v'", prism.Config.Width)
	}

	// Verify metadata was loaded
	if prism.Config.Metadata == nil {
		t.Error("Metadata should not be nil")
	}

	if prism.Source != SourceStandaloneTOML {
		t.Errorf("Expected source SourceStandaloneTOML, got %v", prism.Source)
	}
}

func TestDiscoverPrisms(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Type 1: Full package (directory with binary)
	weatherDir := filepath.Join(tmpDir, "weather")
	os.Mkdir(weatherDir, 0755)
	weatherManifest := `name = "weather"
version = "1.0.0"
path = "shine-weather"
enabled = true
`
	os.WriteFile(filepath.Join(weatherDir, "prism.toml"), []byte(weatherManifest), 0644)

	// Type 2: Data directory (directory without binary)
	spotifyDir := filepath.Join(tmpDir, "spotify")
	os.Mkdir(spotifyDir, 0755)
	spotifyManifest := `name = "spotify"
version = "2.0.0"
path = "shine-spotify"
enabled = false
origin = "bottom"
`
	os.WriteFile(filepath.Join(spotifyDir, "prism.toml"), []byte(spotifyManifest), 0644)

	// Type 3: Standalone config
	clockConfig := `name = "clock"
version = "1.0.0"
enabled = true
origin = "top-right"
width = "150px"
`
	os.WriteFile(filepath.Join(tmpDir, "clock.toml"), []byte(clockConfig), 0644)

	// Discover all prisms
	discovered, err := DiscoverPrisms([]string{tmpDir})
	if err != nil {
		t.Fatalf("DiscoverPrisms failed: %v", err)
	}

	// Verify all three were discovered
	if len(discovered) != 3 {
		t.Errorf("Expected 3 prisms, got %d", len(discovered))
	}

	if _, ok := discovered["weather"]; !ok {
		t.Error("weather prism not discovered")
	}

	if _, ok := discovered["spotify"]; !ok {
		t.Error("spotify prism not discovered")
	}

	if _, ok := discovered["clock"]; !ok {
		t.Error("clock prism not discovered")
	}

	// Verify specific fields
	if discovered["weather"].Config.Version != "1.0.0" {
		t.Errorf("weather version mismatch")
	}

	if discovered["spotify"].Config.Origin != "bottom" {
		t.Errorf("spotify origin mismatch")
	}

	if discovered["clock"].Config.Width.(string) != "150px" {
		t.Errorf("clock width mismatch")
	}
}

func TestMergePrismConfigs(t *testing.T) {
	// Prism source (from prism.toml with metadata)
	prismSource := &PrismConfig{
		Name:    "test",
		Version: "1.0.0",
		Path:    "shine-test",
		Enabled: false,
		Origin:  "top",
		Width:   "200px",
		Height:  "50px",
		Metadata: map[string]interface{}{
			"description": "Test prism",
			"author":      "Test Author",
			"license":     "MIT",
		},
	}

	// User config (from shine.toml, no metadata)
	userConfig := &PrismConfig{
		Name:    "test",
		Enabled: true,
		Width:   "300px",
		Origin:  "bottom",
		Metadata: map[string]interface{}{
			"should": "be ignored",
		},
	}

	// Merge
	merged := MergePrismConfigs(prismSource, userConfig)

	// User overrides should win
	if !merged.Enabled {
		t.Error("Expected enabled=true from user config")
	}

	if merged.Width.(string) != "300px" {
		t.Errorf("Expected width='300px' from user config, got '%v'", merged.Width)
	}

	if merged.Origin != "bottom" {
		t.Errorf("Expected origin='bottom' from user config, got '%s'", merged.Origin)
	}

	// Prism source should provide defaults
	if merged.Height.(string) != "50px" {
		t.Errorf("Expected height='50px' from prism source, got '%v'", merged.Height)
	}

	if merged.Version != "1.0.0" {
		t.Errorf("Expected version='1.0.0' from prism source, got '%s'", merged.Version)
	}

	// CRITICAL: Metadata should come ONLY from prism source
	if merged.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if desc, ok := merged.Metadata["description"].(string); !ok || desc != "Test prism" {
		t.Error("Metadata should be from prism source, not user config")
	}

	if _, exists := merged.Metadata["should"]; exists {
		t.Error("User metadata should be ignored")
	}
}

func TestMetadataFromPrismSourceTakesPriority(t *testing.T) {
	// Prism source with metadata
	prismSource := &PrismConfig{
		Name:    "test",
		Version: "1.0.0",
		Enabled: false,
		Metadata: map[string]interface{}{
			"description": "Official description",
			"author":      "Prism Author",
		},
	}

	// User config in shine.toml (even if it has metadata, it's ignored)
	userConfig := &PrismConfig{
		Name:    "test",
		Enabled: true,
		Metadata: map[string]interface{}{
			"description": "User description",
			"author":      "User",
			"custom":      "field",
		},
	}

	// Merge
	merged := MergePrismConfigs(prismSource, userConfig)

	// Metadata should ONLY come from prism source
	if merged.Metadata == nil {
		t.Fatal("Metadata should not be nil")
	}

	if desc, ok := merged.Metadata["description"].(string); !ok || desc != "Official description" {
		t.Error("Metadata should be from prism source, not user config")
	}

	if author, ok := merged.Metadata["author"].(string); !ok || author != "Prism Author" {
		t.Error("Metadata author should be from prism source")
	}

	// User's custom metadata should not appear
	if _, exists := merged.Metadata["custom"]; exists {
		t.Error("User metadata fields should not appear in merged config")
	}

	// But user's other settings should win
	if !merged.Enabled {
		t.Error("User's enabled setting should take priority")
	}
}

func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string // What the result should contain
	}{
		{
			name:     "tilde expansion",
			input:    "~/test",
			contains: "/test",
		},
		{
			name:     "tilde only",
			input:    "~",
			contains: "/",
		},
		{
			name:     "absolute path",
			input:    "/usr/bin",
			contains: "/usr/bin",
		},
		{
			name:     "relative path",
			input:    "bin",
			contains: "bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := paths.ExpandHome(tt.input)
			if tt.contains != "" && len(result) > 0 {
				// Basic check that path was processed
				if tt.input == "~" && result == "~" {
					t.Error("Tilde should be expanded")
				}
			}
		})
	}
}
