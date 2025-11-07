package panel

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
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

// Anchor represents panel anchor placement
type Anchor int

const (
	AnchorTop Anchor = iota
	AnchorBottom
	AnchorLeft
	AnchorRight
	AnchorCenter
	AnchorNone
	AnchorCenterSized
	AnchorBackground
	AnchorTopLeft
	AnchorTopRight
	AnchorBottomLeft
	AnchorBottomRight
	AnchorAbsolute
)

func (a Anchor) String() string {
	switch a {
	case AnchorTop:
		return "top"
	case AnchorBottom:
		return "bottom"
	case AnchorLeft:
		return "left"
	case AnchorRight:
		return "right"
	case AnchorCenter:
		return "center"
	case AnchorNone:
		return "none"
	case AnchorCenterSized:
		return "center-sized"
	case AnchorBackground:
		return "background"
	case AnchorTopLeft:
		return "top-left"
	case AnchorTopRight:
		return "top-right"
	case AnchorBottomLeft:
		return "bottom-left"
	case AnchorBottomRight:
		return "bottom-right"
	case AnchorAbsolute:
		return "absolute"
	default:
		return "center"
	}
}

// ParseAnchor converts string to Anchor
func ParseAnchor(s string) Anchor {
	switch s {
	case "top":
		return AnchorTop
	case "bottom":
		return AnchorBottom
	case "left":
		return AnchorLeft
	case "right":
		return AnchorRight
	case "center":
		return AnchorCenter
	case "none":
		return AnchorNone
	case "center-sized":
		return AnchorCenterSized
	case "background":
		return AnchorBackground
	case "top-left":
		return AnchorTopLeft
	case "top-right":
		return AnchorTopRight
	case "bottom-left":
		return AnchorBottomLeft
	case "bottom-right":
		return AnchorBottomRight
	case "absolute":
		return AnchorAbsolute
	default:
		return AnchorCenter
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

// Dimension represents a size value (int for cells or string with "px" for pixels)
type Dimension struct {
	Value    int
	IsPixels bool
}

// ParseDimension parses a dimension value from either int or string with "px" suffix
func ParseDimension(v interface{}) (Dimension, error) {
	switch val := v.(type) {
	case int:
		return Dimension{Value: val, IsPixels: false}, nil
	case int64:
		return Dimension{Value: int(val), IsPixels: false}, nil
	case float64:
		return Dimension{Value: int(val), IsPixels: false}, nil
	case string:
		if strings.HasSuffix(val, "px") {
			px := strings.TrimSuffix(val, "px")
			num, err := strconv.Atoi(px)
			if err != nil {
				return Dimension{}, fmt.Errorf("invalid pixel value: %s", val)
			}
			return Dimension{Value: num, IsPixels: true}, nil
		}
		num, err := strconv.Atoi(val)
		if err != nil {
			return Dimension{}, fmt.Errorf("invalid dimension value: %s", val)
		}
		return Dimension{Value: num, IsPixels: false}, nil
	default:
		return Dimension{}, fmt.Errorf("unsupported dimension type: %T", v)
	}
}

// String formats dimension for CLI args
func (d Dimension) String() string {
	if d.IsPixels {
		return fmt.Sprintf("%dpx", d.Value)
	}
	return strconv.Itoa(d.Value)
}

// Position represents x,y coordinates (can be cells or pixels)
type Position struct {
	X Dimension
	Y Dimension
}

// ParsePosition parses position from "x,y" string
func ParsePosition(s string) (Position, error) {
	if s == "" {
		return Position{}, nil
	}

	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return Position{}, fmt.Errorf("invalid position format: %s (expected x,y)", s)
	}

	x, err := ParseDimension(strings.TrimSpace(parts[0]))
	if err != nil {
		return Position{}, fmt.Errorf("invalid x coordinate: %w", err)
	}

	y, err := ParseDimension(strings.TrimSpace(parts[1]))
	if err != nil {
		return Position{}, fmt.Errorf("invalid y coordinate: %w", err)
	}

	return Position{X: x, Y: y}, nil
}

// Config represents layer shell panel configuration
type Config struct {
	// Layer shell properties
	Type        LayerType
	Anchor      Anchor
	FocusPolicy FocusPolicy

	// Size (simplified to width/height)
	Width  Dimension // Width in columns or pixels (e.g., 80 or "1200px")
	Height Dimension // Height in lines or pixels (e.g., 24 or "600px")

	// Position relative to anchor point
	Position Position // Position as "x,y" (e.g., "100,50" or "200px,100px")

	// Margins (now used as refinement offsets)
	MarginTop    int
	MarginLeft   int
	MarginBottom int
	MarginRight  int

	// Exclusive zone
	ExclusiveZone         int
	OverrideExclusiveZone bool

	// Behavior
	HideOnFocusLoss  bool
	ToggleVisibility bool

	// Output (CRITICAL: Must be DP-2, never DP-1)
	OutputName string // Monitor name (e.g., "DP-2")

	// Remote control
	ListenSocket string // Unix socket path

	// Window identification
	WindowTitle string // Window title for targeting specific windows
}

// NewConfig creates a default panel configuration
func NewConfig() *Config {
	return &Config{
		Type:          LayerShellPanel,
		Anchor:        AnchorCenter, // Default changed from "top" to "center"
		FocusPolicy:   FocusNotAllowed,
		Width:         Dimension{Value: 1, IsPixels: false},
		Height:        Dimension{Value: 1, IsPixels: false},
		ExclusiveZone: -1, // Auto
		OutputName:    "DP-2", // CRITICAL: Default to DP-2
	}
}

// getMonitorResolution queries Hyprland for monitor dimensions
func getMonitorResolution(monitorName string) (width, height int, err error) {
	if monitorName == "" {
		monitorName = "DP-2" // CRITICAL: Changed from DP-1 to DP-2
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

// calculateMargins computes final margins from anchor, position, and refinements
func (c *Config) calculateMargins() (top, left, bottom, right int, err error) {
	// Get monitor dimensions
	monWidth, monHeight, err := getMonitorResolution(c.OutputName)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get monitor resolution: %w", err)
	}

	// Convert panel dimensions to pixels
	panelWidth := c.Width.Value
	if !c.Width.IsPixels {
		panelWidth = c.Width.Value * 10 // Estimate: 10px per column
	}

	panelHeight := c.Height.Value
	if !c.Height.IsPixels {
		panelHeight = c.Height.Value * 20 // Estimate: 20px per line
	}

	// Convert position to pixels
	posX := c.Position.X.Value
	if !c.Position.X.IsPixels {
		posX = c.Position.X.Value * 10 // Estimate: 10px per column
	}

	posY := c.Position.Y.Value
	if !c.Position.Y.IsPixels {
		posY = c.Position.Y.Value * 20 // Estimate: 20px per line
	}

	// Calculate base margins based on anchor and position
	switch c.Anchor {
	case AnchorAbsolute:
		// Absolute positioning: top-left is (0,0)
		// Position directly from top-left corner
		left = posX
		top = posY

	case AnchorCenter:
		// Center anchor: (0,0) is screen center
		centerX := monWidth / 2
		centerY := monHeight / 2
		left = centerX + posX - (panelWidth / 2)
		top = centerY + posY - (panelHeight / 2)

	case AnchorTop:
		// Top edge: y=0 at top, x=0 at left
		left = posX
		top = posY

	case AnchorBottom:
		// Bottom edge: y=0 at bottom, x=0 at left
		left = posX
		bottom = posY

	case AnchorLeft:
		// Left edge: x=0 at left, y=0 at top
		left = posX
		top = posY

	case AnchorRight:
		// Right edge: x=0 at right, y=0 at top
		right = posX
		top = posY

	case AnchorTopLeft:
		// Top-left corner: (0,0) at corner
		left = posX
		top = posY

	case AnchorTopRight:
		// Top-right corner: (0,0) at corner
		right = posX
		top = posY

	case AnchorBottomLeft:
		// Bottom-left corner: (0,0) at corner
		left = posX
		bottom = posY

	case AnchorBottomRight:
		// Bottom-right corner: (0,0) at corner
		right = posX
		bottom = posY

	default:
		// Default to center behavior
		centerX := monWidth / 2
		centerY := monHeight / 2
		left = centerX + posX - (panelWidth / 2)
		top = centerY + posY - (panelHeight / 2)
	}

	// Apply margin refinements
	top += c.MarginTop
	left += c.MarginLeft
	bottom += c.MarginBottom
	right += c.MarginRight

	return top, left, bottom, right, nil
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

	// Anchor (use "center" for absolute mode internally)
	anchorStr := c.Anchor.String()
	if c.Anchor == AnchorAbsolute {
		anchorStr = "center"
	}
	if anchorStr != "" {
		panelProps = append(panelProps, fmt.Sprintf("edge=%s", anchorStr))
	}

	// Size
	if c.Width.Value > 0 {
		panelProps = append(panelProps, fmt.Sprintf("columns=%s", c.Width.String()))
	}
	if c.Height.Value > 0 {
		panelProps = append(panelProps, fmt.Sprintf("lines=%s", c.Height.String()))
	}

	// Calculate and apply margins
	top, left, bottom, right, err := c.calculateMargins()
	if err == nil {
		if top > 0 {
			panelProps = append(panelProps, fmt.Sprintf("margin-top=%d", top))
		}
		if left > 0 {
			panelProps = append(panelProps, fmt.Sprintf("margin-left=%d", left))
		}
		if bottom > 0 {
			panelProps = append(panelProps, fmt.Sprintf("margin-bottom=%d", bottom))
		}
		if right > 0 {
			panelProps = append(panelProps, fmt.Sprintf("margin-right=%d", right))
		}
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

	// Window title
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

	// Anchor to edge mapping (use "center" for absolute mode)
	edgeStr := c.Anchor.String()
	if c.Anchor == AnchorAbsolute {
		edgeStr = "center"
	}

	// Corner anchors map to their primary edge
	switch c.Anchor {
	case AnchorTopLeft, AnchorTopRight:
		edgeStr = "top"
	case AnchorBottomLeft, AnchorBottomRight:
		edgeStr = "bottom"
	}

	args = append(args, "--edge="+edgeStr)

	// Layer type
	if c.Type != LayerShellPanel {
		args = append(args, "--layer="+c.Type.String())
	}

	// Size
	if c.Width.Value > 0 {
		args = append(args, "--columns="+c.Width.String())
	}
	if c.Height.Value > 0 {
		args = append(args, "--lines="+c.Height.String())
	}

	// Calculate margins
	top, left, bottom, right, err := c.calculateMargins()
	if err == nil {
		if top > 0 {
			args = append(args, fmt.Sprintf("--margin-top=%d", top))
		}
		if left > 0 {
			args = append(args, fmt.Sprintf("--margin-left=%d", left))
		}
		if bottom > 0 {
			args = append(args, fmt.Sprintf("--margin-bottom=%d", bottom))
		}
		if right > 0 {
			args = append(args, fmt.Sprintf("--margin-right=%d", right))
		}
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

	// Output (CRITICAL: Must be DP-2)
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
