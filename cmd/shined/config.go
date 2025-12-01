package main

import (
	"time"

	"github.com/starbased-co/shine/pkg/config"
)

// PrismEntry wraps config.PrismConfig with restart policies
type PrismEntry struct {
	*config.PrismConfig

	Restart      string `toml:"restart"`       // always | on-failure | unless-stopped | no
	RestartDelay string `toml:"restart_delay"` // Duration string (e.g., "5s")
	MaxRestarts  int    `toml:"max_restarts"`  // Max restarts per hour (0 = unlimited)
}

type RestartPolicy int

const (
	RestartNo RestartPolicy = iota
	RestartOnFailure
	RestartUnlessStopped
	RestartAlways
)

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

func (pe *PrismEntry) ValidateRestartPolicy() error {
	if err := config.ValidateRestartPolicy(pe.Restart); err != nil {
		return err
	}
	return config.ValidateRestartDelay(pe.RestartDelay)
}

func (pe *PrismEntry) GetApps() map[string]*config.AppConfig {
	return pe.PrismConfig.GetApps()
}
