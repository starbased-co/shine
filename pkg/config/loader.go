package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "shine", "shine.toml")
}

// Load loads configuration from the given file path
func Load(path string) (*Config, error) {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", path, err)
	}

	// Parse TOML
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %s: %w", path, err)
	}

	return &cfg, nil
}

// LoadOrDefault attempts to load config from path, returns default if not found
func LoadOrDefault(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		return NewDefaultConfig()
	}

	// Ensure core config has defaults if not specified
	if cfg.Core == nil {
		cfg.Core = &CoreConfig{
			Path: []string{
				"~/.local/share/shine/bin",
				"~/.config/shine/bin",
				"/usr/lib/shine/bin",
			},
		}
	} else {
		// Set defaults if not provided
		if cfg.Core.Path == nil {
			cfg.Core.Path = []string{
				"~/.local/share/shine/bin",
				"~/.config/shine/bin",
				"/usr/lib/shine/bin",
			}
		}
	}

	// Initialize prisms map if nil
	if cfg.Prisms == nil {
		cfg.Prisms = make(map[string]*PrismConfig)
	}

	// Backward compatibility: Migrate old config sections to prisms
	migrated := false

	if cfg.Bar != nil {
		if _, exists := cfg.Prisms["bar"]; !exists {
			cfg.Prisms["bar"] = &PrismConfig{
				Name:            "bar",
				Enabled:         cfg.Bar.Enabled,
				Edge:            cfg.Bar.Edge,
				Lines:           cfg.Bar.Lines,
				Columns:         cfg.Bar.Columns,
				LinesPixels:     cfg.Bar.LinesPixels,
				ColumnsPixels:   cfg.Bar.ColumnsPixels,
				MarginTop:       cfg.Bar.MarginTop,
				MarginLeft:      cfg.Bar.MarginLeft,
				MarginBottom:    cfg.Bar.MarginBottom,
				MarginRight:     cfg.Bar.MarginRight,
				HideOnFocusLoss: cfg.Bar.HideOnFocusLoss,
				FocusPolicy:     cfg.Bar.FocusPolicy,
				OutputName:      cfg.Bar.OutputName,
			}
			migrated = true
		}
	}

	if cfg.Chat != nil {
		if _, exists := cfg.Prisms["chat"]; !exists {
			cfg.Prisms["chat"] = &PrismConfig{
				Name:            "chat",
				Enabled:         cfg.Chat.Enabled,
				Edge:            cfg.Chat.Edge,
				Lines:           cfg.Chat.Lines,
				Columns:         cfg.Chat.Columns,
				LinesPixels:     cfg.Chat.LinesPixels,
				ColumnsPixels:   cfg.Chat.ColumnsPixels,
				MarginTop:       cfg.Chat.MarginTop,
				MarginLeft:      cfg.Chat.MarginLeft,
				MarginBottom:    cfg.Chat.MarginBottom,
				MarginRight:     cfg.Chat.MarginRight,
				HideOnFocusLoss: cfg.Chat.HideOnFocusLoss,
				FocusPolicy:     cfg.Chat.FocusPolicy,
				OutputName:      cfg.Chat.OutputName,
			}
			migrated = true
		}
	}

	if cfg.Clock != nil {
		if _, exists := cfg.Prisms["clock"]; !exists {
			cfg.Prisms["clock"] = &PrismConfig{
				Name:            "clock",
				Enabled:         cfg.Clock.Enabled,
				Edge:            cfg.Clock.Edge,
				Lines:           cfg.Clock.Lines,
				Columns:         cfg.Clock.Columns,
				LinesPixels:     cfg.Clock.LinesPixels,
				ColumnsPixels:   cfg.Clock.ColumnsPixels,
				MarginTop:       cfg.Clock.MarginTop,
				MarginLeft:      cfg.Clock.MarginLeft,
				MarginBottom:    cfg.Clock.MarginBottom,
				MarginRight:     cfg.Clock.MarginRight,
				HideOnFocusLoss: cfg.Clock.HideOnFocusLoss,
				FocusPolicy:     cfg.Clock.FocusPolicy,
				OutputName:      cfg.Clock.OutputName,
			}
			migrated = true
		}
	}

	if cfg.SysInfo != nil {
		if _, exists := cfg.Prisms["sysinfo"]; !exists {
			cfg.Prisms["sysinfo"] = &PrismConfig{
				Name:            "sysinfo",
				Enabled:         cfg.SysInfo.Enabled,
				Edge:            cfg.SysInfo.Edge,
				Lines:           cfg.SysInfo.Lines,
				Columns:         cfg.SysInfo.Columns,
				LinesPixels:     cfg.SysInfo.LinesPixels,
				ColumnsPixels:   cfg.SysInfo.ColumnsPixels,
				MarginTop:       cfg.SysInfo.MarginTop,
				MarginLeft:      cfg.SysInfo.MarginLeft,
				MarginBottom:    cfg.SysInfo.MarginBottom,
				MarginRight:     cfg.SysInfo.MarginRight,
				HideOnFocusLoss: cfg.SysInfo.HideOnFocusLoss,
				FocusPolicy:     cfg.SysInfo.FocusPolicy,
				OutputName:      cfg.SysInfo.OutputName,
			}
			migrated = true
		}
	}

	if migrated {
		fmt.Fprintf(os.Stderr, "\n⚠️  Warning: Detected deprecated config format ([bar], [chat], etc.)\n")
		fmt.Fprintf(os.Stderr, "   Consider migrating to new [prisms.*] format.\n")
		fmt.Fprintf(os.Stderr, "   See: https://github.com/starbased-co/shine/blob/main/docs/PRISM_SYSTEM_DESIGN.md\n\n")
	}

	return cfg
}

// Save saves configuration to the given file path
func Save(cfg *Config, path string) error {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer f.Close()

	// Encode TOML
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config: %w", err)
	}

	return nil
}
