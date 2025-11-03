package config

import (
	"os"
	"path/filepath"
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

	if cfg.Chat == nil {
		t.Fatal("Expected default chat config, got nil")
	}

	if !cfg.Chat.Enabled {
		t.Error("Expected default chat to be enabled")
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
