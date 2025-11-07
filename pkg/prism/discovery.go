package prism

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/starbased-co/shine/pkg/config"
)

// Manager handles prism discovery via PATH lookup
type Manager struct {
	cache map[string]string // name -> absolute path cache
}

// NewManager creates a new prism manager
// searchPaths are prepended to PATH for binary discovery
func NewManager(searchPaths []string) *Manager {
	pm := &Manager{
		cache: make(map[string]string),
	}

	pm.augmentPATH(expandPaths(searchPaths))

	return pm
}

// augmentPATH prepends directories to PATH environment variable
func (pm *Manager) augmentPATH(searchPaths []string) {
	currentPath := os.Getenv("PATH")
	pathParts := []string{}

	// Add prism directories to front of PATH
	for _, dir := range searchPaths {
		if _, err := os.Stat(dir); err == nil {
			pathParts = append(pathParts, dir)
		}
	}

	// Append existing PATH
	if currentPath != "" {
		pathParts = append(pathParts, currentPath)
	}

	newPath := strings.Join(pathParts, string(os.PathListSeparator))
	os.Setenv("PATH", newPath)
}

// FindPrism discovers a prism binary by name using PATH lookup
// Discovery:
// 1. Check for explicit binary path in config (if provided)
// 2. Check cache
// 3. Use exec.LookPath (searches PATH which includes prepended prism directories)
func (pm *Manager) FindPrism(name string, cfg *config.PrismConfig) (string, error) {
	// Determine binary name to search for
	binaryName := name
	if cfg != nil && cfg.Binary != "" {
		binaryName = cfg.Binary
	} else {
		// Apply naming convention: prism-{name}
		if !strings.HasPrefix(name, "prism-") {
			binaryName = fmt.Sprintf("prism-%s", name)
		}
	}

	// Check if it's an absolute/relative path
	if strings.Contains(binaryName, string(os.PathSeparator)) {
		absPath, err := filepath.Abs(binaryName)
		if err == nil && isExecutable(absPath) {
			pm.cache[name] = absPath
			return absPath, nil
		}
		return "", fmt.Errorf("binary not found at path: %s", binaryName)
	}

	// Check cache
	if path, ok := pm.cache[name]; ok {
		if isExecutable(path) {
			return path, nil
		}
		delete(pm.cache, name) // Stale cache entry
	}

	// Use PATH lookup (includes prepended prism directories)
	path, err := exec.LookPath(binaryName)
	if err != nil {
		return "", fmt.Errorf("prism %s not found in PATH (binary: %s)", name, binaryName)
	}

	pm.cache[name] = path
	return path, nil
}

// isExecutable checks if a file exists and is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Mode()&0111 != 0
}

// expandPaths expands ~ and environment variables in paths
func expandPaths(paths []string) []string {
	expanded := make([]string, 0, len(paths))

	for _, path := range paths {
		// Expand ~
		if strings.HasPrefix(path, "~/") {
			home, err := os.UserHomeDir()
			if err == nil {
				path = filepath.Join(home, path[2:])
			}
		}

		// Expand environment variables
		path = os.ExpandEnv(path)

		expanded = append(expanded, path)
	}

	return expanded
}
