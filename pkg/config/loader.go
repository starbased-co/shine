package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/starbased-co/shine/pkg/paths"
)

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	return paths.DefaultConfigPath()
}

// Load loads configuration from the given file path and discovers prisms
func Load(path string) (*Config, error) {
	// Load main shine.toml (metadata will be stripped automatically)
	cfg, err := loadShineConfig(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config file %s: %w", path, err)
	}

	// Initialize prisms map if nil
	if cfg.Prisms == nil {
		cfg.Prisms = make(map[string]*PrismConfig)
	}

	// Discover prisms from configured directories
	if cfg.Core != nil && cfg.Core.Path != nil {
		prismDirs := cfg.Core.GetPaths()
		discovered, err := DiscoverPrisms(prismDirs)
		if err == nil {
			// Merge discovered prisms with shine.toml config
			for name, discoveredPrism := range discovered {
				if userConfig, exists := cfg.Prisms[name]; exists {
					// User has config in shine.toml - merge with discovered config
					cfg.Prisms[name] = MergePrismConfigs(discoveredPrism.Config, userConfig)
				} else {
					// No user config - use discovered config as-is
					cfg.Prisms[name] = discoveredPrism.Config
				}
			}
		}
	}

	return cfg, nil
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
				"~/.config/shine/prisms",
				"/usr/lib/shine/bin",
			},
		}
	} else if cfg.Core.Path == nil {
		cfg.Core.Path = []string{
			"~/.local/share/shine/bin",
			"~/.config/shine/bin",
			"~/.config/shine/prisms",
			"/usr/lib/shine/bin",
		}
	}

	// Initialize prisms map if nil
	if cfg.Prisms == nil {
		cfg.Prisms = make(map[string]*PrismConfig)
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
