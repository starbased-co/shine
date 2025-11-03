package panel

import (
	"strings"
	"testing"
)

// TestKittenArgsGeneration verifies that ToKittenArgs generates correct CLI arguments
func TestKittenArgsGeneration(t *testing.T) {
	cfg := NewConfig()
	cfg.Edge = EdgeBottom
	cfg.Lines = 10
	cfg.MarginLeft = 10
	cfg.MarginRight = 10
	cfg.MarginBottom = 10
	cfg.HideOnFocusLoss = true
	cfg.FocusPolicy = FocusOnDemand
	cfg.ListenSocket = "/tmp/shine-chat.sock"

	args := cfg.ToKittenArgs("shine-chat")
	argsStr := strings.Join(args, " ")

	t.Logf("Generated kitten args: %s", argsStr)

	// Verify structure matches what kitten panel expects
	if args[0] != "panel" {
		t.Errorf("First arg should be 'panel', got %q", args[0])
	}

	if args[len(args)-1] != "shine-chat" {
		t.Errorf("Last arg should be component name, got %q", args[len(args)-1])
	}

	// Verify critical flags are present
	if !strings.Contains(argsStr, "--edge=bottom") {
		t.Error("Missing --edge=bottom")
	}

	if !strings.Contains(argsStr, "--lines=10") {
		t.Error("Missing --lines=10")
	}

	if !strings.Contains(argsStr, "--margin-left=10") {
		t.Error("Missing --margin-left=10")
	}

	if !strings.Contains(argsStr, "--hide-on-focus-loss") {
		t.Error("Missing --hide-on-focus-loss")
	}

	if !strings.Contains(argsStr, "--single-instance") {
		t.Error("Missing --single-instance")
	}

	if !strings.Contains(argsStr, "allow_remote_control=socket-only") {
		t.Error("Missing remote control flag")
	}

	if !strings.Contains(argsStr, "unix:/tmp/shine-chat.sock") {
		t.Error("Missing listen socket")
	}
}

// TestPixelSizing verifies pixel-based size specification
func TestPixelSizing(t *testing.T) {
	cfg := NewConfig()
	cfg.LinesPixels = 200
	cfg.ColumnsPixels = 800

	args := cfg.ToKittenArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--lines=200px") {
		t.Error("Missing --lines=200px")
	}

	if !strings.Contains(argsStr, "--columns=800px") {
		t.Error("Missing --columns=800px")
	}
}

// TestBackgroundEdge verifies background edge sets correct layer
func TestBackgroundEdge(t *testing.T) {
	cfg := NewConfig()
	cfg.Edge = EdgeBackground
	cfg.Type = LayerShellBackground

	args := cfg.ToKittenArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--edge=background") {
		t.Error("Missing --edge=background")
	}

	// Background should also set layer
	if !strings.Contains(argsStr, "--layer=background") {
		t.Error("Missing --layer=background for background edge")
	}
}

// TestOutputName verifies monitor targeting
func TestOutputName(t *testing.T) {
	cfg := NewConfig()
	cfg.OutputName = "DP-1"

	args := cfg.ToKittenArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--output-name=DP-1") {
		t.Error("Missing --output-name=DP-1")
	}
}
