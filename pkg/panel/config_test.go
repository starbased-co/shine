package panel

import (
	"testing"
)

func TestEdgeParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected Edge
	}{
		{"top", EdgeTop},
		{"bottom", EdgeBottom},
		{"left", EdgeLeft},
		{"right", EdgeRight},
		{"center", EdgeCenter},
		{"center-sized", EdgeCenterSized},
		{"background", EdgeBackground},
		{"none", EdgeNone},
		{"invalid", EdgeTop}, // Default to top
	}

	for _, tt := range tests {
		result := ParseEdge(tt.input)
		if result != tt.expected {
			t.Errorf("ParseEdge(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestFocusPolicyParsing(t *testing.T) {
	tests := []struct {
		input    string
		expected FocusPolicy
	}{
		{"not-allowed", FocusNotAllowed},
		{"exclusive", FocusExclusive},
		{"on-demand", FocusOnDemand},
		{"invalid", FocusNotAllowed}, // Default to not-allowed
	}

	for _, tt := range tests {
		result := ParseFocusPolicy(tt.input)
		if result != tt.expected {
			t.Errorf("ParseFocusPolicy(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestToKittenArgs(t *testing.T) {
	cfg := NewConfig()
	cfg.Edge = EdgeBottom
	cfg.Lines = 10
	cfg.MarginLeft = 10
	cfg.MarginRight = 10
	cfg.MarginBottom = 10
	cfg.HideOnFocusLoss = true
	cfg.FocusPolicy = FocusOnDemand
	cfg.ListenSocket = "/tmp/test.sock"

	args := cfg.ToKittenArgs("shine-chat")

	// Verify key arguments are present
	expectedArgs := []string{
		"panel",
		"--edge=bottom",
		"--lines=10",
		"--margin-left=10",
		"--margin-right=10",
		"--margin-bottom=10",
		"--focus-policy=on-demand",
		"--hide-on-focus-loss",
		"--single-instance",
		"-o",
		"allow_remote_control=socket-only",
		"-o",
		"listen_on=unix:/tmp/test.sock",
		"shine-chat",
	}

	argsMap := make(map[string]bool)
	for _, arg := range args {
		argsMap[arg] = true
	}

	for _, expected := range expectedArgs {
		if !argsMap[expected] {
			t.Errorf("Expected arg %q not found in: %v", expected, args)
		}
	}
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Type != LayerShellPanel {
		t.Errorf("Expected type=LayerShellPanel, got %v", cfg.Type)
	}

	if cfg.Edge != EdgeTop {
		t.Errorf("Expected edge=EdgeTop, got %v", cfg.Edge)
	}

	if cfg.FocusPolicy != FocusNotAllowed {
		t.Errorf("Expected focus_policy=FocusNotAllowed, got %v", cfg.FocusPolicy)
	}

	if cfg.ExclusiveZone != -1 {
		t.Errorf("Expected exclusive_zone=-1, got %d", cfg.ExclusiveZone)
	}
}
