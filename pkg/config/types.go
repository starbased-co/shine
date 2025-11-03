package config

import "github.com/starbased-co/shine/pkg/panel"

// Config represents the main shine configuration
type Config struct {
	Chat    *ChatConfig    `toml:"chat"`
	Bar     *BarConfig     `toml:"bar"`
	Clock   *ClockConfig   `toml:"clock"`
	SysInfo *SysInfoConfig `toml:"sysinfo"`
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

// ToPanelConfig converts ChatConfig to panel.Config
func (cc *ChatConfig) ToPanelConfig() *panel.Config {
	cfg := panel.NewConfig()

	// Edge placement
	cfg.Edge = panel.ParseEdge(cc.Edge)

	// Handle background edge special case
	if cfg.Edge == panel.EdgeBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size
	cfg.Lines = cc.Lines
	cfg.Columns = cc.Columns
	cfg.LinesPixels = cc.LinesPixels
	cfg.ColumnsPixels = cc.ColumnsPixels

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
	cfg.Edge = panel.ParseEdge(bc.Edge)

	// Handle background edge special case
	if cfg.Edge == panel.EdgeBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size
	cfg.Lines = bc.Lines
	cfg.Columns = bc.Columns
	cfg.LinesPixels = bc.LinesPixels
	cfg.ColumnsPixels = bc.ColumnsPixels

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
	cfg.Edge = panel.ParseEdge(clc.Edge)

	// Handle background edge special case
	if cfg.Edge == panel.EdgeBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size
	cfg.Lines = clc.Lines
	cfg.Columns = clc.Columns
	cfg.LinesPixels = clc.LinesPixels
	cfg.ColumnsPixels = clc.ColumnsPixels

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
	cfg.Edge = panel.ParseEdge(sic.Edge)

	// Handle background edge special case
	if cfg.Edge == panel.EdgeBackground {
		cfg.Type = panel.LayerShellBackground
	}

	// Size
	cfg.Lines = sic.Lines
	cfg.Columns = sic.Columns
	cfg.LinesPixels = sic.LinesPixels
	cfg.ColumnsPixels = sic.ColumnsPixels

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
		Chat: &ChatConfig{
			Enabled:         true,
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
