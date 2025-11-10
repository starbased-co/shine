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
				X: 100,
				Y: 50,
			},
		},
		{
			name:      "pixel coordinates should fail",
			input:     "200px,100px",
			wantError: true,
		},
		{
			name:      "mixed coordinates should fail",
			input:     "100,50px",
			wantError: true,
		},
		{
			name:  "coordinates with spaces",
			input: "100 , 50",
			want: Position{
				X: 100,
				Y: 50,
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

func TestParseOrigin(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  Origin
	}{
		{"top-left", "top-left", OriginTopLeft},
		{"top-center", "top-center", OriginTopCenter},
		{"top-right", "top-right", OriginTopRight},
		{"left-center", "left-center", OriginLeftCenter},
		{"center", "center", OriginCenter},
		{"center-sized", "center-sized", OriginCenterSized},
		{"right-center", "right-center", OriginRightCenter},
		{"bottom-left", "bottom-left", OriginBottomLeft},
		{"bottom-center", "bottom-center", OriginBottomCenter},
		{"bottom-right", "bottom-right", OriginBottomRight},
		{"invalid", "invalid", OriginCenter}, // Default
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseOrigin(tt.input)
			if got != tt.want {
				t.Errorf("ParseOrigin(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestOriginString(t *testing.T) {
	tests := []struct {
		origin Origin
		want   string
	}{
		{OriginTopLeft, "top-left"},
		{OriginTopCenter, "top-center"},
		{OriginTopRight, "top-right"},
		{OriginLeftCenter, "left-center"},
		{OriginCenter, "center"},
		{OriginCenterSized, "center-sized"},
		{OriginRightCenter, "right-center"},
		{OriginBottomLeft, "bottom-left"},
		{OriginBottomCenter, "bottom-center"},
		{OriginBottomRight, "bottom-right"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.origin.String()
			if got != tt.want {
				t.Errorf("Origin.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg.Origin != OriginTopLeft {
		t.Errorf("NewConfig().Origin = %v, want %v", cfg.Origin, OriginTopLeft)
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

func TestToKittenArgs_Center(t *testing.T) {
	cfg := &Config{
		Type:       LayerShellPanel,
		Origin:     OriginCenter,
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
		t.Errorf("ToKittenArgs() with center origin should include --edge=center, got %v", args)
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
		Origin:     OriginTopRight,
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
		t.Errorf("ToKittenArgs() with top-right origin should include --edge=top, got %v", args)
	}
}

func TestToKittenArgs_StandardFlags(t *testing.T) {
	cfg := &Config{
		Type:             LayerShellPanel,
		Origin:           OriginTopCenter,
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
		Origin:     OriginBottomCenter,
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
		Origin:      OriginCenter,
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

func TestOriginCenterSized(t *testing.T) {
	t.Run("String conversion", func(t *testing.T) {
		if OriginCenterSized.String() != "center-sized" {
			t.Errorf("Expected 'center-sized', got '%s'", OriginCenterSized.String())
		}
	})

	t.Run("Parse from string", func(t *testing.T) {
		origin := ParseOrigin("center-sized")
		if origin != OriginCenterSized {
			t.Errorf("Expected OriginCenterSized, got %v", origin)
		}
	})

	t.Run("originToEdge", func(t *testing.T) {
		cfg := &Config{Origin: OriginCenterSized}
		edge := cfg.originToEdge()
		if edge != "center-sized" {
			t.Errorf("Expected 'center-sized', got '%s'", edge)
		}
	})

	t.Run("calculateMargins returns zero", func(t *testing.T) {
		cfg := &Config{
			Origin:     OriginCenterSized,
			Width:      Dimension{Value: 400, IsPixels: true},
			Height:     Dimension{Value: 300, IsPixels: true},
			OutputName: "DP-2",
		}

		top, left, bottom, right, err := cfg.calculateMargins()
		if err != nil {
			t.Fatalf("calculateMargins failed: %v", err)
		}

		if top != 0 || left != 0 || bottom != 0 || right != 0 {
			t.Errorf("Expected all margins to be 0, got top=%d, left=%d, bottom=%d, right=%d",
				top, left, bottom, right)
		}
	})

	t.Run("ToRemoteControlArgs omits margin args", func(t *testing.T) {
		cfg := &Config{
			Origin:     OriginCenterSized,
			Width:      Dimension{Value: 400, IsPixels: true},
			Height:     Dimension{Value: 300, IsPixels: true},
			OutputName: "DP-2",
		}

		args := cfg.ToRemoteControlArgs("/usr/bin/prism")

		hasEdge := false
		for _, arg := range args {
			if arg == "edge=center-sized" {
				hasEdge = true
			}
			if len(arg) >= 7 && arg[:7] == "margin-" {
				t.Errorf("Unexpected margin arg in center-sized: %s", arg)
			}
		}
		if !hasEdge {
			t.Errorf("Expected 'edge=center-sized' in args: %v", args)
		}
	})
}

func TestOriginCenterFourMargins(t *testing.T) {
	t.Run("calculateMargins without offset", func(t *testing.T) {
		cfg := &Config{
			Origin:     OriginCenter,
			Width:      Dimension{Value: 400, IsPixels: true},
			Height:     Dimension{Value: 300, IsPixels: true},
			Position:   Position{X: 0, Y: 0},
			OutputName: "DP-2",
		}

		top, left, bottom, right, err := cfg.calculateMargins()
		if err != nil {
			t.Fatalf("calculateMargins failed: %v", err)
		}

		if top == 0 || left == 0 || bottom == 0 || right == 0 {
			t.Errorf("Expected all margins to be non-zero, got top=%d, left=%d, bottom=%d, right=%d",
				top, left, bottom, right)
		}

		if top != bottom {
			t.Errorf("Expected top == bottom, got top=%d, bottom=%d", top, bottom)
		}
		if left != right {
			t.Errorf("Expected left == right, got left=%d, right=%d", left, right)
		}
	})

	t.Run("calculateMargins with offset", func(t *testing.T) {
		cfg := &Config{
			Origin:     OriginCenter,
			Width:      Dimension{Value: 400, IsPixels: true},
			Height:     Dimension{Value: 300, IsPixels: true},
			Position:   Position{X: 50, Y: 100},
			OutputName: "DP-2",
		}

		top, left, bottom, right, err := cfg.calculateMargins()
		if err != nil {
			t.Fatalf("calculateMargins failed: %v", err)
		}

		if top == 0 || left == 0 || bottom == 0 || right == 0 {
			t.Errorf("Expected all margins to be non-zero, got top=%d, left=%d, bottom=%d, right=%d",
				top, left, bottom, right)
		}

		if top == bottom {
			t.Errorf("Expected top != bottom with offset, got top=%d, bottom=%d", top, bottom)
		}
		if left == right {
			t.Errorf("Expected left != right with offset, got left=%d, right=%d", left, right)
		}

		if left <= right {
			t.Errorf("Expected left > right (offsetX=50), got left=%d, right=%d", left, right)
		}
		if top <= bottom {
			t.Errorf("Expected top > bottom (offsetY=100), got top=%d, bottom=%d", top, bottom)
		}
	})

	t.Run("ToRemoteControlArgs includes all four margins", func(t *testing.T) {
		cfg := &Config{
			Origin:     OriginCenter,
			Width:      Dimension{Value: 400, IsPixels: true},
			Height:     Dimension{Value: 300, IsPixels: true},
			Position:   Position{X: 0, Y: 0},
			OutputName: "DP-2",
		}

		args := cfg.ToRemoteControlArgs("/usr/bin/prism")

		hasEdge := false
		hasTop := false
		hasLeft := false
		hasBottom := false
		hasRight := false

		for _, arg := range args {
			if arg == "edge=center" {
				hasEdge = true
			}
			if len(arg) >= 11 && arg[:11] == "margin-top=" {
				hasTop = true
			}
			if len(arg) >= 12 && arg[:12] == "margin-left=" {
				hasLeft = true
			}
			if len(arg) >= 14 && arg[:14] == "margin-bottom=" {
				hasBottom = true
			}
			if len(arg) >= 13 && arg[:13] == "margin-right=" {
				hasRight = true
			}
		}

		if !hasEdge {
			t.Errorf("Expected 'edge=center' in args")
		}
		if !hasTop || !hasLeft || !hasBottom || !hasRight {
			t.Errorf("Missing margin args: top=%v, left=%v, bottom=%v, right=%v",
				hasTop, hasLeft, hasBottom, hasRight)
		}
	})
}
