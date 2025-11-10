package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config represents the shinectl configuration
type Config struct {
	Prisms []PrismEntry `toml:"prism"`
}

// PrismEntry represents a single prism configuration entry
type PrismEntry struct {
	Name         string `toml:"name"`
	Restart      string `toml:"restart"`       // always | on-failure | unless-stopped | no
	RestartDelay string `toml:"restart_delay"` // Duration string (e.g., "5s")
	MaxRestarts  int    `toml:"max_restarts"`  // Max restarts per hour (0 = unlimited)
}

// RestartPolicy represents the restart behavior
type RestartPolicy int

const (
	RestartNo RestartPolicy = iota
	RestartOnFailure
	RestartUnlessStopped
	RestartAlways
)

// GetRestartPolicy converts string to RestartPolicy enum
func (pe *PrismEntry) GetRestartPolicy() RestartPolicy {
	switch pe.Restart {
	case "always":
		return RestartAlways
	case "on-failure":
		return RestartOnFailure
	case "unless-stopped":
		return RestartUnlessStopped
	case "no", "":
		return RestartNo
	default:
		return RestartNo
	}
}

// GetRestartDelay parses the restart_delay string into a Duration
func (pe *PrismEntry) GetRestartDelay() time.Duration {
	if pe.RestartDelay == "" {
		return 1 * time.Second // Default
	}
	d, err := time.ParseDuration(pe.RestartDelay)
	if err != nil {
		return 1 * time.Second // Fallback
	}
	return d
}

// DefaultConfigPath returns the default prism.toml location
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "shine", "prism.toml")
}

// LoadConfig loads prism.toml from the given path
func LoadConfig(path string) (*Config, error) {
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		path = filepath.Join(home, path[1:])
	}

	var cfg Config
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// LoadConfigOrDefault loads config from path, returns empty config if not found
func LoadConfigOrDefault(path string) *Config {
	cfg, err := LoadConfig(path)
	if err != nil {
		// Return empty config - no prisms to spawn
		return &Config{Prisms: []PrismEntry{}}
	}
	return cfg
}

// Validate checks if the config is valid
func (c *Config) Validate() error {
	seen := make(map[string]bool)
	for i, prism := range c.Prisms {
		if prism.Name == "" {
			return fmt.Errorf("prism[%d]: name is required", i)
		}
		if seen[prism.Name] {
			return fmt.Errorf("prism[%d]: duplicate name %q", i, prism.Name)
		}
		seen[prism.Name] = true

		// Validate restart policy
		switch prism.Restart {
		case "", "no", "on-failure", "unless-stopped", "always":
			// Valid
		default:
			return fmt.Errorf("prism[%d] %q: invalid restart policy %q", i, prism.Name, prism.Restart)
		}

		// Validate restart_delay if present
		if prism.RestartDelay != "" {
			if _, err := time.ParseDuration(prism.RestartDelay); err != nil {
				return fmt.Errorf("prism[%d] %q: invalid restart_delay %q: %w", i, prism.Name, prism.RestartDelay, err)
			}
		}

		// Validate max_restarts
		if prism.MaxRestarts < 0 {
			return fmt.Errorf("prism[%d] %q: max_restarts must be >= 0", i, prism.Name)
		}
	}
	return nil
}
