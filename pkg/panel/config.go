package panel

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// LayerType represents the Wayland layer shell type
type LayerType int

const (
	LayerShellNone LayerType = iota
	LayerShellBackground
	LayerShellPanel
	LayerShellTop
	LayerShellOverlay
)

func (lt LayerType) String() string {
	switch lt {
	case LayerShellBackground:
		return "background"
	case LayerShellPanel:
		return "bottom"
	case LayerShellTop:
		return "top"
	case LayerShellOverlay:
		return "overlay"
	default:
		return ""
	}
}

// Edge represents panel edge placement
type Edge int

const (
	EdgeTop Edge = iota
	EdgeBottom
	EdgeLeft
	EdgeRight
	EdgeCenter
	EdgeNone
	EdgeCenterSized
	EdgeBackground
	EdgeTopLeft
	EdgeTopRight
	EdgeBottomLeft
	EdgeBottomRight
)

func (e Edge) String() string {
	switch e {
	case EdgeTop:
		return "top"
	case EdgeBottom:
		return "bottom"
	case EdgeLeft:
		return "left"
	case EdgeRight:
		return "right"
	case EdgeCenter:
		return "center"
	case EdgeNone:
		return "none"
	case EdgeCenterSized:
		return "center-sized"
	case EdgeBackground:
		return "background"
	case EdgeTopLeft:
		return "top-left"
	case EdgeTopRight:
		return "top-right"
	case EdgeBottomLeft:
		return "bottom-left"
	case EdgeBottomRight:
		return "bottom-right"
	default:
		return "top"
	}
}

// ParseEdge converts string to Edge
func ParseEdge(s string) Edge {
	switch s {
	case "top":
		return EdgeTop
	case "bottom":
		return EdgeBottom
	case "left":
		return EdgeLeft
	case "right":
		return EdgeRight
	case "center":
		return EdgeCenter
	case "none":
		return EdgeNone
	case "center-sized":
		return EdgeCenterSized
	case "background":
		return EdgeBackground
	case "top-left":
		return EdgeTopLeft
	case "top-right":
		return EdgeTopRight
	case "bottom-left":
		return EdgeBottomLeft
	case "bottom-right":
		return EdgeBottomRight
	default:
		return EdgeTop
	}
}

// FocusPolicy represents keyboard focus policy
type FocusPolicy int

const (
	FocusNotAllowed FocusPolicy = iota
	FocusExclusive
	FocusOnDemand
)

func (fp FocusPolicy) String() string {
	switch fp {
	case FocusNotAllowed:
		return "not-allowed"
	case FocusExclusive:
		return "exclusive"
	case FocusOnDemand:
		return "on-demand"
	default:
		return "not-allowed"
	}
}

// ParseFocusPolicy converts string to FocusPolicy
func ParseFocusPolicy(s string) FocusPolicy {
	switch s {
	case "not-allowed":
		return FocusNotAllowed
	case "exclusive":
		return FocusExclusive
	case "on-demand":
		return FocusOnDemand
	default:
		return FocusNotAllowed
	}
}

// Config represents layer shell panel configuration
// Ported from kitty/types.py:72-88 (LayerShellConfig)
type Config struct {
	// Layer shell properties
	Type        LayerType
	Edge        Edge
	FocusPolicy FocusPolicy

	// Size (cells or pixels)
	Lines         int // Height in terminal lines
	Columns       int // Width in terminal columns
	LinesPixels   int // Height in pixels (overrides Lines)
	ColumnsPixels int // Width in pixels (overrides Columns)

	// Margins
	MarginTop    int
	MarginLeft   int
	MarginBottom int
	MarginRight  int

	// Exclusive zone
	ExclusiveZone         int
	OverrideExclusiveZone bool

	// Behavior
	HideOnFocusLoss bool
	ToggleVisibility bool

	// Output
	OutputName string // Monitor name (e.g., "DP-1")

	// Remote control
	ListenSocket string // Unix socket path

	// Window identification for multi-window single instance
	WindowTitle string // Window title for targeting specific windows
}

// NewConfig creates a default panel configuration
func NewConfig() *Config {
	return &Config{
		Type:          LayerShellPanel,
		Edge:          EdgeTop,
		FocusPolicy:   FocusNotAllowed,
		Lines:         1,
		Columns:       1,
		ExclusiveZone: -1, // Auto
	}
}

// getMonitorResolution queries Hyprland for monitor dimensions
func getMonitorResolution(monitorName string) (width, height int, err error) {
	if monitorName == "" {
		monitorName = "DP-1" // Default, but this should be primary
	}

	cmd := exec.Command("hyprctl", "monitors", "-j")
	output, err := cmd.Output()
	if err != nil {
		return 0, 0, fmt.Errorf("failed to query monitors: %w", err)
	}

	var monitors []map[string]interface{}
	if err := json.Unmarshal(output, &monitors); err != nil {
		return 0, 0, fmt.Errorf("failed to parse monitor data: %w", err)
	}

	for _, mon := range monitors {
		if name, ok := mon["name"].(string); ok && name == monitorName {
			if w, ok := mon["width"].(float64); ok {
				if h, ok := mon["height"].(float64); ok {
					return int(w), int(h), nil
				}
			}
		}
	}

	return 0, 0, fmt.Errorf("monitor %s not found", monitorName)
}

// ToRemoteControlArgs converts Config to kitty @ launch arguments
func (c *Config) ToRemoteControlArgs(componentPath string) []string {
	args := []string{
		"@",
		"launch",
		"--type=os-panel",
	}

	// Panel properties via --os-panel
	panelProps := []string{}

	// Edge
	edgeStr := c.Edge.String()
	if edgeStr != "" {
		panelProps = append(panelProps, fmt.Sprintf("edge=%s", edgeStr))
	}

	// Size (prefer pixels, fall back to cells)
	if c.LinesPixels > 0 {
		panelProps = append(panelProps, fmt.Sprintf("lines=%dpx", c.LinesPixels))
	} else if c.Lines > 0 {
		panelProps = append(panelProps, fmt.Sprintf("lines=%d", c.Lines))
	}

	if c.ColumnsPixels > 0 {
		panelProps = append(panelProps, fmt.Sprintf("columns=%dpx", c.ColumnsPixels))
	} else if c.Columns > 0 {
		panelProps = append(panelProps, fmt.Sprintf("columns=%d", c.Columns))
	}

	// Margins
	if c.MarginTop > 0 {
		panelProps = append(panelProps, fmt.Sprintf("margin-top=%d", c.MarginTop))
	}
	if c.MarginLeft > 0 {
		panelProps = append(panelProps, fmt.Sprintf("margin-left=%d", c.MarginLeft))
	}
	if c.MarginBottom > 0 {
		panelProps = append(panelProps, fmt.Sprintf("margin-bottom=%d", c.MarginBottom))
	}
	if c.MarginRight > 0 {
		panelProps = append(panelProps, fmt.Sprintf("margin-right=%d", c.MarginRight))
	}

	// Focus policy
	if c.FocusPolicy != FocusNotAllowed {
		panelProps = append(panelProps, fmt.Sprintf("focus-policy=%s", c.FocusPolicy.String()))
	}

	// Output name
	if c.OutputName != "" {
		panelProps = append(panelProps, fmt.Sprintf("output-name=%s", c.OutputName))
	}

	// Add each panel property with its own --os-panel flag
	for _, prop := range panelProps {
		args = append(args, "--os-panel", prop)
	}

	// Window title (for tracking/control)
	if c.WindowTitle != "" {
		args = append(args, "--title", c.WindowTitle)
	}

	// Component path
	args = append(args, componentPath)

	return args
}

// ToKittenArgs converts Config to kitten panel CLI arguments
func (c *Config) ToKittenArgs(component string) []string {
	args := []string{"panel"}

	// Calculate actual margins for corner positioning
	marginTop := c.MarginTop
	marginLeft := c.MarginLeft
	marginBottom := c.MarginBottom
	marginRight := c.MarginRight

	// Edge placement - corner modes need calculated margins
	edgeStr := c.Edge.String()
	isCorner := false

	switch c.Edge {
	case EdgeTopLeft, EdgeTopRight:
		edgeStr = "top"
		isCorner = true
	case EdgeBottomLeft, EdgeBottomRight:
		edgeStr = "bottom"
		isCorner = true
	}

	// For corner positioning, calculate margins based on monitor resolution
	if isCorner {
		monWidth, _, err := getMonitorResolution(c.OutputName)
		if err == nil {
			// Get panel dimensions
			panelWidth := c.ColumnsPixels
			if panelWidth == 0 {
				panelWidth = c.Columns * 10 // Rough estimate: 10px per column
			}
			panelHeight := c.LinesPixels
			if panelHeight == 0 {
				panelHeight = c.Lines * 20 // Rough estimate: 20px per line
			}

			switch c.Edge {
			case EdgeTopRight:
				// Position on right side
				marginLeft = monWidth - panelWidth - marginRight
			case EdgeTopLeft:
				// Left side is default, keep marginLeft as is
			case EdgeBottomRight:
				// Position on right side
				marginLeft = monWidth - panelWidth - marginRight
			case EdgeBottomLeft:
				// Left side is default, keep marginLeft as is
			}
		}
	}

	args = append(args, "--edge="+edgeStr)

	// Layer type (only if not default)
	if c.Type != LayerShellPanel {
		args = append(args, "--layer="+c.Type.String())
	}

	// Size specification
	if c.LinesPixels > 0 {
		args = append(args, "--lines="+strconv.Itoa(c.LinesPixels)+"px")
	} else if c.Lines > 0 {
		args = append(args, "--lines="+strconv.Itoa(c.Lines))
	}

	if c.ColumnsPixels > 0 {
		args = append(args, "--columns="+strconv.Itoa(c.ColumnsPixels)+"px")
	} else if c.Columns > 0 {
		args = append(args, "--columns="+strconv.Itoa(c.Columns))
	}

	// Margins (use calculated values for corners)
	if marginTop > 0 {
		args = append(args, fmt.Sprintf("--margin-top=%d", marginTop))
	}
	if marginLeft > 0 {
		args = append(args, fmt.Sprintf("--margin-left=%d", marginLeft))
	}
	if marginBottom > 0 {
		args = append(args, fmt.Sprintf("--margin-bottom=%d", marginBottom))
	}
	if marginRight > 0 {
		args = append(args, fmt.Sprintf("--margin-right=%d", marginRight))
	}

	// Focus policy
	if c.FocusPolicy != FocusNotAllowed {
		args = append(args, "--focus-policy="+c.FocusPolicy.String())
	}

	// Exclusive zone
	if c.OverrideExclusiveZone {
		args = append(args, fmt.Sprintf("--exclusive-zone=%d", c.ExclusiveZone))
		args = append(args, "--override-exclusive-zone")
	}

	// Behavior flags
	if c.HideOnFocusLoss {
		args = append(args, "--hide-on-focus-loss")
	}
	args = append(args, "--single-instance")
	args = append(args, "--instance-group=shine")
	if c.ToggleVisibility {
		args = append(args, "--toggle-visibility")
	}

	// Output
	if c.OutputName != "" {
		args = append(args, "--output-name="+c.OutputName)
	}

	// Remote control
	if c.ListenSocket != "" {
		args = append(args, "-o", "allow_remote_control=socket-only")
		args = append(args, "-o", "listen_on=unix:"+c.ListenSocket)
	}

	// Component binary
	args = append(args, component)

	return args
}
