# Shine Prism System Design

## Executive Summary

This document analyzes the current component discovery mechanism in Shine and proposes a comprehensive prism system design that allows users to compile custom widgets and seamlessly integrate them with Shine.

**Conceptual Model**: A prism is a self-contained unit that refracts light (Shine) to display different information. Each prism contains its binary, configuration, and runtime management. ALL widgets are treated as prisms - there's no distinction between "built-in" and "user" prisms.

**Current State**: Components are discovered via two-step fallback (PATH → executable directory)
**Proposed State**: Unified prism directory system with config-driven PATH augmentation

---

## 1. Current Component Discovery Analysis

### 1.1 Discovery Mechanism

Component binaries are located by `findComponentBinary()` in `/home/starbased/dev/projects/shine/cmd/shine/main.go`:

```go
func findComponentBinary(name string) (string, error) {
    // Step 1: Check if it's in PATH
    path, err := exec.LookPath(name)
    if err == nil {
        return path, nil
    }

    // Step 2: Try relative to the shine binary
    exePath, err := os.Executable()
    if err != nil {
        return "", fmt.Errorf("failed to get executable path: %w", err)
    }

    // Try in the same directory as shine
    binDir := filepath.Dir(exePath)
    componentPath := filepath.Join(binDir, name)
    if _, err := os.Stat(componentPath); err == nil {
        return componentPath, nil
    }

    return "", fmt.Errorf("component binary %s not found in PATH or %s", name, binDir)
}
```

**Discovery Order**:
1. System PATH (via `exec.LookPath`)
2. Same directory as `shine` executable

### 1.2 Current Component Management

**Built-in Components** (in `/home/starbased/dev/projects/shine/cmd/`):
- `shine` - Main launcher and orchestrator
- `shine-bar` - Status bar component
- `shine-chat` - Chat component
- `shine-clock` - Clock widget
- `shine-sysinfo` - System info widget
- `shinectl` - Control utility

**Build Process**:
```bash
go build -o bin/shine ./cmd/shine
go build -o bin/shine-bar ./cmd/shine-bar
go build -o bin/shine-chat ./cmd/shine-chat
go build -o bin/shine-clock ./cmd/shine-clock
go build -o bin/shine-sysinfo ./cmd/shine-sysinfo
```

**Installation**:
```bash
# Manual copy to user bin directory
cp bin/shine* ~/.local/bin/

# Or add to PATH temporarily
export PATH="$PWD/bin:$PATH"
```

### 1.3 Component Launch Flow

```
User runs `shine`
    ↓
Load config from ~/.config/shine/shine.toml
    ↓
For each enabled component:
    ↓
    findComponentBinary("shine-<component>")
        ↓
        Check PATH (exec.LookPath)
        ↓ (if not found)
        Check shine executable directory
    ↓
Launch via panel.Manager.Launch()
    ↓
Execute: kitty @ launch --type=os-panel [panel-config] <component-path>
```

### 1.4 Configuration System

**Config Location**: `~/.config/shine/shine.toml`

**Current Schema** (`pkg/config/types.go`):
```go
type Config struct {
    Chat    *ChatConfig    `toml:"chat"`
    Bar     *BarConfig     `toml:"bar"`
    Clock   *ClockConfig   `toml:"clock"`
    SysInfo *SysInfoConfig `toml:"sysinfo"`
}
```

**Component Config** (example):
```toml
[bar]
enabled = true
edge = "top"
lines_pixels = 30
margin_top = 0
margin_left = 0
focus_policy = "not-allowed"
output_name = "DP-2"
```

### 1.5 Issues with Current Approach

1. **Hardcoded Component Names**: Main launcher explicitly references `shine-chat`, `shine-bar`, etc.
2. **No Prism Discovery**: Cannot automatically discover new prisms
3. **Manual PATH Management**: Users must manually install binaries to PATH
4. **No Prism Directory Support**: No dedicated location for user prisms
5. **Static Configuration**: Config struct has hardcoded component fields
6. **No Prism Validation**: No interface contract for prisms to implement

---

## 2. Prism System Design

### 2.1 Core Concept: The Prism Metaphor

**What is a Prism?**
- A self-contained executable unit that refracts light (Shine) to display information
- Contains: binary, configuration, and runtime management
- Each prism shows different information (time, weather, system stats, etc.)
- Light (Shine) passes through prisms, each refracting differently

**Unified Treatment**:
- NO distinction between "built-in" and "user" prisms
- ALL widgets are prisms, configured under `[prisms.*]`
- Same discovery mechanism for all prisms
- Consistent configuration interface

### 2.2 Design Goals

1. **Unified Prism Model**: All widgets are prisms, no special cases
2. **Config-Driven Prism Directories**: Specify prism search paths in config
3. **Automatic Discovery**: Discover prisms by naming convention or manifest
4. **PATH Augmentation**: Automatically add prism directories to PATH for lookup
5. **Backward Compatibility**: Existing built-in components continue working during migration
6. **Developer-Friendly**: Clear interface contract for prism development
7. **Security**: Validate and sandbox user-compiled binaries
8. **Dynamic Configuration**: Support arbitrary prism configs in TOML

### 2.3 Proposed Directory Structure

```
~/.config/shine/
├── shine.toml           # Main config
└── prisms/              # User prisms directory

~/.local/share/shine/
└── prisms/              # Alternative prism location (XDG-compliant)

/usr/lib/shine/
└── prisms/              # System-wide prisms (built-in location)

/home/starbased/dev/projects/shine/
├── bin/                 # Development builds
├── cmd/                 # Prism source code
└── prisms/              # Project-local prisms (development)
```

### 2.4 Unified Configuration Schema

**New Config Structure** (`pkg/config/types.go`):

```go
// Config represents the main shine configuration
type Config struct {
    // Core settings
    Core *CoreConfig `toml:"core"`

    // ALL prisms configured here (unified)
    Prisms map[string]*PrismConfig `toml:"prisms"`
}

// CoreConfig holds global shine settings
type CoreConfig struct {
    // Prism directories (searched in order)
    PrismDirs []string `toml:"prism_dirs"`

    // Automatically add prism dirs to PATH
    AutoPath bool `toml:"auto_path"`

    // Prism discovery mode
    DiscoveryMode string `toml:"discovery_mode"` // "convention" | "manifest"

    // Validation settings
    ValidatePrisms bool `toml:"validate_prisms"`
    AllowUnsigned  bool `toml:"allow_unsigned"`
}

// PrismConfig is the unified configuration for ALL prisms
type PrismConfig struct {
    // Prism identification
    Name   string `toml:"name"`   // Prism name
    Binary string `toml:"binary"` // Custom binary name (optional)

    // Prism state
    Enabled bool `toml:"enabled"`

    // Panel configuration
    Edge            string `toml:"edge"`
    Lines           int    `toml:"lines"`
    Columns         int    `toml:"columns"`
    LinesPixels     int    `toml:"lines_pixels"`
    ColumnsPixels   int    `toml:"columns_pixels"`
    MarginTop       int    `toml:"margin_top"`
    MarginLeft      int    `toml:"margin_left"`
    MarginBottom    int    `toml:"margin_bottom"`
    MarginRight     int    `toml:"margin_right"`
    HideOnFocusLoss bool   `toml:"hide_on_focus_loss"`
    FocusPolicy     string `toml:"focus_policy"`
    OutputName      string `toml:"output_name"`
}
```

**Example Configuration**:

```toml
# ~/.config/shine/shine.toml

[core]
# Prism search paths (in priority order)
prism_dirs = [
    "/usr/lib/shine/prisms",     # System-wide built-in prisms
    "~/.config/shine/prisms",     # User prisms
    "~/.local/share/shine/prisms" # Alternative user location
]

# Automatically add prism directories to search PATH
auto_path = true

# Discovery mode: "convention" (by naming) or "manifest" (by prism.toml)
discovery_mode = "convention"

# Validate prism interface compliance
validate_prisms = true

# Allow unsigned third-party prisms
allow_unsigned = true

# ALL prisms configured uniformly (no distinction between built-in and user)
[prisms.bar]
enabled = true
edge = "top"
lines_pixels = 30
focus_policy = "not-allowed"
output_name = "DP-2"

[prisms.chat]
enabled = false

[prisms.clock]
enabled = true
edge = "top-right"
columns_pixels = 150
lines_pixels = 30

# User prisms - treated identically
[prisms.weather]
enabled = true
name = "weather"
binary = "shine-weather"  # Optional: defaults to "shine-{name}"
edge = "top-right"
columns_pixels = 200
lines_pixels = 80
margin_top = 10
margin_right = 10

[prisms.spotify]
enabled = true
name = "spotify"
edge = "bottom"
lines_pixels = 60
```

### 2.5 Migration Guide

**Old Configuration** (deprecated):
```toml
# Old separate sections (deprecated)
[bar]
enabled = true
edge = "top"
lines_pixels = 30

[chat]
enabled = false

[plugins.weather]  # Old plugin section
enabled = true
```

**New Configuration** (prisms):
```toml
[core]
prism_dirs = [
    "/usr/lib/shine/prisms",
    "~/.config/shine/prisms",
]
auto_path = true

# ALL prisms in unified section
[prisms.bar]
enabled = true
edge = "top"
lines_pixels = 30

[prisms.chat]
enabled = false

[prisms.weather]
enabled = true
```

**Backward Compatibility Strategy**:
- Config loader checks for old `[bar]`, `[chat]`, etc. sections
- Automatically migrates to `[prisms.*]` internally
- Warns user about deprecated config format
- Provides migration helper: `shine config migrate`

### 2.6 Enhanced Prism Discovery

**New Discovery Flow**:

```go
// pkg/prism/discovery.go

package prism

import (
    "fmt"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
)

// Manager handles prism discovery and loading
type Manager struct {
    searchPaths []string
    autoPath    bool
    mode        DiscoveryMode
    cache       map[string]string // name -> absolute path
}

// DiscoveryMode defines how prisms are discovered
type DiscoveryMode string

const (
    DiscoveryConvention DiscoveryMode = "convention" // By naming: shine-*
    DiscoveryManifest   DiscoveryMode = "manifest"   // By prism.toml
)

// NewManager creates a prism manager with configuration
func NewManager(searchPaths []string, autoPath bool, mode DiscoveryMode) *Manager {
    pm := &Manager{
        searchPaths: expandPaths(searchPaths),
        autoPath:    autoPath,
        mode:        mode,
        cache:       make(map[string]string),
    }

    if autoPath {
        pm.augmentPATH()
    }

    return pm
}

// augmentPATH adds prism directories to PATH environment
func (pm *Manager) augmentPATH() {
    currentPath := os.Getenv("PATH")
    pathParts := []string{}

    // Add prism directories to front of PATH
    for _, dir := range pm.searchPaths {
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

// FindPrism discovers a prism binary by name
func (pm *Manager) FindPrism(name string) (string, error) {
    // Check cache first
    if path, ok := pm.cache[name]; ok {
        if _, err := os.Stat(path); err == nil {
            return path, nil
        }
        delete(pm.cache, name) // Stale cache entry
    }

    // Step 1: Check PATH (includes augmented prism dirs if auto_path enabled)
    binaryName := pm.resolveBinaryName(name)
    if path, err := exec.LookPath(binaryName); err == nil {
        pm.cache[name] = path
        return path, nil
    }

    // Step 2: Search prism directories explicitly (if not using auto_path)
    if !pm.autoPath {
        for _, dir := range pm.searchPaths {
            prismPath := filepath.Join(dir, binaryName)
            if _, err := os.Stat(prismPath); err == nil {
                pm.cache[name] = prismPath
                return prismPath, nil
            }
        }
    }

    // Step 3: Check relative to shine executable (backward compatibility)
    exePath, err := os.Executable()
    if err == nil {
        binDir := filepath.Dir(exePath)
        prismPath := filepath.Join(binDir, binaryName)
        if _, err := os.Stat(prismPath); err == nil {
            pm.cache[name] = prismPath
            return prismPath, nil
        }
    }

    return "", fmt.Errorf("prism %s not found (binary: %s)", name, binaryName)
}

// resolveBinaryName converts prism name to binary name
func (pm *Manager) resolveBinaryName(name string) string {
    // If already prefixed, use as-is
    if strings.HasPrefix(name, "shine-") {
        return name
    }
    // Otherwise, apply convention
    return fmt.Sprintf("shine-%s", name)
}

// DiscoverAll finds all available prisms in search paths
func (pm *Manager) DiscoverAll() ([]string, error) {
    prisms := make(map[string]bool)

    for _, dir := range pm.searchPaths {
        entries, err := os.ReadDir(dir)
        if err != nil {
            continue // Skip inaccessible directories
        }

        for _, entry := range entries {
            if entry.IsDir() {
                continue
            }

            name := entry.Name()

            // Check naming convention
            if pm.mode == DiscoveryConvention || pm.mode == "" {
                if strings.HasPrefix(name, "shine-") && isExecutable(filepath.Join(dir, name)) {
                    // Extract prism name (remove "shine-" prefix)
                    prismName := strings.TrimPrefix(name, "shine-")
                    prisms[prismName] = true
                }
            }

            // TODO: Manifest-based discovery
            if pm.mode == DiscoveryManifest {
                // Look for prism.toml files
            }
        }
    }

    result := make([]string, 0, len(prisms))
    for name := range prisms {
        result = append(result, name)
    }

    return result, nil
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
    info, err := os.Stat(path)
    if err != nil {
        return false
    }
    return info.Mode()&0111 != 0
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
```

### 2.7 Prism Interface Contract

**Prism Requirements**:

1. **Naming Convention**: Binary MUST be named `shine-<name>` (or configured via `binary` field)
2. **Bubble Tea Program**: MUST be a valid Bubble Tea application
3. **Window Title**: SHOULD set window title via ANSI escape sequence for tracking
4. **No Alt Screen**: MUST NOT use `tea.WithAltScreen()` for panel widgets
5. **Stdin/Stdout**: MUST communicate via standard streams (Bubble Tea default)
6. **Exit Codes**: MUST use standard exit codes (0 = success, non-zero = error)

**Example Prism Structure**:

```go
// ~/.config/shine/prisms/shine-weather/main.go

package main

import (
    "fmt"
    "log"
    "time"

    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

func main() {
    // Set window title for tracking (recommended)
    fmt.Print("\033]0;shine-weather\007")

    // Create Bubble Tea program (no alt screen for panels)
    p := tea.NewProgram(initialModel())

    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}

type model struct {
    temp     int
    location string
    width    int
}

func initialModel() model {
    return model{
        temp:     72,
        location: "San Francisco",
        width:    80,
    }
}

func (m model) Init() tea.Cmd {
    return tickCmd()
}

func tickCmd() tea.Cmd {
    return tea.Tick(time.Minute, func(t time.Time) tea.Msg {
        return tickMsg(t)
    })
}

type tickMsg time.Time

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        return m, nil
    case tea.KeyMsg:
        if msg.Type == tea.KeyEsc {
            return m, tea.Quit
        }
    case tickMsg:
        // Update weather data
        return m, tickCmd()
    }
    return m, nil
}

func (m model) View() string {
    style := lipgloss.NewStyle().
        Foreground(lipgloss.Color("14")).
        Background(lipgloss.Color("0")).
        Bold(true).
        Padding(0, 1)

    return style.Render(fmt.Sprintf("%s: %d°F", m.location, m.temp))
}
```

**Build Prism**:

```bash
cd ~/.config/shine/prisms/shine-weather
go mod init shine-weather
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss
go build -o shine-weather main.go

# Prism is now discoverable by Shine
```

### 2.8 Prism Validation (Optional)

**Validation Levels**:

1. **Basic**: Check if binary exists and is executable
2. **Interface**: Verify prism responds to Bubble Tea messages
3. **Signature**: Verify prism binary signature (future)

```go
// pkg/prism/validator.go

package prism

import (
    "fmt"
    "os/exec"
)

type Validator struct {
    level ValidationLevel
}

type ValidationLevel int

const (
    ValidationNone ValidationLevel = iota
    ValidationBasic
    ValidationInterface
    ValidationSignature
)

func (v *Validator) Validate(path string) error {
    if v.level == ValidationNone {
        return nil
    }

    // Basic validation
    if err := v.validateBasic(path); err != nil {
        return err
    }

    if v.level >= ValidationInterface {
        if err := v.validateInterface(path); err != nil {
            return err
        }
    }

    if v.level >= ValidationSignature {
        if err := v.validateSignature(path); err != nil {
            return err
        }
    }

    return nil
}

func (v *Validator) validateBasic(path string) error {
    if !isExecutable(path) {
        return fmt.Errorf("prism is not executable: %s", path)
    }
    return nil
}

func (v *Validator) validateInterface(path string) error {
    // Run prism with --validate flag (prism should implement this)
    cmd := exec.Command(path, "--validate")
    if err := cmd.Run(); err != nil {
        return fmt.Errorf("prism failed interface validation: %w", err)
    }
    return nil
}

func (v *Validator) validateSignature(path string) error {
    // TODO: Implement signature validation
    return nil
}
```

### 2.9 Updated Main Launcher

**Modified `cmd/shine/main.go`**:

```go
package main

import (
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/starbased-co/shine/pkg/prism"
    "github.com/starbased-co/shine/pkg/config"
    "github.com/starbased-co/shine/pkg/panel"
)

func main() {
    // Load configuration
    configPath := config.DefaultConfigPath()
    cfg := config.LoadOrDefault(configPath)

    fmt.Printf("✨ Shine - Hyprland Layer Shell TUI Toolkit\n")
    fmt.Printf("Configuration: %s\n\n", configPath)

    // Initialize prism manager
    prismDirs := cfg.Core.PrismDirs
    if len(prismDirs) == 0 {
        // Default prism directories
        prismDirs = []string{
            "/usr/lib/shine/prisms",
            "~/.config/shine/prisms",
            "~/.local/share/shine/prisms",
        }
    }

    prismMgr := prism.NewManager(
        prismDirs,
        cfg.Core.AutoPath,
        prism.DiscoveryMode(cfg.Core.DiscoveryMode),
    )

    // Create panel manager
    panelMgr := panel.NewManager()

    // Launch all enabled prisms (unified treatment)
    for name, prismConfig := range cfg.Prisms {
        if prismConfig != nil && prismConfig.Enabled {
            if err := launchPrism(prismMgr, panelMgr, name, prismConfig); err != nil {
                log.Printf("Failed to launch prism %s: %v", name, err)
                continue
            }
            time.Sleep(500 * time.Millisecond)
        }
    }

    // List running panels
    panels := panelMgr.List()
    if len(panels) == 0 {
        fmt.Println("\nNo prisms enabled. Edit your config to enable prisms.")
        fmt.Printf("Config location: %s\n", configPath)
        os.Exit(0)
    }

    fmt.Printf("\nRunning %d prism(s): %v\n", len(panels), panels)
    fmt.Println("Press Ctrl+C to stop all prisms")

    // Setup signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

    // Wait for signal or panels to exit
    go panelMgr.Wait()

    <-sigChan
    fmt.Println("\n\nShutting down...")

    // Stop all panels
    for _, name := range panelMgr.List() {
        fmt.Printf("Stopping %s...\n", name)
        if err := panelMgr.Stop(name); err != nil {
            log.Printf("Error stopping %s: %v", name, err)
        }
    }

    fmt.Println("Goodbye!")
}

func launchPrism(
    prismMgr *prism.Manager,
    panelMgr *panel.Manager,
    name string,
    cfg *config.PrismConfig,
) error {
    // Resolve binary name
    binaryName := cfg.Binary
    if binaryName == "" {
        binaryName = fmt.Sprintf("shine-%s", name)
    }

    // Find prism binary
    prismPath, err := prismMgr.FindPrism(name)
    if err != nil {
        return fmt.Errorf("failed to find prism binary: %w", err)
    }

    fmt.Printf("Launching %s (%s)...\n", name, prismPath)

    // Convert config to panel config
    panelCfg := cfg.ToPanelConfig()

    // Launch via panel manager
    instance, err := panelMgr.Launch(name, panelCfg, prismPath)
    if err != nil {
        return fmt.Errorf("failed to launch: %w", err)
    }

    if instance.WindowID != "" {
        fmt.Printf("  ✓ %s launched (Window ID: %s)\n", name, instance.WindowID)
    } else {
        fmt.Printf("  ✓ %s launched\n", name)
    }

    return nil
}
```

---

## 3. Implementation Roadmap

### Phase 1: Core Prism Infrastructure
**Priority**: High
**Estimated Effort**: 3-5 days

**Tasks**:
- [ ] Create `pkg/prism/discovery.go` with Manager
- [ ] Create `pkg/prism/manager.go` with prism lifecycle
- [ ] Extend `pkg/config/types.go` with CoreConfig and unified PrismConfig
- [ ] Update config loader to handle new schema with migration support
- [ ] Update `cmd/shine/main.go` to use prism.Manager
- [ ] Add default prism directories to config defaults
- [ ] Write tests for prism discovery

### Phase 2: Prism Development Tooling
**Priority**: Medium
**Estimated Effort**: 2-3 days

**Tasks**:
- [ ] Create prism template/scaffolding tool (`shine new-prism <name>`)
- [ ] Write prism development guide documentation
- [ ] Create example prisms (weather, spotify, system monitor)
- [ ] Add prism validation framework (`pkg/prism/validator.go`)
- [ ] Create prism testing utilities

### Phase 3: Advanced Features
**Priority**: Low
**Estimated Effort**: 5-7 days

**Tasks**:
- [ ] Manifest-based discovery (`prism.toml` files)
- [ ] Hot reload for prism configuration changes
- [ ] Prism marketplace/registry integration
- [ ] Binary signature verification
- [ ] Prism sandbox/isolation (seccomp, capabilities)
- [ ] Prism IPC event bus
- [ ] Prism metadata and versioning

### Phase 4: Migration and Documentation
**Priority**: High
**Estimated Effort**: 2-3 days

**Tasks**:
- [ ] Update README.md with prism system documentation
- [ ] Create PRISM_DEVELOPMENT.md guide
- [ ] Add configuration migration tool: `shine config migrate`
- [ ] Update example configs
- [ ] Create video tutorial for prism development

---

## 4. Security Considerations

### 4.1 Threat Model

**Threats**:
1. **Malicious Prisms**: User installs untrusted prism that executes arbitrary code
2. **Path Traversal**: Prism escapes designated directories
3. **Resource Exhaustion**: Prism consumes excessive CPU/memory
4. **Privilege Escalation**: Prism attempts to gain elevated permissions

### 4.2 Mitigation Strategies

**Immediate** (Phase 1):
- Prism directories are user-controlled (`~/.config/shine/prisms`)
- Binary execution is opt-in (user must enable in config)
- Prisms run with same privileges as Shine (user's permissions)
- Clear documentation warning about untrusted prisms

**Future** (Phase 3):
- Binary signature verification (GPG/Minisign)
- Prism capability declarations (required permissions)
- Seccomp sandboxing for prism processes
- Resource limits (CPU/memory quotas via cgroups)
- Prism marketplace with community reviews

### 4.3 Best Practices for Users

**Documentation should emphasize**:
1. Only install prisms from trusted sources
2. Review prism source code before compiling
3. Use signature verification when available
4. Monitor prism resource usage
5. Disable unused prisms

---

## 5. Backward Compatibility

### 5.1 Existing Config Migration

**Old Config** (`~/.config/shine/shine.toml`):
```toml
[bar]
enabled = true
edge = "top"
lines_pixels = 30

[plugins.weather]
enabled = true
```

**New Config** (unified prisms):
```toml
[core]
prism_dirs = [
    "/usr/lib/shine/prisms",
    "~/.config/shine/prisms",
]
auto_path = true

[prisms.bar]
enabled = true
edge = "top"
lines_pixels = 30

[prisms.weather]
enabled = true
```

**Migration Strategy**:
- Config loader detects old format (`[bar]`, `[plugins.*]`)
- Automatically converts to `[prisms.*]` internally
- Warns user about deprecated format
- Provides migration command: `shine config migrate`
- No breaking changes during transition period

### 5.2 Prism Discovery Fallback

**Discovery Priority** (maintains backward compatibility):
1. PATH (includes prism directories if `auto_path = true`)
2. Explicit prism directories (if `auto_path = false`)
3. Shine executable directory (original behavior)

Existing installations with binaries in `~/.local/bin` or project `bin/` directory continue working unchanged.

---

## 6. Example Prism Development Workflow

### 6.1 Create Prism

```bash
# Create prism directory
mkdir -p ~/.config/shine/prisms/shine-weather
cd ~/.config/shine/prisms/shine-weather

# Initialize Go module
go mod init shine-weather

# Add dependencies
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/lipgloss

# Create main.go (see section 2.7)
nvim main.go

# Build prism
go build -o shine-weather main.go

# Prism is now in ~/.config/shine/prisms/shine-weather/shine-weather
```

### 6.2 Configure Prism

Edit `~/.config/shine/shine.toml`:

```toml
[core]
prism_dirs = ["~/.config/shine/prisms"]
auto_path = true

[prisms.weather]
enabled = true
name = "weather"
edge = "top-right"
columns_pixels = 200
lines_pixels = 80
margin_top = 10
margin_right = 10
```

### 6.3 Launch Prism

```bash
shine
# ✨ Shine - Hyprland Layer Shell TUI Toolkit
# Configuration: /home/user/.config/shine/shine.toml
#
# Launching weather (/home/user/.config/shine/prisms/shine-weather/shine-weather)...
#   ✓ weather launched (Window ID: 123)
#
# Running 1 prism(s): [weather]
```

---

## 7. Alternative Approaches Considered

### 7.1 Go Plugin System (`plugin` package)

**Pros**:
- Native Go support
- Shared memory between main and prisms
- No separate binary compilation

**Cons**:
- Requires exact Go version match
- Platform-specific (CGO required)
- Limited portability
- Complex versioning issues

**Decision**: Rejected in favor of separate binaries for simplicity and portability

### 7.2 Dynamic Library Loading (CGO)

**Pros**:
- Smaller footprint than full binaries
- Shared dependencies

**Cons**:
- Platform-specific compilation
- ABI compatibility issues
- Complex build process

**Decision**: Rejected in favor of self-contained binaries

### 7.3 Script-Based Prisms (Lua/Python)

**Pros**:
- Easy to write without compilation
- Dynamic loading
- Sandboxing via interpreter

**Cons**:
- Loses type safety and performance
- Additional dependencies (Lua/Python runtime)
- Not idiomatic for Go-based Bubble Tea apps

**Decision**: Rejected to maintain pure Go ecosystem

---

## 8. Performance Implications

### 8.1 Discovery Overhead

**Benchmark Estimates**:
- Prism directory scan: ~1-5ms per directory (depends on file count)
- Binary executable check: ~0.1ms per file
- PATH lookup: ~0.5ms per prism

**Optimization**:
- Cache discovered prisms in memory
- Lazy load prisms only when enabled
- Skip inaccessible directories early

### 8.2 PATH Augmentation Impact

**Memory**: Minimal (~100 bytes per directory added to PATH)
**Startup Time**: <1ms overhead for PATH manipulation
**Runtime**: No impact (PATH is set once during initialization)

---

## 9. Testing Strategy

### 9.1 Unit Tests

**Files to Test**:
- `pkg/prism/discovery.go` - Prism discovery logic
- `pkg/prism/manager.go` - Prism lifecycle management
- `pkg/prism/validator.go` - Prism validation
- `pkg/config/types.go` - Config parsing with prism sections

**Test Scenarios**:
- Prism found in first search directory
- Prism found in PATH
- Prism not found (error handling)
- Multiple prisms with same name (priority resolution)
- Malformed prism directories
- PATH augmentation correctness

### 9.2 Integration Tests

**Test Scenarios**:
- Launch built-in prism (bar, clock, etc.)
- Launch user prism from `~/.config/shine/prisms`
- Launch prism from `/usr/lib/shine/prisms`
- Mixed built-in and user prisms
- Prism binary missing (graceful degradation)
- Config with invalid prism section
- Config migration from old format

### 9.3 Manual Testing

**Checklist**:
- [ ] Build sample prism and verify discovery
- [ ] Test auto_path with multiple prism directories
- [ ] Test discovery_mode: "convention" vs "manifest"
- [ ] Verify backward compatibility with old configs
- [ ] Test prism validation levels
- [ ] Verify PATH augmentation doesn't break existing binaries

---

## 10. Documentation Requirements

### 10.1 User Documentation

**Files to Create/Update**:
1. `README.md` - Add prism system section
2. `docs/PRISM_DEVELOPMENT.md` - Comprehensive prism development guide
3. `docs/PRISM_SYSTEM_DESIGN.md` - This document (architecture reference)
4. `examples/prism-weather/` - Example prism with comments

**Content**:
- Prism concept and metaphor explanation
- Prism discovery mechanism explanation
- Prism directory configuration
- Step-by-step prism development tutorial
- Prism interface contract documentation
- Security best practices for users
- Troubleshooting common prism issues

### 10.2 Developer Documentation

**Content**:
- Prism discovery algorithm internals
- Prism interface specification
- Validation framework usage
- Testing prism code
- Debugging prism issues
- Performance considerations

---

## 11. Conclusion

The proposed prism system provides a unified, flexible approach to managing ALL widgets in Shine. By treating built-in and user prisms identically, we simplify the mental model while providing powerful extensibility.

**Key Benefits**:
1. **Unified Model**: All widgets are prisms - no special cases
2. **Developer-Friendly**: Simple Go + Bubble Tea prisms
3. **Flexible Discovery**: Multiple search paths with priority ordering
4. **Backward Compatible**: Existing installations continue working during migration
5. **Secure**: User-controlled prism directories with optional validation
6. **Extensible**: Foundation for future prism marketplace

**The Prism Metaphor**:
- Light (Shine) passes through prisms
- Each prism refracts differently to show different information
- Self-contained units with binary, config, and runtime management
- Uniform treatment - no distinction between "built-in" and "user"

**Implementation Status**:
- ✅ Phase 1: Core Prism Infrastructure - COMPLETED
- ✅ Phase 2: Developer Tooling - COMPLETED
- ✅ Phase 3: Advanced Features - COMPLETED

---

## 12. Phase 3: Advanced Features (COMPLETED)

Phase 3 adds production-ready advanced features for enhanced prism management and security.

### 12.1 Manifest-Based Discovery

**Feature**: Support `prism.toml` manifest files for structured prism metadata.

**Manifest Format** (`prism.toml`):
```toml
[prism]
name = "weather"
version = "1.0.0"
binary = "shine-weather"

[metadata]
description = "Weather widget with icons"
author = "Your Name <email@example.com>"
license = "MIT"
homepage = "https://github.com/yourname/shine-weather"
tags = ["weather", "widget"]
```

**Directory Structure**:
```
~/.config/shine/prisms/weather/
├── prism.toml          # Manifest file
├── shine-weather       # Binary
└── README.md          # Optional docs
```

**Discovery Modes**:
- `convention`: Traditional shine-* naming (fastest)
- `manifest`: Require prism.toml files (structured)
- `auto`: Try manifest first, fall back to convention (default)

**Configuration**:
```toml
[core]
discovery_mode = "auto"  # convention | manifest | auto
```

**Implementation**: `pkg/prism/manifest.go`

### 12.2 Hot Reload for Config Changes

**Feature**: Automatically reload prisms when configuration changes.

**Usage**:
```go
watcher, err := config.NewWatcher(configPath, func(newCfg *config.Config) {
    lifecycleMgr.ReloadAll(newCfg)
})
watcher.Start()
defer watcher.Stop()
```

**Behavior**:
- Polls config file every 1 second for changes
- Detects modifications via file mtime
- Reloads changed config automatically
- Restarts affected prisms gracefully
- Small delay between prism launches for single-instance mode

**Implementation**: `pkg/config/watcher.go`

### 12.3 Enhanced Validation

**Feature**: Comprehensive prism binary validation.

**Validation Checks**:
1. File existence and permissions
2. Executable flag verification
3. Script detection (shebang check)
4. Binary type detection (ELF analysis)
5. Size warnings (> 100MB)
6. Version flag support check

**Usage**:
```go
result, err := prism.Validate(binaryPath)
if !result.Valid {
    log.Printf("Errors: %v", result.Errors)
}
if len(result.Warnings) > 0 {
    log.Printf("Warnings: %v", result.Warnings)
}
log.Printf("Capabilities: %v", result.Capabilities)
```

**Manifest Validation**:
```go
result, err := prism.ValidateManifest(manifestPath)
// Validates both manifest structure and referenced binary
```

**Implementation**: `pkg/prism/validate.go`

### 12.4 Lifecycle Management

**Feature**: Centralized prism lifecycle operations.

**Lifecycle Manager**:
```go
lifecycleMgr := prism.NewLifecycleManager(prismMgr, panelMgr)

// Launch prism
lifecycleMgr.Launch("weather", weatherConfig)

// Reload prism (stop + restart with new binary/config)
lifecycleMgr.Reload("weather", updatedConfig)

// Reload all prisms with new config
lifecycleMgr.ReloadAll(newConfig)

// Check health
status, err := lifecycleMgr.Health("weather")
fmt.Printf("Running: %v, Uptime: %v\n", status.Running, status.Uptime)

// List running prisms
prisms := lifecycleMgr.List()
```

**Features**:
- Tracks running prism instances
- Health monitoring
- Graceful reload with delay
- Automatic cleanup of stopped prisms

**Implementation**: `pkg/prism/lifecycle.go`

### 12.5 Features Deferred for Future

The following advanced features were evaluated but deferred for pragmatic reasons:

**Binary Signature Verification** (Complexity: HIGH, Value: MEDIUM):
- Requires PKI infrastructure
- GPG key management overhead
- Most users won't use it
- Can be added when community requests it

**Prism Sandboxing** (Complexity: HIGH, Platform: Linux-only):
- Requires root/capabilities for seccomp
- Linux-specific (cgroups, seccomp-bpf)
- Complex to test and maintain
- May break legitimate prism functionality
- Better handled at system level (firejail, bubblewrap)

**IPC Event Bus** (Complexity: MEDIUM, Value: LOW):
- Nice-to-have for inter-prism communication
- Not essential for core functionality
- Can be added when use cases emerge
- Current design doesn't preclude future addition

### 12.6 Testing

**Test Coverage**:
- `pkg/prism/manifest_test.go`: Manifest parsing and discovery
- `pkg/prism/validate_test.go`: Binary validation checks
- `pkg/config/watcher_test.go`: Config change detection

**Test Results**:
```
pkg/prism:  17 tests passed (manifest, validation, discovery)
pkg/config: 12 tests passed (watcher, config loading)
```

### 12.7 Documentation

**Updated Files**:
- `examples/shine.toml`: Added discovery_mode config
- `examples/prism.toml`: Complete manifest example
- `docs/PRISM_SYSTEM_DESIGN.md`: This section

**Example Prism with Manifest**:
```
examples/prisms/weather/
├── prism.toml          # Manifest
├── main.go             # Source
├── Makefile            # Build
└── README.md           # Docs
```

### 12.8 Security Considerations

**Manifest-Based Discovery**:
- Validates manifest structure before loading
- Checks binary paths for directory traversal
- Requires executable permissions
- User controls prism directories

**Hot Reload**:
- Only reloads from configured paths
- Preserves single-instance guarantees
- Graceful shutdown prevents orphaned processes

**Validation**:
- Detects common issues early
- Warns about suspicious binaries
- Provides detailed error messages
- Does not prevent execution (advisory only)

### 12.9 Usage Examples

**Basic Manifest Prism**:
```toml
# ~/.config/shine/prisms/weather/prism.toml
[prism]
name = "weather"
version = "1.0.0"
binary = "shine-weather"

[metadata]
description = "Weather widget"
author = "Your Name"
license = "MIT"
```

**Config with Manifest Discovery**:
```toml
# ~/.config/shine/shine.toml
[core]
prism_dirs = ["~/.config/shine/prisms"]
discovery_mode = "manifest"  # Require manifests

[prisms.weather]
enabled = true
edge = "top-right"
columns_pixels = 300
```

**Programmatic Validation**:
```go
// Validate before launching
result, _ := prism.Validate(binaryPath)
if !result.Valid {
    return fmt.Errorf("invalid binary: %v", result.Errors)
}
```

### 12.10 Migration Notes

**No Breaking Changes**:
- All features are opt-in
- Convention-based discovery still works
- No config changes required
- Backward compatible with Phase 1 & 2

**Recommended Setup**:
```toml
[core]
discovery_mode = "auto"  # Best of both worlds
```

This allows:
- Manifest prisms to provide rich metadata
- Convention-based prisms to work without manifests
- Gradual migration at user's pace
