# Configuration System

```
~/.config/shine/
├── shine.toml              # Main config (type 1: inline prisms in [prisms.*])
└── prisms/                 # Prism discovery directories
    ├── clock/              # type 2: Directory with prism.toml
    │   ├── prism.toml      #   prism definition file
    │   └── shine-clock     #   Optional bundled binary
    └── weather.toml        # Type 3: Standalone TOML file
```

## Prism Discovery Paths

The `core.path` setting specifies where to search for prisms

```toml
[core]
path = ["~/.config/shine/prisms"]
```

Default: `~/.config/shine/prisms`

**Note:** Binary resolution uses the system PATH (via `exec.LookPath`), not these directories. The only exception is Type 2 prisms where a binary can be bundled alongside the prism.toml in the same directory.

## Loading Sequence

The configuration system loads in this order:

1. **Load shine.toml**
   - Parsed by `config.Load(path)` via `loadShineConfig()`
   - Path defaults to `~/.config/shine/shine.toml`

2. **Discover Prisms**
   - Scans directories specified in `core.path`
   - **Directories**: Looks for `prism.toml` inside
   - **Files**: Picks up standalone `*.toml` files (not named `prism.toml`)
   - Returns `DiscoveredPrism` with metadata about source

3. **Merge Configurations**
   - Prism source (prism.toml) provides defaults
   - User config (shine.toml `[prisms.*]`) overrides specific fields
   - Merged result becomes the active configuration

## Configuration Structures

### Core Configuration

Located in `pkg/config/types.go`:

```go
type Config struct {
    Core   *CoreConfig             `toml:"core"`
    Prisms map[string]*PrismConfig `toml:"prisms"`
}

type CoreConfig struct {
    Path interface{} `toml:"path"`  // Single string or []string
}
```

The `Path` field is flexible and accepts either:

- Single string: `path = "~/.config/shine/prisms"`
- Array: `path = ["~/.local/bin", "~/.config/shine/prisms"]`

### Prism Configuration

Located in `pkg/config/types.go`:

```go
type PrismConfig struct {
    // Core Identification
    Name    string `toml:"name"`
    Version string `toml:"version,omitempty"`
    Path    string `toml:"path,omitempty"`      // Binary path/name

    // Runtime State
    Enabled bool `toml:"enabled"`

    // Positioning & Layout
    Origin   string      `toml:"origin,omitempty"`   // top-left, top-center, top-right, etc.
    Position string      `toml:"position,omitempty"` // "x,y" offset from origin
    Width    interface{} `toml:"width,omitempty"`    // int or string "100px"/"50%"
    Height   interface{} `toml:"height,omitempty"`   // int or string "100px"/"50%"

    // Behavior
    HideOnFocusLoss bool   `toml:"hide_on_focus_loss,omitempty"`
    FocusPolicy     string `toml:"focus_policy,omitempty"`
    OutputName      string `toml:"output_name,omitempty"`

    // Metadata (optional)
    Metadata map[string]interface{} `toml:"metadata,omitempty"`

    // Internal field (not from TOML)
    ResolvedPath string `toml:"-"`  // Set during discovery
}
```

### Extended Prism Entry (shined-specific)

Located in `cmd/shined/config.go`:

```go
type PrismEntry struct {
    *config.PrismConfig

    // Restart Policies
    Restart      string `toml:"restart"`       // no|on-failure|unless-stopped|always
    RestartDelay string `toml:"restart_delay"` // Duration: "5s", "500ms"
    MaxRestarts  int    `toml:"max_restarts"`  // Per hour, 0 = unlimited
}
```

## Configuration Examples

### shine.toml

The main configuration file defines enabled prisms and overrides defaults:

```toml
[core]
path = ["~/.config/shine/prisms", "~/.local/bin"]

[prisms.clock]
name = "shine-clock"
enabled = true
origin = "top-right"
width = "200px"
height = "50px"

[prisms.bar]
name = "shine-bar"
enabled = true
origin = "bottom"
width = "100%"
height = "32px"

[prisms.weather]
# Inherit defaults from prism.toml, only override what's needed
origin = "top-left"
enabled = true
```

### prism.toml (in a prism directory)

The manifest file defines defaults for a prism:

```toml
name = "shine-weather"
version = "1.0.0"
enabled = false
origin = "top-right"
width = "200px"
height = "100px"
path = "shine-weather"  # Optional: custom binary name

[metadata]
description = "Weather widget with forecasts"
author = "your-name"
license = "MIT"
repository = "https://github.com/your-org/shine-weather"
```

### Standalone TOML (in prisms/ directory)

A simpler alternative without a prism directory:

```toml
# ~/.config/shine/prisms/weather.toml
name = "shine-weather"
version = "1.0.0"
enabled = false
origin = "top-right"
width = "200px"
height = "100px"

[metadata]
description = "Weather widget"
```

## Prism Source Types

Shine supports three types of prism sources:

### Type 1: Inline in shine.toml

Prisms defined directly in the `[prisms.*]` sections of shine.toml:

```toml
# ~/.config/shine/shine.toml
[prisms.my-widget]
name = "shine-my-widget"
enabled = true
origin = "top-right"
width = "200px"
height = "100px"
```

Binary resolution:

- Searches system PATH for `path` field value or default `shine-{name}`

This type requires no separate prism.toml file - all configuration lives in shine.toml.

### Type 2: Directory with prism.toml

A directory containing a `prism.toml` manifest, optionally with a bundled binary:

```
~/.config/shine/prisms/weather/
├── prism.toml      # Configuration manifest (required)
├── shine-weather   # Bundled binary (optional)
└── data/           # Optional: supporting data files
```

Binary resolution order:

1. Checks for binary in the prism directory (using `path` field or default `shine-{name}`)
2. If not found locally, searches system PATH

### Type 3: Standalone TOML

A single `.toml` file in a search directory (not named `prism.toml`):

```
~/.config/shine/prisms/weather.toml   # All config in one file
```

Binary resolution:

- Always searches system PATH (no local directory to check)

## Runtime Changes

### Hot-Reload (SIGHUP)

Send SIGHUP to shined to reload configuration:

```bash
pkill -HUP shined
```

1. Reloads shine.toml
2. Rediscovers prisms
3. Spawns new prisms
4. Stops removed prisms
5. Preserves existing prism state (doesn't restart)
