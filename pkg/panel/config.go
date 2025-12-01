package panel

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

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

type Origin int

const (
	OriginTopLeft Origin = iota
	OriginTopCenter
	OriginTopRight
	OriginLeftCenter
	OriginCenter
	OriginCenterSized // Auto-centers with no margin calculations
	OriginRightCenter
	OriginBottomLeft
	OriginBottomCenter
	OriginBottomRight
)

func (o Origin) String() string {
	switch o {
	case OriginTopLeft:
		return "top-left"
	case OriginTopCenter:
		return "top-center"
	case OriginTopRight:
		return "top-right"
	case OriginLeftCenter:
		return "left-center"
	case OriginCenter:
		return "center"
	case OriginCenterSized:
		return "center-sized"
	case OriginRightCenter:
		return "right-center"
	case OriginBottomLeft:
		return "bottom-left"
	case OriginBottomCenter:
		return "bottom-center"
	case OriginBottomRight:
		return "bottom-right"
	default:
		return "center"
	}
}

func ParseOrigin(s string) Origin {
	switch s {
	case "top-left":
		return OriginTopLeft
	case "top-center":
		return OriginTopCenter
	case "top-right":
		return OriginTopRight
	case "left-center":
		return OriginLeftCenter
	case "center":
		return OriginCenter
	case "center-sized":
		return OriginCenterSized
	case "right-center":
		return OriginRightCenter
	case "bottom-left":
		return OriginBottomLeft
	case "bottom-center":
		return OriginBottomCenter
	case "bottom-right":
		return OriginBottomRight
	default:
		return OriginCenter
	}
}

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

type Dimension struct {
	Value    int
	IsPixels bool
}

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

func (d Dimension) String() string {
	if d.IsPixels {
		return fmt.Sprintf("%dpx", d.Value)
	}
	return strconv.Itoa(d.Value)
}

type Position struct {
	X int
	Y int
}

func ParsePosition(s string) (Position, error) {
	if s == "" {
		return Position{}, nil
	}

	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return Position{}, fmt.Errorf("invalid position format: %s (expected x,y)", s)
	}

	x, err := strconv.Atoi(strings.TrimSpace(parts[0]))
	if err != nil {
		return Position{}, fmt.Errorf("invalid x position: %w", err)
	}

	y, err := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err != nil {
		return Position{}, fmt.Errorf("invalid y position: %w", err)
	}

	return Position{X: x, Y: y}, nil
}

func (p Position) String() string {
	return fmt.Sprintf("%d,%d", p.X, p.Y)
}

type Config struct {
	Type        LayerType
	Origin      Origin
	FocusPolicy FocusPolicy

	Width    Dimension // Width in columns or pixels (e.g., 80 or "1200px")
	Height   Dimension // Height in lines or pixels (e.g., 24 or "600px")
	Position Position  // Offset as "x,y" (e.g., "10,50")

	ExclusiveZone         int
	OverrideExclusiveZone bool

	HideOnFocusLoss  bool
	ToggleVisibility bool

	OutputName   string // Monitor name (e.g., "DP-2") - CRITICAL: Must be DP-2, never DP-1
	ListenSocket string // Unix socket path
	WindowTitle  string // Window title for targeting specific windows
}

func NewConfig() *Config {
	return &Config{
		Type:          LayerShellPanel,
		Origin:        OriginTopLeft,
		FocusPolicy:   FocusNotAllowed,
		Width:         Dimension{Value: 1, IsPixels: false},
		Height:        Dimension{Value: 1, IsPixels: false},
		Position:      Position{X: 0, Y: 0},
		ExclusiveZone: -1, // Auto
		OutputName:    "DP-2", // CRITICAL: Default to DP-2
	}
}

func getMonitorResolution(monitorName string) (width, height int, err error) {
	if monitorName == "" {
		monitorName = "DP-2"
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

func (c *Config) originToEdge() string {
	switch c.Origin {
	case OriginTopLeft, OriginTopCenter, OriginTopRight:
		return "top"
	case OriginBottomLeft, OriginBottomCenter, OriginBottomRight:
		return "bottom"
	case OriginLeftCenter:
		return "left"
	case OriginRightCenter:
		return "right"
	case OriginCenter:
		return "center"
	case OriginCenterSized:
		return "center-sized"
	default:
		return "center"
	}
}

func (c *Config) calculateMargins() (top, left, bottom, right int, err error) {
	if c.Origin == OriginCenterSized {
		return 0, 0, 0, 0, nil
	}

	monWidth, monHeight, err := getMonitorResolution(c.OutputName)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("failed to get monitor resolution: %w", err)
	}

	panelWidth := c.Width.Value
	if !c.Width.IsPixels {
		panelWidth = c.Width.Value * 10
	}

	panelHeight := c.Height.Value
	if !c.Height.IsPixels {
		panelHeight = c.Height.Value * 20
	}

	offsetX := c.Position.X
	offsetY := c.Position.Y

	switch c.Origin {
	case OriginTopLeft:
		left = offsetX
		top = offsetY

	case OriginTopCenter:
		left = (monWidth / 2) - (panelWidth / 2) + offsetX
		top = offsetY

	case OriginTopRight:
		right = offsetX
		top = offsetY

	case OriginLeftCenter:
		left = offsetX
		top = (monHeight / 2) - (panelHeight / 2) + offsetY

	case OriginCenter:
		// edge=center anchors to ALL sides, so we MUST set all four margins
		// to properly center and size the panel. Kitty will shrink the panel and place it using these margin constraints.
		left = (monWidth / 2) - (panelWidth / 2) + offsetX
		top = (monHeight / 2) - (panelHeight / 2) + offsetY
		right = (monWidth / 2) - (panelWidth / 2) - offsetX
		bottom = (monHeight / 2) - (panelHeight / 2) - offsetY

	case OriginRightCenter:
		right = offsetX
		top = (monHeight / 2) - (panelHeight / 2) + offsetY

	case OriginBottomLeft:
		left = offsetX
		bottom = offsetY

	case OriginBottomCenter:
		left = (monWidth / 2) - (panelWidth / 2) + offsetX
		bottom = offsetY

	case OriginBottomRight:
		right = offsetX
		bottom = offsetY

	default:
		left = (monWidth / 2) - (panelWidth / 2) + offsetX
		top = (monHeight / 2) - (panelHeight / 2) + offsetY
		right = (monWidth / 2) - (panelWidth / 2) - offsetX
		bottom = (monHeight / 2) - (panelHeight / 2) - offsetY
	}

	return top, left, bottom, right, nil
}

func (c *Config) ToPanelArgs(componentPath string) []string {
	args := []string{
		"@",
		"launch",
		"--type=os-panel",
	}

	panelProps := []string{}

	edgeStr := c.originToEdge()
	if edgeStr != "" {
		panelProps = append(panelProps, fmt.Sprintf("edge=%s", edgeStr))
	}

	if c.Width.Value > 0 {
		panelProps = append(panelProps, fmt.Sprintf("columns=%s", c.Width.String()))
	}
	if c.Height.Value > 0 {
		panelProps = append(panelProps, fmt.Sprintf("lines=%s", c.Height.String()))
	}

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

	if c.FocusPolicy != FocusNotAllowed {
		panelProps = append(panelProps, fmt.Sprintf("focus-policy=%s", c.FocusPolicy.String()))
	}

	if c.OutputName != "" {
		panelProps = append(panelProps, fmt.Sprintf("output-name=%s", c.OutputName))
	}

	for _, prop := range panelProps {
		args = append(args, "--os-panel", prop)
	}

	if c.WindowTitle != "" {
		args = append(args, "--title", c.WindowTitle)
	}

	args = append(args, componentPath)

	return args
}

