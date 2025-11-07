package prism

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Manifest represents a prism.toml file for manifest-based discovery
type Manifest struct {
	Prism    PrismInfo      `toml:"prism"`
	Metadata map[string]any `toml:"metadata"`
}

// PrismInfo contains core prism metadata
type PrismInfo struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
	Binary  string `toml:"binary"`
}

// DiscoveryMode determines how prisms are discovered
type DiscoveryMode string

const (
	DiscoveryConvention DiscoveryMode = "convention"
	DiscoveryManifest   DiscoveryMode = "manifest"
	DiscoveryAuto       DiscoveryMode = "auto"
)

// LoadManifest reads and parses a prism.toml file
func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest Manifest
	if err := toml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

// Validate checks if the manifest is valid
func (m *Manifest) Validate() error {
	if m.Prism.Name == "" {
		return fmt.Errorf("prism name is required")
	}
	if m.Prism.Binary == "" {
		return fmt.Errorf("prism binary is required")
	}
	return nil
}

// FindManifestDir searches for a prism directory containing prism.toml
// Returns the directory path and manifest, or error if not found
func FindManifestDir(searchPaths []string, prismName string) (string, *Manifest, error) {
	for _, dir := range searchPaths {
		prismDir := filepath.Join(dir, prismName)
		manifestPath := filepath.Join(prismDir, "prism.toml")

		if _, err := os.Stat(manifestPath); err == nil {
			manifest, err := LoadManifest(manifestPath)
			if err != nil {
				continue
			}

			if err := manifest.Validate(); err != nil {
				continue
			}

			return prismDir, manifest, nil
		}
	}

	return "", nil, fmt.Errorf("no manifest found for prism %s", prismName)
}

// DiscoverByManifest finds prisms using manifest-based discovery
// Returns a map of prism name -> binary path
func DiscoverByManifest(searchPaths []string) (map[string]string, error) {
	prisms := make(map[string]string)

	for _, dir := range searchPaths {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			prismDir := filepath.Join(dir, entry.Name())
			manifestPath := filepath.Join(prismDir, "prism.toml")

			if _, err := os.Stat(manifestPath); err != nil {
				continue
			}

			manifest, err := LoadManifest(manifestPath)
			if err != nil {
				continue
			}

			if err := manifest.Validate(); err != nil {
				continue
			}

			binaryPath := filepath.Join(prismDir, manifest.Prism.Binary)
			if isExecutable(binaryPath) {
				prisms[manifest.Prism.Name] = binaryPath
			}
		}
	}

	return prisms, nil
}
