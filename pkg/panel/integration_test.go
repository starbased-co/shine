package panel

import (
	"strings"
	"testing"
)

func TestKittenArgsGeneration(t *testing.T) {
	cfg := NewConfig()
	cfg.Anchor = AnchorBottom
	cfg.Height = Dimension{Value: 10, IsPixels: false}
	cfg.MarginLeft = 10
	cfg.MarginRight = 10
	cfg.MarginBottom = 10
	cfg.HideOnFocusLoss = true
	cfg.FocusPolicy = FocusOnDemand
	cfg.ListenSocket = "/tmp/shine-chat.sock"

	args := cfg.ToKittenArgs("shine-chat")
	argsStr := strings.Join(args, " ")

	t.Logf("Generated kitten args: %s", argsStr)

	if args[0] != "panel" {
		t.Errorf("First arg should be 'panel', got %q", args[0])
	}

	if args[len(args)-1] != "shine-chat" {
		t.Errorf("Last arg should be component name, got %q", args[len(args)-1])
	}

	if !strings.Contains(argsStr, "--edge=bottom") {
		t.Error("Missing --edge=bottom")
	}

	if !strings.Contains(argsStr, "--lines=10") {
		t.Error("Missing --lines=10")
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

func TestPixelSizing(t *testing.T) {
	cfg := NewConfig()
	cfg.Height = Dimension{Value: 200, IsPixels: true}
	cfg.Width = Dimension{Value: 800, IsPixels: true}

	args := cfg.ToKittenArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--lines=200px") {
		t.Error("Missing --lines=200px")
	}

	if !strings.Contains(argsStr, "--columns=800px") {
		t.Error("Missing --columns=800px")
	}
}

func TestBackgroundAnchor(t *testing.T) {
	cfg := NewConfig()
	cfg.Anchor = AnchorBackground
	cfg.Type = LayerShellBackground

	args := cfg.ToKittenArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--edge=background") {
		t.Error("Missing --edge=background")
	}

	if !strings.Contains(argsStr, "--layer=background") {
		t.Error("Missing --layer=background for background anchor")
	}
}

func TestOutputName(t *testing.T) {
	cfg := NewConfig()
	cfg.OutputName = "DP-2"

	args := cfg.ToKittenArgs("test")
	argsStr := strings.Join(args, " ")

	if !strings.Contains(argsStr, "--output-name=DP-2") {
		t.Error("Missing --output-name=DP-2")
	}
}
