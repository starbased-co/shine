package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

//go:embed templates/*
var templateFS embed.FS

// prismTemplateData holds data for template rendering
type prismTemplateData struct {
	Name       string // Prism name (e.g., "myprism")
	NameTitle  string // Title case name (e.g., "Myprism")
	WindowName string // Window name (e.g., "shine-myprism")
}

// newPrism creates a new prism from template
func newPrism(name string) error {
	if name == "" {
		return fmt.Errorf("prism name is required")
	}

	// Validate name (lowercase, alphanumeric, hyphens)
	if !isValidPrismName(name) {
		return fmt.Errorf("invalid prism name: must be lowercase alphanumeric with hyphens (e.g., my-prism)")
	}

	// Default location: ~/.config/shine/prisms/<name>
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	targetDir := filepath.Join(homeDir, ".config", "shine", "prisms", name)

	// Check if directory already exists
	if _, err := os.Stat(targetDir); !os.IsNotExist(err) {
		return fmt.Errorf("prism directory already exists: %s", targetDir)
	}

	// Create directory
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Prepare template data
	data := prismTemplateData{
		Name:       name,
		NameTitle:  strings.Title(strings.ReplaceAll(name, "-", " ")),
		WindowName: fmt.Sprintf("shine-%s", name),
	}

	// Files to generate
	files := map[string]string{
		"main.go":    "templates/main.go.tmpl",
		"go.mod":     "templates/go.mod.tmpl",
		"Makefile":   "templates/Makefile.tmpl",
		"README.md":  "templates/README.md.tmpl",
		".gitignore": "templates/gitignore.tmpl",
	}

	// Generate each file
	for filename, tmplPath := range files {
		if err := generateFile(targetDir, filename, tmplPath, data); err != nil {
			return fmt.Errorf("failed to generate %s: %w", filename, err)
		}
	}

	// Print success message with instructions
	printSuccessMessage(name, targetDir)

	return nil
}

// generateFile generates a single file from template
func generateFile(targetDir, filename, tmplPath string, data prismTemplateData) error {
	// Read template from embedded FS
	tmplContent, err := templateFS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read template: %w", err)
	}

	// Parse template
	tmpl, err := template.New(filename).Parse(string(tmplContent))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	// Create target file
	targetPath := filepath.Join(targetDir, filename)
	f, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer f.Close()

	// Execute template
	if err := tmpl.Execute(f, data); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

// isValidPrismName checks if name follows naming conventions
func isValidPrismName(name string) bool {
	if len(name) == 0 {
		return false
	}

	for _, ch := range name {
		if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
			return false
		}
	}

	return true
}

// printSuccessMessage prints instructions after successful creation
func printSuccessMessage(name, targetDir string) {
	fmt.Printf("\n✓ Successfully created prism: %s\n", name)
	fmt.Printf("  Location: %s\n\n", targetDir)
	fmt.Println("Next steps:")
	fmt.Printf("  1. cd %s\n", targetDir)
	fmt.Println("  2. make build          # Build the prism")
	fmt.Println("  3. make install        # Install to ~/.local/bin")
	fmt.Println("  4. Edit ~/.config/shine/shine.toml and add:")
	fmt.Printf("\n")
	fmt.Printf("     [prisms.%s]\n", name)
	fmt.Printf("     enabled = true\n")
	fmt.Printf("     origin = \"top-left\"    # Or: top-center, top-right, center, etc.\n")
	fmt.Printf("     position = \"10,50\"     # Offset from origin (horizontal, vertical)\n")
	fmt.Printf("     width = \"300px\"        # Or: 80 for columns\n")
	fmt.Printf("     height = \"100px\"       # Or: 24 for lines\n")
	fmt.Printf("     focus_policy = \"not-allowed\"\n")
	fmt.Printf("\n")
	fmt.Println("  5. shine               # Launch shine to test")
	fmt.Println()
	fmt.Println("See README.md in the prism directory for more details.")
}
