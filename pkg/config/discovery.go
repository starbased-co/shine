package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

// PrismSource represents where a prism configuration was discovered
type PrismSource int

const (
	SourceUnknown        PrismSource = iota
	SourceShineToml                  // From shine.toml [prisms.*]
	SourcePrismDir                   // From directory with prism.toml
	SourceStandaloneTOML             // From standalone .toml file in prisms/
	SourceBinaryOnly                 // Binary found but no config
)

// DiscoveredPrism represents a discovered prism with its source
type DiscoveredPrism struct {
	Config *PrismConfig
	Source PrismSource
	Path   string // Path to config file or directory
}

// DiscoverPrisms searches for prisms in the configured prism directories
// Returns a map of prism name -> discovered prism information
func DiscoverPrisms(prismDirs []string) (map[string]*DiscoveredPrism, error) {
	discovered := make(map[string]*DiscoveredPrism)

	for _, baseDir := range prismDirs {
		// Expand home directory
		expandedDir := expandPath(baseDir)

		// Check if directory exists
		if _, err := os.Stat(expandedDir); os.IsNotExist(err) {
			continue
		}

		// Read directory entries
		entries, err := os.ReadDir(expandedDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			// Type 1 & 2: Directory-based prisms
			if entry.IsDir() {
				prism, err := discoverDirectoryPrism(expandedDir, entry.Name())
				if err == nil && prism != nil {
					// Only add if not already discovered (first match wins)
					if _, exists := discovered[prism.Config.Name]; !exists {
						discovered[prism.Config.Name] = prism
					}
				}
				continue
			}

			// Type 3: Standalone .toml files
			if strings.HasSuffix(entry.Name(), ".toml") && entry.Name() != "prism.toml" {
				prism, err := discoverStandalonePrism(expandedDir, entry.Name())
				if err == nil && prism != nil {
					// Only add if not already discovered (first match wins)
					if _, exists := discovered[prism.Config.Name]; !exists {
						discovered[prism.Config.Name] = prism
					}
				}
			}
		}
	}

	return discovered, nil
}

// discoverDirectoryPrism handles Type 1 (full package) and Type 2 (data directory)
func discoverDirectoryPrism(baseDir, dirName string) (*DiscoveredPrism, error) {
	prismDir := filepath.Join(baseDir, dirName)
	manifestPath := filepath.Join(prismDir, "prism.toml")

	// Check for prism.toml
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("no prism.toml found in %s", prismDir)
	}

	// Load prism.toml
	config, err := loadPrismConfig(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load prism.toml: %w", err)
	}

	// Validate name is set
	if config.Name == "" {
		return nil, fmt.Errorf("prism.toml missing required 'name' field")
	}

	// Try to resolve binary path
	var resolvedPath string
	if config.Path != "" {
		// Check if binary is in the prism directory (Type 1: Full package)
		binaryPath := filepath.Join(prismDir, config.Path)
		if isExecutable(binaryPath) {
			resolvedPath = binaryPath
		} else {
			// Type 2: Data directory, binary via PATH
			resolvedPath = findInPATH(config.Path)
		}
	} else {
		// Default binary name
		defaultName := "shine-" + config.Name
		binaryPath := filepath.Join(prismDir, defaultName)
		if isExecutable(binaryPath) {
			resolvedPath = binaryPath
		} else {
			resolvedPath = findInPATH(defaultName)
		}
	}

	config.ResolvedPath = resolvedPath

	return &DiscoveredPrism{
		Config: config,
		Source: SourcePrismDir,
		Path:   manifestPath,
	}, nil
}

// discoverStandalonePrism handles Type 3 (standalone .toml files)
func discoverStandalonePrism(baseDir, filename string) (*DiscoveredPrism, error) {
	configPath := filepath.Join(baseDir, filename)

	// Load configuration
	config, err := loadPrismConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", filename, err)
	}

	// Validate name is set
	if config.Name == "" {
		return nil, fmt.Errorf("%s missing required 'name' field", filename)
	}

	// Resolve binary path (always via PATH for standalone configs)
	var binaryName string
	if config.Path != "" {
		binaryName = config.Path
	} else {
		binaryName = "shine-" + config.Name
	}

	config.ResolvedPath = findInPATH(binaryName)

	return &DiscoveredPrism{
		Config: config,
		Source: SourceStandaloneTOML,
		Path:   configPath,
	}, nil
}

// loadPrismConfig loads a PrismConfig from a TOML file
func loadPrismConfig(path string) (*PrismConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config PrismConfig
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// loadShineConfig loads the main shine.toml configuration
func loadShineConfig(path string) (*Config, error) {
	data, err := os.ReadFile(expandPath(path))
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// MergePrismConfigs merges user config from shine.toml over defaults from prism sources
func MergePrismConfigs(prismSource, userConfig *PrismConfig) *PrismConfig {
	merged := &PrismConfig{}

	// === Core Identification ===
	merged.Name = prismSource.Name
	if userConfig.Name != "" {
		merged.Name = userConfig.Name
	}

	merged.Version = prismSource.Version // Version always from source

	if userConfig.Path != "" {
		merged.Path = userConfig.Path
	} else {
		merged.Path = prismSource.Path
	}

	// === Runtime State ===
	// User's enabled setting takes priority
	merged.Enabled = userConfig.Enabled || prismSource.Enabled

	// === Positioning & Layout ===
	merged.Origin = prismSource.Origin
	if userConfig.Origin != "" {
		merged.Origin = userConfig.Origin
	}

	merged.Position = prismSource.Position
	if userConfig.Position != "" {
		merged.Position = userConfig.Position
	}

	merged.Width = prismSource.Width
	if userConfig.Width != nil {
		merged.Width = userConfig.Width
	}

	merged.Height = prismSource.Height
	if userConfig.Height != nil {
		merged.Height = userConfig.Height
	}

	// === Behavior ===
	merged.HideOnFocusLoss = prismSource.HideOnFocusLoss
	if userConfig.HideOnFocusLoss {
		merged.HideOnFocusLoss = userConfig.HideOnFocusLoss
	}

	merged.FocusPolicy = prismSource.FocusPolicy
	if userConfig.FocusPolicy != "" {
		merged.FocusPolicy = userConfig.FocusPolicy
	}

	merged.OutputName = prismSource.OutputName
	if userConfig.OutputName != "" {
		merged.OutputName = userConfig.OutputName
	}

	// === Metadata ===
	// Note: Metadata from user config is intentionally skipped
	merged.Metadata = prismSource.Metadata

	// === Internal fields ===
	merged.ResolvedPath = prismSource.ResolvedPath

	return merged
}

// findInPATH searches for an executable in the system PATH
func findInPATH(name string) string {
	path, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return path
}

// isExecutable checks if a file exists and is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Check if file and has execute permission
	return !info.IsDir() && (info.Mode()&0111 != 0)
}

// expandPath expands ~ to home directory
func expandPath(path string) string {
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
