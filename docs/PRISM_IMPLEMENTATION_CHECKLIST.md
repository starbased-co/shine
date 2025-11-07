# Shine Prism System - Implementation Checklist

Quick reference for implementing the prism system. See [PRISM_SYSTEM_DESIGN.md](PRISM_SYSTEM_DESIGN.md) for complete design documentation.

**Concept**: A prism refracts light (Shine) to display information. All widgets are prisms - no distinction between built-in and user prisms.

---

## Phase 1: Core Prism Infrastructure

### 1. Create Prism Discovery Package

**File**: `/home/starbased/dev/projects/shine/pkg/prism/discovery.go`

**Key Functions**:
- [ ] `NewManager(searchPaths []string, autoPath bool, mode DiscoveryMode) *Manager`
- [ ] `(pm *Manager) augmentPATH()` - Add prism dirs to PATH
- [ ] `(pm *Manager) FindPrism(name string) (string, error)` - Find prism binary
- [ ] `(pm *Manager) DiscoverAll() ([]string, error)` - List all available prisms
- [ ] `expandPaths(paths []string) []string` - Expand ~ and env vars
- [ ] `isExecutable(path string) bool` - Check if file is executable

**Discovery Priority**:
1. Check in-memory cache
2. Search PATH (includes prism dirs if auto_path enabled)
3. Search prism directories explicitly (if auto_path disabled)
4. Check relative to shine executable (backward compatibility)

### 2. Create Prism Manager Package

**File**: `/home/starbased/dev/projects/shine/pkg/prism/manager.go`

**Key Functions**:
- [ ] Prism lifecycle management
- [ ] Prism state tracking
- [ ] Prism health monitoring

### 3. Extend Configuration Types

**File**: `/home/starbased/dev/projects/shine/pkg/config/types.go`

**New Structs**:
```go
type CoreConfig struct {
    PrismDirs      []string `toml:"prism_dirs"`
    AutoPath       bool     `toml:"auto_path"`
    DiscoveryMode  string   `toml:"discovery_mode"`
    ValidatePrisms bool     `toml:"validate_prisms"`
    AllowUnsigned  bool     `toml:"allow_unsigned"`
}

type PrismConfig struct {
    Name            string `toml:"name"`
    Binary          string `toml:"binary"`
    Enabled         bool   `toml:"enabled"`
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

**Update Main Config**:
```go
type Config struct {
    Core   *CoreConfig                 `toml:"core"`
    Prisms map[string]*PrismConfig     `toml:"prisms"`
}
```

**Tasks**:
- [ ] Add CoreConfig struct
- [ ] Create unified PrismConfig (replaces ChatConfig/BarConfig/etc)
- [ ] Add Prisms map to main Config
- [ ] Update NewDefaultConfig() with sensible core defaults
- [ ] Add (pc *PrismConfig) ToPanelConfig() method

### 4. Update Configuration Loader

**File**: `/home/starbased/dev/projects/shine/pkg/config/loader.go`

**Tasks**:
- [ ] Ensure Load() handles new Config structure
- [ ] Implement migration from old format:
  - Detect `[bar]`, `[chat]`, `[clock]`, `[sysinfo]` sections
  - Detect `[plugins.*]` sections
  - Convert to `[prisms.*]` internally
  - Warn user about deprecated format
- [ ] Set default prism_dirs if not specified:
  - `/usr/lib/shine/prisms`
  - `~/.config/shine/prisms`
  - `~/.local/share/shine/prisms`
- [ ] Set default auto_path = true
- [ ] Set default discovery_mode = "convention"

### 5. Add Configuration Migration Command

**File**: `/home/starbased/dev/projects/shine/cmd/shine/config.go`

**Tasks**:
- [ ] Implement `shine config migrate` command
- [ ] Read old config format
- [ ] Convert to new prism format
- [ ] Backup old config
- [ ] Write new config
- [ ] Report migration results

### 6. Update Main Launcher

**File**: `/home/starbased/dev/projects/shine/cmd/shine/main.go`

**Changes**:
- [ ] Import `github.com/starbased-co/shine/pkg/prism`
- [ ] Replace `findComponentBinary()` with prism.Manager
- [ ] Initialize prism.Manager from config:
  ```go
  prismMgr := prism.NewManager(
      cfg.Core.PrismDirs,
      cfg.Core.AutoPath,
      prism.DiscoveryMode(cfg.Core.DiscoveryMode),
  )
  ```
- [ ] Create unified `launchPrism()` function
- [ ] Launch all enabled prisms from cfg.Prisms map
- [ ] Update status messages to use "prism" terminology
- [ ] Update error messages for better clarity

### 7. Write Tests

**File**: `/home/starbased/dev/projects/shine/pkg/prism/discovery_test.go`

**Test Cases**:
- [ ] `TestManager_FindPrism_InPATH`
- [ ] `TestManager_FindPrism_InPrismDir`
- [ ] `TestManager_FindPrism_RelativeToExecutable`
- [ ] `TestManager_FindPrism_NotFound`
- [ ] `TestManager_AugmentPATH`
- [ ] `TestManager_DiscoverAll`
- [ ] `TestExpandPaths_TildeExpansion`
- [ ] `TestExpandPaths_EnvVarExpansion`
- [ ] `TestIsExecutable`

**File**: `/home/starbased/dev/projects/shine/pkg/config/types_test.go`

**Test Cases**:
- [ ] `TestConfig_ParseWithCoreSection`
- [ ] `TestConfig_ParseWithPrisms`
- [ ] `TestConfig_Migration_FromOldFormat`
- [ ] `TestConfig_Migration_FromPluginsFormat`
- [ ] `TestPrismConfig_ToPanelConfig`

---

## Phase 2: Prism Development Tooling

### 1. Prism Template Generator

**File**: `/home/starbased/dev/projects/shine/cmd/shine/prism.go`

**Command**: `shine new-prism <name>`

**Tasks**:
- [ ] Create command to scaffold new prism
- [ ] Generate directory structure
- [ ] Create boilerplate main.go
- [ ] Create go.mod with dependencies
- [ ] Create README.md with development instructions
- [ ] Add build script

**Template Structure**:
```
~/.config/shine/prisms/shine-<name>/
├── main.go          # Boilerplate Bubble Tea app
├── go.mod           # Go module definition
├── go.sum           # Dependency checksums
├── README.md        # Prism documentation
└── build.sh         # Build script
```

### 2. Example Prisms

**Create 3 Example Prisms**:

**Prism 1**: Weather Widget
- [ ] File: `docs/examples/prism-weather/`
- [ ] Displays current weather from API
- [ ] Updates every 15 minutes
- [ ] Top-right corner placement

**Prism 2**: Spotify Now Playing
- [ ] File: `docs/examples/prism-spotify/`
- [ ] Shows currently playing track
- [ ] Updates in real-time via D-Bus
- [ ] Bottom center placement

**Prism 3**: System Monitor
- [ ] File: `docs/examples/prism-sysmon/`
- [ ] CPU, memory, disk usage
- [ ] Updates every second
- [ ] Compact top-left widget

### 3. Prism Development Guide

**File**: `/home/starbased/dev/projects/shine/docs/PRISM_DEVELOPMENT.md`

**Sections**:
- [ ] Understanding the Prism Concept
- [ ] Getting Started
- [ ] Prism Interface Requirements
- [ ] Bubble Tea Best Practices for Panels
- [ ] Building and Testing
- [ ] Configuration Integration
- [ ] Debugging Techniques
- [ ] Performance Optimization
- [ ] Security Considerations
- [ ] Publishing and Sharing

---

## Phase 3: Advanced Features

### 1. Prism Validation Framework

**File**: `/home/starbased/dev/projects/shine/pkg/prism/validator.go`

**Tasks**:
- [ ] Create Validator struct
- [ ] Implement ValidationLevel enum
- [ ] Implement validateBasic() - check executable
- [ ] Implement validateInterface() - run with --validate flag
- [ ] Implement validateSignature() - verify GPG signature (future)
- [ ] Integrate validator into Manager.FindPrism()

### 2. Manifest-Based Discovery

**File**: `/home/starbased/dev/projects/shine/pkg/prism/manifest.go`

**Manifest Format** (`prism.toml`):
```toml
[prism]
name = "weather"
version = "1.0.0"
binary = "shine-weather"

[metadata]
author = "User <user@example.com>"
description = "Weather widget for Shine"
license = "MIT"

[capabilities]
network = true
filesystem = false

[defaults]
edge = "top-right"
columns_pixels = 200
lines_pixels = 80
```

**Tasks**:
- [ ] Define manifest schema
- [ ] Implement manifest parser
- [ ] Add manifest discovery mode to Manager
- [ ] Validate prism requirements from manifest
- [ ] Merge manifest defaults with user config

### 3. Hot Reload

**File**: `/home/starbased/dev/projects/shine/pkg/prism/reload.go`

**Tasks**:
- [ ] Watch config file for changes
- [ ] Reload config on SIGHUP
- [ ] Restart changed prisms
- [ ] Maintain running prisms that didn't change

---

## Configuration Migration

### Update Example Config

**File**: `/home/starbased/dev/projects/shine/examples/shine.toml`

**Add New Sections**:
```toml
[core]
prism_dirs = [
    "/usr/lib/shine/prisms",
    "~/.config/shine/prisms",
    "~/.local/share/shine/prisms",
]
auto_path = true
discovery_mode = "convention"
validate_prisms = false

# ALL prisms configured uniformly
[prisms.bar]
enabled = true
edge = "top"
lines_pixels = 30

[prisms.clock]
enabled = true
edge = "top-right"
columns_pixels = 150

[prisms.weather]
enabled = true
name = "weather"
edge = "top-right"
columns_pixels = 200
lines_pixels = 80
```

---

## Documentation Updates

### 1. README.md

**Sections to Add**:
- [ ] Prism System Overview
- [ ] Understanding Prisms (the metaphor)
- [ ] Installing Prisms
- [ ] Creating Custom Prisms
- [ ] Prism Configuration

### 2. Create New Documentation

**New Files**:
- [ ] `docs/PRISM_SYSTEM_DESIGN.md` - Complete architecture (created)
- [ ] `docs/PRISM_DEVELOPMENT.md` - Developer guide
- [ ] `docs/PRISM_EXAMPLES.md` - Collection of example prisms
- [ ] `docs/PRISM_SECURITY.md` - Security best practices

### 3. Update Existing Documentation

**Files to Update**:
- [ ] All references to "plugin" → "prism"
- [ ] All references to "component" (when referring to widgets) → "prism"
- [ ] Configuration examples
- [ ] Command help text

---

## Testing Checklist

### Integration Testing

**Scenarios**:
- [ ] Launch Shine with no custom prisms (built-ins only)
- [ ] Launch Shine with one custom prism
- [ ] Launch Shine with multiple custom prisms
- [ ] Launch Shine with prism in custom directory
- [ ] Prism binary not found (error handling)
- [ ] Prism crashes on startup (error handling)
- [ ] Config with invalid prism section
- [ ] Backward compatibility with old `[bar]` format
- [ ] Backward compatibility with old `[plugins.*]` format
- [ ] Config migration command

### Manual Testing

**Steps**:
1. [ ] Create test prism in `~/.config/shine/prisms/`
2. [ ] Build test prism
3. [ ] Add prism to config
4. [ ] Launch Shine and verify prism loads
5. [ ] Test prism updates (rebuild and reload)
6. [ ] Test prism removal (disable in config)
7. [ ] Test mixed built-in and custom prism setup
8. [ ] Test config migration from old format

---

## Success Criteria

### Phase 1 Complete When

- [ ] Prism discovery works from config-specified directories
- [ ] PATH augmentation works correctly
- [ ] Built-in prisms still launch (backward compatibility)
- [ ] Custom prisms can be loaded from `~/.config/shine/prisms/`
- [ ] Old config format auto-migrates
- [ ] `shine config migrate` command works
- [ ] All unit tests pass
- [ ] Integration tests pass
- [ ] Documentation updated with prism terminology

### Phase 2 Complete When

- [ ] `shine new-prism` command works
- [ ] 3 example prisms created and tested
- [ ] Prism development guide published
- [ ] Users can create and load custom prisms

### Phase 3 Complete When

- [ ] Prism validation framework implemented
- [ ] Manifest-based discovery works
- [ ] Hot reload functional
- [ ] Advanced features documented

---

## File Changes Summary

**New Files**:
- `/home/starbased/dev/projects/shine/pkg/prism/discovery.go`
- `/home/starbased/dev/projects/shine/pkg/prism/discovery_test.go`
- `/home/starbased/dev/projects/shine/pkg/prism/manager.go`
- `/home/starbased/dev/projects/shine/pkg/prism/validator.go` (Phase 3)
- `/home/starbased/dev/projects/shine/pkg/prism/manifest.go` (Phase 3)
- `/home/starbased/dev/projects/shine/cmd/shine/config.go` (migration command)
- `/home/starbased/dev/projects/shine/docs/PRISM_SYSTEM_DESIGN.md` (created)
- `/home/starbased/dev/projects/shine/docs/PRISM_DEVELOPMENT.md` (Phase 2)
- `/home/starbased/dev/projects/shine/docs/examples/prism-weather/` (Phase 2)

**Modified Files**:
- `/home/starbased/dev/projects/shine/pkg/config/types.go` - Add CoreConfig, unify to PrismConfig
- `/home/starbased/dev/projects/shine/pkg/config/loader.go` - Handle new config structure + migration
- `/home/starbased/dev/projects/shine/cmd/shine/main.go` - Use prism.Manager
- `/home/starbased/dev/projects/shine/examples/shine.toml` - Add prism examples
- `/home/starbased/dev/projects/shine/README.md` - Document prism system

**Files to Rename**:
- `/home/starbased/dev/projects/shine/docs/examples/plugin-weather/` → `prism-weather/`

**No Changes Required**:
- `/home/starbased/dev/projects/shine/pkg/panel/` - Panel management unchanged
- `/home/starbased/dev/projects/shine/cmd/shine-bar/` - Prism implementations unchanged
- `/home/starbased/dev/projects/shine/cmd/shinectl/` - Control utility unchanged

---

## Quick Start (for Development)

### 1. Implement Core Discovery

```bash
cd /home/starbased/dev/projects/shine

# Create prism package
mkdir -p pkg/prism
touch pkg/prism/discovery.go
touch pkg/prism/discovery_test.go
touch pkg/prism/manager.go

# Edit discovery.go (see design doc section 2.6)
nvim pkg/prism/discovery.go
```

### 2. Update Configuration

```bash
# Edit types.go
nvim pkg/config/types.go

# Add CoreConfig struct
# Create unified PrismConfig
# Add Prisms map
```

### 3. Update Main Launcher

```bash
# Edit main.go
nvim cmd/shine/main.go

# Replace findComponentBinary with prism.Manager
# Add prism loading logic
```

### 4. Test

```bash
# Run tests
go test ./pkg/prism -v
go test ./pkg/config -v

# Build
go build -o bin/shine ./cmd/shine

# Test with example prism
mkdir -p ~/.config/shine/prisms
# ... create test prism ...
./bin/shine
```

---

## Terminology Migration Guide

### Old → New Mapping

| Old Term | New Term | Context |
|----------|----------|---------|
| Plugin | Prism | Custom/user widgets |
| Component | Prism | When referring to widgets (bar, clock, etc.) |
| Plugin Manager | Prism Manager | Discovery and lifecycle management |
| plugin_dirs | prism_dirs | Config setting |
| [plugins.*] | [prisms.*] | Config section |
| FindComponent() | FindPrism() | Discovery function |
| PluginConfig | PrismConfig | Configuration struct |
| ComponentConfig | PrismConfig | Unified config struct |

### Code Pattern Changes

**Old**:
```go
pluginMgr := component.NewPluginManager(...)
path, err := pluginMgr.FindComponent(name)
```

**New**:
```go
prismMgr := prism.NewManager(...)
path, err := prismMgr.FindPrism(name)
```

**Old Config**:
```toml
[plugins.weather]
enabled = true
```

**New Config**:
```toml
[prisms.weather]
enabled = true
```

---

## Questions to Resolve

- [ ] Should built-in prisms (bar, clock, etc.) be installed to `/usr/lib/shine/prisms`?
  - **Recommendation**: Yes, for unified treatment
- [ ] Should prism directories be recursive?
  - **Recommendation**: Non-recursive (simpler, faster)
- [ ] Should we support prism dependencies?
  - **Recommendation**: Not in Phase 1 (add in future)
- [ ] Should validation be opt-in or opt-out?
  - **Recommendation**: Opt-in initially (validate_prisms = false default)
- [ ] How long should we support old config format?
  - **Recommendation**: Auto-migration indefinitely, deprecation warnings for 2-3 releases
