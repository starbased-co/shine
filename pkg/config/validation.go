package config

import (
	"fmt"
	"time"

	"github.com/starbased-co/shine/pkg/panel"
)

// Validate checks if the config is valid
func (c *Config) Validate() error {
	seen := make(map[string]bool)
	for name, prism := range c.Prisms {
		if prism.Name == "" {
			return fmt.Errorf("prism %q: name is required", name)
		}
		if seen[prism.Name] {
			return fmt.Errorf("prism %q: duplicate name", prism.Name)
		}
		seen[prism.Name] = true

		if err := prism.Validate(); err != nil {
			return fmt.Errorf("prism %q: %w", name, err)
		}
	}
	return nil
}

// Validate checks if the PrismConfig is valid
func (pc *PrismConfig) Validate() error {
	// Validate positioning fields using pkg/panel parsers
	if pc.Origin != "" {
		// Validate origin is recognized
		_ = panel.ParseOrigin(pc.Origin)
	}

	if pc.Position != "" {
		if _, err := panel.ParsePosition(pc.Position); err != nil {
			return fmt.Errorf("invalid position %q: %w", pc.Position, err)
		}
	}

	if pc.Width != nil {
		if _, err := panel.ParseDimension(pc.Width); err != nil {
			return fmt.Errorf("invalid width %v: %w", pc.Width, err)
		}
	}

	if pc.Height != nil {
		if _, err := panel.ParseDimension(pc.Height); err != nil {
			return fmt.Errorf("invalid height %v: %w", pc.Height, err)
		}
	}

	if pc.FocusPolicy != "" {
		// Validate focus policy is recognized
		_ = panel.ParseFocusPolicy(pc.FocusPolicy)
	}

	return nil
}

// ValidateRestartPolicy validates restart policy strings
func ValidateRestartPolicy(policy string) error {
	switch policy {
	case "", "no", "on-failure", "unless-stopped", "always":
		return nil
	default:
		return fmt.Errorf("invalid restart policy %q", policy)
	}
}

// ValidateRestartDelay validates restart delay duration strings
func ValidateRestartDelay(delay string) error {
	if delay == "" {
		return nil
	}
	if _, err := time.ParseDuration(delay); err != nil {
		return fmt.Errorf("invalid restart_delay %q: %w", delay, err)
	}
	return nil
}
