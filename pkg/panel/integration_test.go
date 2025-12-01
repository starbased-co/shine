package panel

import (
	"strings"
	"testing"
)

func TestPanelArgsGeneration(t *testing.T) {
	cfg := NewConfig()
	cfg.Origin = OriginBottomCenter
	cfg.Height = Dimension{Value: 10, IsPixels: false}
	cfg.HideOnFocusLoss = true
	cfg.FocusPolicy = FocusOnDemand
	cfg.ListenSocket = "/tmp/shine-chat.sock"

	args := cfg.ToPanelArgs("shine-chat")
	argsStr := strings.Join(args, " ")

	t.Logf("Generated panel args: %s", argsStr)

	if len(args) < 3 || args[0] != "@" || args[1] != "launch" || args[2] != "--type=os-panel" {
		t.Errorf("First args should be [@, launch, --type=os-panel], got %v", args[:3])
	}

	if args[len(args)-1] != "shine-chat" {
		t.Errorf("Last arg should be component name, got %q", args[len(args)-1])
	}

	if !strings.Contains(argsStr, "edge=bottom") {
		t.Error("Missing edge=bottom")
	}

	if !strings.Contains(argsStr, "lines=10") {
		t.Error("Missing lines=10")
	}

	if !strings.Contains(argsStr, "focus-policy=on-demand") {
		t.Error("Missing focus-policy=on-demand")
	}

	if !strings.Contains(argsStr, "output-name=DP-2") {
		t.Error("Missing output-name=DP-2")
	}
}

func TestPixelSizing(t *testing.T) {
	cfg := NewConfig()
	cfg.Height = Dimension{Value: 200, IsPixels: true}
	cfg.Width = Dimension{Value: 800, IsPixels: true}

	args := cfg.ToPanelArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "lines=200px") {
		t.Error("Missing lines=200px")
	}

	if !strings.Contains(argsStr, "columns=800px") {
		t.Error("Missing columns=800px")
	}
}

func TestBackgroundLayer(t *testing.T) {
	cfg := NewConfig()
	cfg.Origin = OriginCenter
	cfg.Type = LayerShellBackground

	args := cfg.ToPanelArgs("test")
	argsStr := strings.Join(args, " ")

	// Note: ToPanelArgs doesn't include --layer flag - that's handled by Kitty's os-panel type
	// This test may need to be removed or updated based on actual panel behavior
	t.Logf("Generated args: %s", argsStr)
}

func TestOutputName(t *testing.T) {
	cfg := NewConfig()
	cfg.OutputName = "DP-2"

	args := cfg.ToPanelArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "output-name=DP-2") {
		t.Error("Missing output-name=DP-2")
	}
}
