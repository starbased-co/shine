package config

import "github.com/starbased-co/shine/pkg/panel"

// Config represents the main shine configuration
type Config struct {
	Core   *CoreConfig               `toml:"core"`
	Prisms map[string]*PrismConfig   `toml:"prisms"`

	// Deprecated: Use Prisms["chat"] instead
	Chat    *ChatConfig    `toml:"chat"`
	// Deprecated: Use Prisms["bar"] instead
	Bar     *BarConfig     `toml:"bar"`
	// Deprecated: Use Prisms["clock"] instead
	Clock   *ClockConfig   `toml:"clock"`
	// Deprecated: Use Prisms["sysinfo"] instead
	SysInfo *SysInfoConfig `toml:"sysinfo"`
}

// CoreConfig holds global shine settings
type CoreConfig struct {
	// Path specifies directories to prepend to PATH for prism binary discovery
	// Can be a single string or array of strings
	// Example: "~/.local/share/shine/bin" or ["~/.local/share/shine/bin", "~/.config/shine/bin"]
	Path interface{} `toml:"path"`
}

// GetPaths normalizes the Path field to []string
// Handles both string and []string types
func (cc *CoreConfig) GetPaths() []string {
	if cc.Path == nil {
		return []string{}
	}

	switch v := cc.Path.(type) {
	case string:
		return []string{v}
	case []interface{}:
		paths := make([]string, 0, len(v))
		for _, item := range v {
			if str, ok := item.(string); ok {
				paths = append(paths, str)
			}
		}
		return paths
	case []string:
		return v
	default:
		return []string{}
	}
}

// PrismConfig is the unified configuration for ALL prisms (built-in and user)
type PrismConfig struct {
	// Name is the prism identifier
	Name string `toml:"name"`

	// Binary specifies a custom binary name or path (optional)
	// If empty, defaults to "shine-{name}"
	// Can be a simple name (e.g., "shine-weather") or a path (e.g., "/usr/bin/shine-weather")
	Binary string `toml:"binary"`

	// Enabled controls whether this prism should be launched
	Enabled bool `toml:"enabled"`

	// Panel configuration (new format)
	Anchor          string      `toml:"anchor"`
	Width           interface{} `toml:"width"`   // int or string (with "px")
	Height          interface{} `toml:"height"`  // int or string (with "px")
	Position        string      `toml:"position"` // "x,y" format
	MarginTop       int         `toml:"margin_top"`
	MarginLeft      int         `toml:"margin_left"`
	MarginBottom    int         `toml:"margin_bottom"`
	MarginRight     int         `toml:"margin_right"`
	HideOnFocusLoss bool        `toml:"hide_on_focus_loss"`
	FocusPolicy     string      `toml:"focus_policy"`
	OutputName      string      `toml:"output_name"`

	// Deprecated fields (for backward compatibility)
	Edge          string `toml:"edge,omitempty"`
	Lines         int    `toml:"lines,omitempty"`
	Columns       int    `toml:"columns,omitempty"`
	LinesPixels   int    `toml:"lines_pixels,omitempty"`
	ColumnsPixels int    `toml:"columns_pixels,omitempty"`
}

// ToPanelConfig converts PrismConfig to panel.Config
func (pc *PrismConfig) ToPanelConfig() *panel.Config {
	cfg := panel.NewConfig()

	// Anchor placement (with backward compatibility for "edge")
	anchor := pc.Anchor
	if anchor == "" && pc.Edge != "" {
		// Legacy support: map edge to anchor
		anchor = pc.Edge
	}
	cfg.Anchor = panel.ParseAnchor(anchor)

	// Handle background anchor special case
	if cfg.Anchor == panel.AnchorBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size (new format with backward compatibility)
	var err error
	if pc.Width != nil {
		cfg.Width, err = panel.ParseDimension(pc.Width)
		if err != nil {
			// Fallback to deprecated fields
			if pc.ColumnsPixels > 0 {
				cfg.Width = panel.Dimension{Value: pc.ColumnsPixels, IsPixels: true}
			} else if pc.Columns > 0 {
				cfg.Width = panel.Dimension{Value: pc.Columns, IsPixels: false}
			}
		}
	} else {
		// Backward compatibility: use old fields
		if pc.ColumnsPixels > 0 {
			cfg.Width = panel.Dimension{Value: pc.ColumnsPixels, IsPixels: true}
		} else if pc.Columns > 0 {
			cfg.Width = panel.Dimension{Value: pc.Columns, IsPixels: false}
		}
	}

	if pc.Height != nil {
		cfg.Height, err = panel.ParseDimension(pc.Height)
		if err != nil {
			// Fallback to deprecated fields
			if pc.LinesPixels > 0 {
				cfg.Height = panel.Dimension{Value: pc.LinesPixels, IsPixels: true}
			} else if pc.Lines > 0 {
				cfg.Height = panel.Dimension{Value: pc.Lines, IsPixels: false}
			}
		}
	} else {
		// Backward compatibility: use old fields
		if pc.LinesPixels > 0 {
			cfg.Height = panel.Dimension{Value: pc.LinesPixels, IsPixels: true}
		} else if pc.Lines > 0 {
			cfg.Height = panel.Dimension{Value: pc.Lines, IsPixels: false}
		}
	}

	// Position
	if pc.Position != "" {
		cfg.Position, err = panel.ParsePosition(pc.Position)
		// Silently ignore position parse errors, keep default
	}

	// Margins
	cfg.MarginTop = pc.MarginTop
	cfg.MarginLeft = pc.MarginLeft
	cfg.MarginBottom = pc.MarginBottom
	cfg.MarginRight = pc.MarginRight

	// Behavior
	cfg.HideOnFocusLoss = pc.HideOnFocusLoss

	// Focus policy
	cfg.FocusPolicy = panel.ParseFocusPolicy(pc.FocusPolicy)

	// If hide_on_focus_loss is enabled, ensure focus policy is on-demand
	if cfg.HideOnFocusLoss {
		cfg.FocusPolicy = panel.FocusOnDemand
	}

	// Output
	cfg.OutputName = pc.OutputName

	// Remote control socket (shared across all components)
	cfg.ListenSocket = "/tmp/shine.sock"

	return cfg
}

// ChatConfig represents chat component configuration
type ChatConfig struct {
	Enabled         bool   `toml:"enabled"`
	Edge            string `toml:"edge"`
	Lines           int    `toml:"lines"`
	Columns         int    `toml:"columns"`
	LinesPixels     int    `toml:"lines_pixels"`
	ColumnsPixels   int    `toml:"columns_pixels"`
	MarginTop       int    `toml:"margin_top"`
	MarginLeft      int    `toml:"margin_left"`
	MarginBottom    int    `toml:"margin_bottom"`
	MarginRight     int    `toml:"margin_right"`
	HideOnFocusLoss bool   `toml:"hide_on_focus_loss"`
	FocusPolicy     string `toml:"focus_policy"`
	OutputName      string `toml:"output_name"`
}

// ToPanelConfig converts ChatConfig to panel.Config (deprecated)
func (cc *ChatConfig) ToPanelConfig() *panel.Config {
	cfg := panel.NewConfig()

	// Anchor placement
	cfg.Anchor = panel.ParseAnchor(cc.Edge)

	// Handle background anchor special case
	if cfg.Anchor == panel.AnchorBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size (backward compatibility with old field names)
	if cc.ColumnsPixels > 0 {
		cfg.Width = panel.Dimension{Value: cc.ColumnsPixels, IsPixels: true}
	} else if cc.Columns > 0 {
		cfg.Width = panel.Dimension{Value: cc.Columns, IsPixels: false}
	}

	if cc.LinesPixels > 0 {
		cfg.Height = panel.Dimension{Value: cc.LinesPixels, IsPixels: true}
	} else if cc.Lines > 0 {
		cfg.Height = panel.Dimension{Value: cc.Lines, IsPixels: false}
	}

	// Margins
	cfg.MarginTop = cc.MarginTop
	cfg.MarginLeft = cc.MarginLeft
	cfg.MarginBottom = cc.MarginBottom
	cfg.MarginRight = cc.MarginRight

	// Behavior
	cfg.HideOnFocusLoss = cc.HideOnFocusLoss

	// Focus policy
	cfg.FocusPolicy = panel.ParseFocusPolicy(cc.FocusPolicy)

	// If hide_on_focus_loss is enabled, ensure focus policy is on-demand
	if cfg.HideOnFocusLoss {
		cfg.FocusPolicy = panel.FocusOnDemand
	}

	// Output
	cfg.OutputName = cc.OutputName

	// Remote control socket (shared across all components)
	cfg.ListenSocket = "/tmp/shine.sock"

	return cfg
}

// BarConfig represents status bar component configuration
type BarConfig struct {
	Enabled         bool   `toml:"enabled"`
	Edge            string `toml:"edge"`
	Lines           int    `toml:"lines"`
	Columns         int    `toml:"columns"`
	LinesPixels     int    `toml:"lines_pixels"`
	ColumnsPixels   int    `toml:"columns_pixels"`
	MarginTop       int    `toml:"margin_top"`
	MarginLeft      int    `toml:"margin_left"`
	MarginBottom    int    `toml:"margin_bottom"`
	MarginRight     int    `toml:"margin_right"`
	HideOnFocusLoss bool   `toml:"hide_on_focus_loss"`
	FocusPolicy     string `toml:"focus_policy"`
	OutputName      string `toml:"output_name"`
}

// ToPanelConfig converts BarConfig to panel.Config
func (bc *BarConfig) ToPanelConfig() *panel.Config {
	cfg := panel.NewConfig()

	// Edge placement
	cfg.Anchor = panel.ParseAnchor(bc.Edge)

	// Handle background edge special case
	if cfg.Anchor == panel.AnchorBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size (backward compatibility)
	if bc.ColumnsPixels > 0 {
		cfg.Width = panel.Dimension{Value: bc.ColumnsPixels, IsPixels: true}
	} else if bc.Columns > 0 {
		cfg.Width = panel.Dimension{Value: bc.Columns, IsPixels: false}
	}
	if bc.LinesPixels > 0 {
		cfg.Height = panel.Dimension{Value: bc.LinesPixels, IsPixels: true}
	} else if bc.Lines > 0 {
		cfg.Height = panel.Dimension{Value: bc.Lines, IsPixels: false}
	}

	// Margins
	cfg.MarginTop = bc.MarginTop
	cfg.MarginLeft = bc.MarginLeft
	cfg.MarginBottom = bc.MarginBottom
	cfg.MarginRight = bc.MarginRight

	// Behavior
	cfg.HideOnFocusLoss = bc.HideOnFocusLoss

	// Focus policy
	cfg.FocusPolicy = panel.ParseFocusPolicy(bc.FocusPolicy)

	// If hide_on_focus_loss is enabled, ensure focus policy is on-demand
	if cfg.HideOnFocusLoss {
		cfg.FocusPolicy = panel.FocusOnDemand
	}

	// Output
	cfg.OutputName = bc.OutputName

	// Remote control socket (shared across all components)
	cfg.ListenSocket = "/tmp/shine.sock"

	return cfg
}

// ClockConfig represents clock component configuration
type ClockConfig struct {
	Enabled         bool   `toml:"enabled"`
	Edge            string `toml:"edge"`
	Lines           int    `toml:"lines"`
	Columns         int    `toml:"columns"`
	LinesPixels     int    `toml:"lines_pixels"`
	ColumnsPixels   int    `toml:"columns_pixels"`
	MarginTop       int    `toml:"margin_top"`
	MarginLeft      int    `toml:"margin_left"`
	MarginBottom    int    `toml:"margin_bottom"`
	MarginRight     int    `toml:"margin_right"`
	HideOnFocusLoss bool   `toml:"hide_on_focus_loss"`
	FocusPolicy     string `toml:"focus_policy"`
	OutputName      string `toml:"output_name"`
}

// ToPanelConfig converts ClockConfig to panel.Config
func (clc *ClockConfig) ToPanelConfig() *panel.Config {
	cfg := panel.NewConfig()

	// Edge placement
	cfg.Anchor = panel.ParseAnchor(clc.Edge)

	// Handle background edge special case
	if cfg.Anchor == panel.AnchorBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size (backward compatibility)
	if clc.ColumnsPixels > 0 {
		cfg.Width = panel.Dimension{Value: clc.ColumnsPixels, IsPixels: true}
	} else if clc.Columns > 0 {
		cfg.Width = panel.Dimension{Value: clc.Columns, IsPixels: false}
	}
	if clc.LinesPixels > 0 {
		cfg.Height = panel.Dimension{Value: clc.LinesPixels, IsPixels: true}
	} else if clc.Lines > 0 {
		cfg.Height = panel.Dimension{Value: clc.Lines, IsPixels: false}
	}

	// Margins
	cfg.MarginTop = clc.MarginTop
	cfg.MarginLeft = clc.MarginLeft
	cfg.MarginBottom = clc.MarginBottom
	cfg.MarginRight = clc.MarginRight

	// Behavior
	cfg.HideOnFocusLoss = clc.HideOnFocusLoss

	// Focus policy
	cfg.FocusPolicy = panel.ParseFocusPolicy(clc.FocusPolicy)

	// If hide_on_focus_loss is enabled, ensure focus policy is on-demand
	if cfg.HideOnFocusLoss {
		cfg.FocusPolicy = panel.FocusOnDemand
	}

	// Output
	cfg.OutputName = clc.OutputName

	// Remote control socket (shared across all components)
	cfg.ListenSocket = "/tmp/shine.sock"

	return cfg
}

// SysInfoConfig represents system info component configuration
type SysInfoConfig struct {
	Enabled         bool   `toml:"enabled"`
	Edge            string `toml:"edge"`
	Lines           int    `toml:"lines"`
	Columns         int    `toml:"columns"`
	LinesPixels     int    `toml:"lines_pixels"`
	ColumnsPixels   int    `toml:"columns_pixels"`
	MarginTop       int    `toml:"margin_top"`
	MarginLeft      int    `toml:"margin_left"`
	MarginBottom    int    `toml:"margin_bottom"`
	MarginRight     int    `toml:"margin_right"`
	HideOnFocusLoss bool   `toml:"hide_on_focus_loss"`
	FocusPolicy     string `toml:"focus_policy"`
	OutputName      string `toml:"output_name"`
}

// ToPanelConfig converts SysInfoConfig to panel.Config
func (sic *SysInfoConfig) ToPanelConfig() *panel.Config {
	cfg := panel.NewConfig()

	// Edge placement
	cfg.Anchor = panel.ParseAnchor(sic.Edge)

	// Handle background edge special case
	if cfg.Anchor == panel.AnchorBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size (backward compatibility)
	if sic.ColumnsPixels > 0 {
		cfg.Width = panel.Dimension{Value: sic.ColumnsPixels, IsPixels: true}
	} else if sic.Columns > 0 {
		cfg.Width = panel.Dimension{Value: sic.Columns, IsPixels: false}
	}
	if sic.LinesPixels > 0 {
		cfg.Height = panel.Dimension{Value: sic.LinesPixels, IsPixels: true}
	} else if sic.Lines > 0 {
		cfg.Height = panel.Dimension{Value: sic.Lines, IsPixels: false}
	}

	// Margins
	cfg.MarginTop = sic.MarginTop
	cfg.MarginLeft = sic.MarginLeft
	cfg.MarginBottom = sic.MarginBottom
	cfg.MarginRight = sic.MarginRight

	// Behavior
	cfg.HideOnFocusLoss = sic.HideOnFocusLoss

	// Focus policy
	cfg.FocusPolicy = panel.ParseFocusPolicy(sic.FocusPolicy)

	// If hide_on_focus_loss is enabled, ensure focus policy is on-demand
	if cfg.HideOnFocusLoss {
		cfg.FocusPolicy = panel.FocusOnDemand
	}

	// Output
	cfg.OutputName = sic.OutputName

	// Remote control socket (shared across all components)
	cfg.ListenSocket = "/tmp/shine.sock"

	return cfg
}

// NewDefaultConfig creates a default configuration
func NewDefaultConfig() *Config {
	return &Config{
		Core: &CoreConfig{
			Path: []string{
				"~/.local/share/shine/bin",
				"~/.config/shine/bin",
				"/usr/lib/shine/bin",
			},
		},
		Prisms: map[string]*PrismConfig{
			"bar": {
				Name:        "bar",
				Enabled:     true,
				Edge:        "top",
				LinesPixels: 30,
				FocusPolicy: "not-allowed",
			},
		},
		// Backward compatibility: old config format
		Chat: &ChatConfig{
			Enabled:         false,
			Edge:            "bottom",
			Lines:           10,
			MarginLeft:      10,
			MarginRight:     10,
			MarginBottom:    10,
			HideOnFocusLoss: true,
			FocusPolicy:     "on-demand",
		},
	}
}
