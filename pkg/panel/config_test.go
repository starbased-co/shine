package panel

import (
	"testing"
)

func TestParseDimension(t *testing.T) {
	tests := []struct {
		name      string
		input     interface{}
		want      Dimension
		wantError bool
	}{
		{
			name:  "integer value",
			input: 80,
			want:  Dimension{Value: 80, IsPixels: false},
		},
		{
			name:  "int64 value",
			input: int64(100),
			want:  Dimension{Value: 100, IsPixels: false},
		},
		{
			name:  "float64 value",
			input: float64(50),
			want:  Dimension{Value: 50, IsPixels: false},
		},
		{
			name:  "pixel string",
			input: "1200px",
			want:  Dimension{Value: 1200, IsPixels: true},
		},
		{
			name:  "numeric string",
			input: "24",
			want:  Dimension{Value: 24, IsPixels: false},
		},
		{
			name:      "invalid pixel string",
			input:     "abcpx",
			wantError: true,
		},
		{
			name:      "invalid string",
			input:     "invalid",
			wantError: true,
		},
		{
			name:      "unsupported type",
			input:     true,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseDimension(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseDimension() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDimension() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ParseDimension() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDimensionString(t *testing.T) {
	tests := []struct {
		name string
		dim  Dimension
		want string
	}{
		{
			name: "cells",
			dim:  Dimension{Value: 80, IsPixels: false},
			want: "80",
		},
		{
			name: "pixels",
			dim:  Dimension{Value: 1200, IsPixels: true},
			want: "1200px",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.dim.String()
			if got != tt.want {
				t.Errorf("Dimension.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePosition(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Position
		wantError bool
	}{
		{
			name:  "empty string",
			input: "",
			want:  Position{},
		},
		{
			name:  "integer coordinates",
			input: "100,50",
			want: Position{
				X: Dimension{Value: 100, IsPixels: false},
				Y: Dimension{Value: 50, IsPixels: false},
			},
		},
		{
			name:  "pixel coordinates",
			input: "200px,100px",
			want: Position{
				X: Dimension{Value: 200, IsPixels: true},
				Y: Dimension{Value: 100, IsPixels: true},
			},
		},
		{
			name:  "mixed coordinates",
			input: "100,50px",
			want: Position{
				X: Dimension{Value: 100, IsPixels: false},
				Y: Dimension{Value: 50, IsPixels: true},
			},
		},
		{
			name:  "coordinates with spaces",
			input: "100 , 50",
			want: Position{
				X: Dimension{Value: 100, IsPixels: false},
				Y: Dimension{Value: 50, IsPixels: false},
			},
		},
		{
			name:      "invalid format - single value",
			input:     "100",
			wantError: true,
		},
		{
			name:      "invalid format - three values",
			input:     "100,50,25",
			wantError: true,
		},
		{
			name:      "invalid x coordinate",
			input:     "abc,50",
			wantError: true,
		},
		{
			name:      "invalid y coordinate",
			input:     "100,xyz",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePosition(tt.input)
			if tt.wantError {
				if err == nil {
					t.Errorf("ParsePosition() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParsePosition() unexpected error: %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("ParsePosition() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParseAnchor(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Anchor
	}{
		{"top", "top", AnchorTop},
		{"bottom", "bottom", AnchorBottom},
		{"left", "left", AnchorLeft},
		{"right", "right", AnchorRight},
		{"center", "center", AnchorCenter},
		{"none", "none", AnchorNone},
		{"center-sized", "center-sized", AnchorCenterSized},
		{"background", "background", AnchorBackground},
		{"top-left", "top-left", AnchorTopLeft},
		{"top-right", "top-right", AnchorTopRight},
		{"bottom-left", "bottom-left", AnchorBottomLeft},
		{"bottom-right", "bottom-right", AnchorBottomRight},
		{"absolute", "absolute", AnchorAbsolute},
		{"invalid", "invalid", AnchorCenter}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseAnchor(tt.input)
			if got != tt.want {
				t.Errorf("ParseAnchor(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestAnchorString(t *testing.T) {
	tests := []struct {
		anchor Anchor
		want   string
	}{
		{AnchorTop, "top"},
		{AnchorBottom, "bottom"},
		{AnchorLeft, "left"},
		{AnchorRight, "right"},
		{AnchorCenter, "center"},
		{AnchorNone, "none"},
		{AnchorCenterSized, "center-sized"},
		{AnchorBackground, "background"},
		{AnchorTopLeft, "top-left"},
		{AnchorTopRight, "top-right"},
		{AnchorBottomLeft, "bottom-left"},
		{AnchorBottomRight, "bottom-right"},
		{AnchorAbsolute, "absolute"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.anchor.String()
			if got != tt.want {
				t.Errorf("Anchor.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Anchor != AnchorCenter {
		t.Errorf("NewConfig().Anchor = %v, want %v", cfg.Anchor, AnchorCenter)
	}

	if cfg.OutputName != "DP-2" {
		t.Errorf("NewConfig().OutputName = %q, want %q", cfg.OutputName, "DP-2")
	}

	if cfg.Type != LayerShellPanel {
		t.Errorf("NewConfig().Type = %v, want %v", cfg.Type, LayerShellPanel)
	}

	if cfg.FocusPolicy != FocusNotAllowed {
		t.Errorf("NewConfig().FocusPolicy = %v, want %v", cfg.FocusPolicy, FocusNotAllowed)
	}

	if cfg.ExclusiveZone != -1 {
		t.Errorf("NewConfig().ExclusiveZone = %d, want -1", cfg.ExclusiveZone)
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
		{"invalid", FocusNotAllowed},
	}

	for _, tt := range tests {
		result := ParseFocusPolicy(tt.input)
		if result != tt.expected {
			t.Errorf("ParseFocusPolicy(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestToKittenArgs_AbsoluteAnchor(t *testing.T) {
	cfg := &Config{
		Type:       LayerShellPanel,
		Anchor:     AnchorAbsolute,
		Width:      Dimension{Value: 200, IsPixels: true},
		Height:     Dimension{Value: 100, IsPixels: true},
		OutputName: "DP-2",
	}

	args := cfg.ToKittenArgs("/usr/bin/component")

	found := false
	for _, arg := range args {
		if arg == "--edge=center" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ToKittenArgs() with absolute anchor should include --edge=center, got %v", args)
	}

	foundWidth := false
	foundHeight := false
	for _, arg := range args {
		if arg == "--columns=200px" {
			foundWidth = true
		}
		if arg == "--lines=100px" {
			foundHeight = true
		}
	}
	if !foundWidth {
		t.Errorf("ToKittenArgs() missing --columns=200px in %v", args)
	}
	if !foundHeight {
		t.Errorf("ToKittenArgs() missing --lines=100px in %v", args)
	}
}

func TestToKittenArgs_TopRightCorner(t *testing.T) {
	cfg := &Config{
		Type:       LayerShellPanel,
		Anchor:     AnchorTopRight,
		Width:      Dimension{Value: 150, IsPixels: true},
		Height:     Dimension{Value: 30, IsPixels: true},
		OutputName: "DP-2",
	}

	args := cfg.ToKittenArgs("/usr/bin/component")

	found := false
	for _, arg := range args {
		if arg == "--edge=top" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ToKittenArgs() with top-right anchor should include --edge=top, got %v", args)
	}
}

func TestToKittenArgs_StandardFlags(t *testing.T) {
	cfg := &Config{
		Type:             LayerShellPanel,
		Anchor:           AnchorTop,
		Width:            Dimension{Value: 80, IsPixels: false},
		Height:           Dimension{Value: 1, IsPixels: false},
		FocusPolicy:      FocusOnDemand,
		HideOnFocusLoss:  true,
		ToggleVisibility: true,
		OutputName:       "DP-2",
		ListenSocket:     "/tmp/test.sock",
	}

	args := cfg.ToKittenArgs("/usr/bin/component")

	expectedFlags := []string{
		"--edge=top",
		"--columns=80",
		"--lines=1",
		"--focus-policy=on-demand",
		"--hide-on-focus-loss",
		"--single-instance",
		"--instance-group=shine",
		"--toggle-visibility",
		"--output-name=DP-2",
		"-o",
		"allow_remote_control=socket-only",
		"-o",
		"listen_on=unix:/tmp/test.sock",
		"/usr/bin/component",
	}

	for _, expected := range expectedFlags {
		found := false
		for _, arg := range args {
			if arg == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ToKittenArgs() missing expected flag %q in %v", expected, args)
		}
	}
}

func TestToKittenArgs_PixelDimensions(t *testing.T) {
	cfg := &Config{
		Type:       LayerShellPanel,
		Anchor:     AnchorBottom,
		Width:      Dimension{Value: 600, IsPixels: true},
		Height:     Dimension{Value: 120, IsPixels: true},
		OutputName: "DP-2",
	}

	args := cfg.ToKittenArgs("/usr/bin/component")

	foundWidth := false
	foundHeight := false
	for _, arg := range args {
		if arg == "--columns=600px" {
			foundWidth = true
		}
		if arg == "--lines=120px" {
			foundHeight = true
		}
	}

	if !foundWidth {
		t.Errorf("ToKittenArgs() missing --columns=600px in %v", args)
	}
	if !foundHeight {
		t.Errorf("ToKittenArgs() missing --lines=120px in %v", args)
	}
}

func TestToRemoteControlArgs(t *testing.T) {
	cfg := &Config{
		Type:        LayerShellPanel,
		Anchor:      AnchorCenter,
		Width:       Dimension{Value: 1200, IsPixels: true},
		Height:      Dimension{Value: 600, IsPixels: true},
		FocusPolicy: FocusExclusive,
		WindowTitle: "test-window",
		OutputName:  "DP-2",
	}

	args := cfg.ToRemoteControlArgs("/usr/bin/component")

	if len(args) < 3 || args[0] != "@" || args[1] != "launch" || args[2] != "--type=os-panel" {
		t.Errorf("ToRemoteControlArgs() should start with [@, launch, --type=os-panel], got %v", args[:3])
	}

	foundTitle := false
	for i, arg := range args {
		if arg == "--title" && i+1 < len(args) && args[i+1] == "test-window" {
			foundTitle = true
			break
		}
	}
	if !foundTitle {
		t.Errorf("ToRemoteControlArgs() missing --title test-window in %v", args)
	}

	if args[len(args)-1] != "/usr/bin/component" {
		t.Errorf("ToRemoteControlArgs() component path should be last, got %v", args)
	}
}
