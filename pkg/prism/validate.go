package prism

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ValidationResult contains validation information
type ValidationResult struct {
	Valid  bool
	Errors []string
}

// Validate performs basic prism binary validation
func Validate(binaryPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	// Check 1: File exists
	_, err := os.Stat(binaryPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("File not found: %v", err))
		return result, nil
	}

	// Check 2: Executable permissions
	if !isExecutable(binaryPath) {
		result.Valid = false
		result.Errors = append(result.Errors, "File is not executable")
		return result, nil
	}

	return result, nil
}

// ValidateManifest validates a prism manifest and its referenced binary
func ValidateManifest(manifestPath string) (*ValidationResult, error) {
	result := &ValidationResult{
		Valid:  true,
		Errors: []string{},
	}

	manifest, err := LoadManifest(manifestPath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to load manifest: %v", err))
		return result, nil
	}

	if err := manifest.Validate(); err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid manifest: %v", err))
		return result, nil
	}

	// Validate referenced binary
	binaryPath := manifest.Prism.Path
	if !strings.Contains(binaryPath, string(os.PathSeparator)) {
		manifestDir := filepath.Dir(manifestPath)
		binaryPath = filepath.Join(manifestDir, binaryPath)
	}

	binaryResult, err := Validate(binaryPath)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Binary validation error: %v", err))
		result.Valid = false
		return result, nil
	}

	// Merge binary validation results
	result.Errors = append(result.Errors, binaryResult.Errors...)
	result.Valid = result.Valid && binaryResult.Valid

	return result, nil
}
