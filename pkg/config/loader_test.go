package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad(t *testing.T) {
	// Create temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")

	configContent := `[chat]
enabled = true
edge = "bottom"
lines = 10
margin_left = 10
margin_right = 10
margin_bottom = 10
hide_on_focus_loss = true
focus_policy = "on-demand"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load config
	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify chat config
	if cfg.Chat == nil {
		t.Fatal("Chat config is nil")
	}

	if !cfg.Chat.Enabled {
		t.Error("Expected chat to be enabled")
	}

	if cfg.Chat.Edge != "bottom" {
		t.Errorf("Expected edge=bottom, got %s", cfg.Chat.Edge)
	}

	if cfg.Chat.Lines != 10 {
		t.Errorf("Expected lines=10, got %d", cfg.Chat.Lines)
	}

	if cfg.Chat.MarginLeft != 10 {
		t.Errorf("Expected margin_left=10, got %d", cfg.Chat.MarginLeft)
	}

	if !cfg.Chat.HideOnFocusLoss {
		t.Error("Expected hide_on_focus_loss=true")
	}

	if cfg.Chat.FocusPolicy != "on-demand" {
		t.Errorf("Expected focus_policy=on-demand, got %s", cfg.Chat.FocusPolicy)
	}
}

func TestLoadOrDefault(t *testing.T) {
	// Test with non-existent file
	cfg := LoadOrDefault("/tmp/nonexistent-config-file-xyz.toml")
	if cfg == nil {
		t.Fatal("Expected default config, got nil")
	}

	// Default config should have core and prisms
	if cfg.Core == nil {
		t.Fatal("Expected default core config, got nil")
	}

	if cfg.Prisms == nil {
		t.Fatal("Expected default prisms, got nil")
	}

	// Default should have bar prism enabled
	if barCfg, ok := cfg.Prisms["bar"]; !ok || !barCfg.Enabled {
		t.Error("Expected default bar prism to be enabled")
	}
}

func TestSave(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "saved.toml")

	cfg := NewDefaultConfig()
	if err := Save(cfg, configPath); err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("Config file not created: %v", err)
	}

	// Load it back
	loaded, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loaded.Chat.Enabled != cfg.Chat.Enabled {
		t.Error("Saved and loaded config don't match")
	}
}

func TestLoad_PrismConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")

	configContent := `[core]
path = ["/usr/lib/shine/bin", "~/.config/shine/bin"]

[prisms.bar]
enabled = true
edge = "top"
lines_pixels = 30
focus_policy = "not-allowed"

[prisms.weather]
enabled = true
name = "weather"
binary = "shine-weather"
edge = "top-right"
columns_pixels = 200
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg, err := Load(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify core config
	if cfg.Core == nil {
		t.Fatal("Core config is nil")
	}

	paths := cfg.Core.GetPaths()
	if len(paths) != 2 {
		t.Errorf("Expected 2 paths, got %d", len(paths))
	}

	// Verify prisms
	if cfg.Prisms == nil {
		t.Fatal("Prisms map is nil")
	}

	barCfg, ok := cfg.Prisms["bar"]
	if !ok {
		t.Fatal("Bar prism not found")
	}

	if !barCfg.Enabled {
		t.Error("Expected bar to be enabled")
	}

	if barCfg.Edge != "top" {
		t.Errorf("Expected edge 'top', got %s", barCfg.Edge)
	}

	weatherCfg, ok := cfg.Prisms["weather"]
	if !ok {
		t.Fatal("Weather prism not found")
	}

	if weatherCfg.Binary != "shine-weather" {
		t.Errorf("Expected binary 'shine-weather', got %s", weatherCfg.Binary)
	}
}

func TestLoadOrDefault_BackwardCompatibility(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")

	// Old config format
	configContent := `[bar]
enabled = true
edge = "top"
lines_pixels = 30

[chat]
enabled = false
edge = "bottom"
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Capture stderr for deprecation warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	cfg := LoadOrDefault(configPath)

	w.Close()
	os.Stderr = oldStderr

	var buf [1024]byte
	n, _ := r.Read(buf[:])
	output := string(buf[:n])

	// Check for deprecation warning
	if !strings.Contains(output, "deprecated") {
		t.Error("Expected deprecation warning for old config format")
	}

	// Verify migration
	if cfg.Prisms == nil {
		t.Fatal("Prisms map is nil")
	}

	barCfg, ok := cfg.Prisms["bar"]
	if !ok {
		t.Fatal("Bar prism not migrated")
	}

	if !barCfg.Enabled {
		t.Error("Expected bar to be enabled after migration")
	}

	chatCfg, ok := cfg.Prisms["chat"]
	if !ok {
		t.Fatal("Chat prism not migrated")
	}

	if chatCfg.Enabled {
		t.Error("Expected chat to be disabled after migration")
	}
}

func TestLoadOrDefault_MixedFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")

	// Mix of new and old format
	configContent := `[core]
path = ["~/.config/shine/bin"]

[prisms.bar]
enabled = true
edge = "top"

[chat]
enabled = false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg := LoadOrDefault(configPath)

	// Both should be migrated
	if _, ok := cfg.Prisms["bar"]; !ok {
		t.Error("Bar prism not found")
	}

	if _, ok := cfg.Prisms["chat"]; !ok {
		t.Error("Chat prism not migrated")
	}
}

func TestLoadOrDefault_InitializesDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test.toml")

	// Minimal config without core
	configContent := `[prisms.bar]
enabled = true
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg := LoadOrDefault(configPath)

	// Should have default core config
	if cfg.Core == nil {
		t.Fatal("Core config should be initialized")
	}

	paths := cfg.Core.GetPaths()
	if len(paths) == 0 {
		t.Error("Should have default paths")
	}
}

func TestPrismConfig_ToPanelConfig(t *testing.T) {
	prismCfg := &PrismConfig{
		Name:          "test",
		Enabled:       true,
		Edge:          "top",
		LinesPixels:   30,
		MarginTop:     10,
		MarginLeft:    20,
		FocusPolicy:   "not-allowed",
		OutputName:    "DP-2",
	}

	panelCfg := prismCfg.ToPanelConfig()

	if panelCfg == nil {
		t.Fatal("Panel config is nil")
	}

	if panelCfg.Height.Value != 30 || !panelCfg.Height.IsPixels {
		t.Errorf("Expected height 30px, got %+v", panelCfg.Height)
	}

	if panelCfg.MarginTop != 10 {
		t.Errorf("Expected margin_top 10, got %d", panelCfg.MarginTop)
	}

	if panelCfg.MarginLeft != 20 {
		t.Errorf("Expected margin_left 20, got %d", panelCfg.MarginLeft)
	}

	if panelCfg.OutputName != "DP-2" {
		t.Errorf("Expected output_name 'DP-2', got %s", panelCfg.OutputName)
	}
}

func TestNewDefaultConfig_HasPrisms(t *testing.T) {
	cfg := NewDefaultConfig()

	if cfg == nil {
		t.Fatal("Default config is nil")
	}

	if cfg.Core == nil {
		t.Fatal("Default config missing core")
	}

	if cfg.Prisms == nil {
		t.Fatal("Default config missing prisms")
	}

	// Should have at least bar prism
	barCfg, ok := cfg.Prisms["bar"]
	if !ok {
		t.Fatal("Default config missing bar prism")
	}

	if !barCfg.Enabled {
		t.Error("Default bar prism should be enabled")
	}
}
