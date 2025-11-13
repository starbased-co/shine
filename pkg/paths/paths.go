package paths

import (
	"fmt"
	"os"
	"path/filepath"
)

// ExpandHome expands ~ to the user's home directory
func ExpandHome(path string) string {
	if len(path) == 0 || path[0] != '~' {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if len(path) == 1 {
		return home
	}
	return filepath.Join(home, path[1:])
}

// ConfigDir returns the shine configuration directory
func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "shine")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "shine")
}

// DataDir returns the shine data directory
func DataDir() string {
	if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
		return filepath.Join(xdg, "shine")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "shine")
}

// LogDir returns the shine log directory
func LogDir() string {
	return filepath.Join(DataDir(), "logs")
}

// RuntimeDir returns the shine runtime directory
func RuntimeDir() string {
	uid := os.Getuid()
	return filepath.Join("/run/user", fmt.Sprintf("%d", uid), "shine")
}

// ShinectlSocket returns the path to the shinectl control socket
func ShinectlSocket() string {
	return filepath.Join(RuntimeDir(), "shinectl.sock")
}

// PrismSocket returns the path to a prism's control socket
func PrismSocket(instance string) string {
	return filepath.Join(RuntimeDir(), fmt.Sprintf("prism-%s.sock", instance))
}

// DefaultConfigPath returns the default configuration file path
func DefaultConfigPath() string {
	return filepath.Join(ConfigDir(), "shine.toml")
}
