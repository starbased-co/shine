# Shine Panel Configuration Guide

This guide explains the panel configuration system based on the current implementation in `pkg/`.

## Overview

Shine uses a unified configuration system where prisms (widgets) are configured in `~/.config/shine/shine.toml`. The configuration uses a two-field size system (`width`/`height`) and origin-based positioning.

## Configuration Structure

### Core Configuration

The `[core]` section configures global Shine settings:

```toml
[core]
# Directories to search for prism binaries and configurations
# Can be a single string or array of strings
path = [
    "~/.local/share/shine/bin",
    "~/.config/shine/bin",
    "~/.config/shine/prisms",
    "/usr/lib/shine/bin",
]
```

### Prism Configuration

Prisms are configured under `[prisms.*]` sections:

```toml
[prisms.mywidget]
enabled = true
origin = "top-center"
width = 80
height = "30px"
position = "0,0"
output_name = "DP-2"
focus_policy = "not-allowed"
hide_on_focus_loss = false
```

## Configuration Fields Reference

### Core Identification

#### `name` (string)
**Required in prism.toml, optional in shine.toml**

Prism identifier. In shine.toml, derived from section name (e.g., `[prisms.weather]` → name is "weather").

#### `version` (string)
**Optional, primarily for prism.toml**

Semantic version string (e.g., "1.0.0").

#### `path` (string)
**Optional**

Custom binary name or path. If empty, defaults to `shine-{name}`.

Examples:
- `path = "shine-weather"` - Binary name to find in PATH
- `path = "/usr/bin/shine-weather"` - Absolute path

### Runtime State

#### `enabled` (bool)
**Default: false**

Controls whether this prism should be launched by Shine.

```toml
enabled = true  # Launch this prism
```

### Positioning & Layout

#### `origin` (string)
**Default: "center"**

Specifies the anchor point on screen for positioning.

**Valid values:**
- `"top-left"` - Top-left corner
- `"top-center"` - Top edge, centered horizontally
- `"top-right"` - Top-right corner
- `"left-center"` - Left edge, centered vertically
- `"center"` - Screen center
- `"right-center"` - Right edge, centered vertically
- `"bottom-left"` - Bottom-left corner
- `"bottom-center"` - Bottom edge, centered horizontally
- `"bottom-right"` - Bottom-right corner

```toml
origin = "top-center"
```

#### `position` (string)
**Format: "x,y"**
**Default: "0,0"**

Offset from the origin point in pixels. Both x and y must be integers.

```toml
position = "100,50"  # 100px horizontal, 50px vertical offset from origin
```

**Coordinate behavior by origin:**

| Origin | X Direction | Y Direction |
|--------|-------------|-------------|
| `top-left` | Right | Down |
| `top-center` | Horizontal offset | Down |
| `top-right` | Left (from right edge) | Down |
| `left-center` | Right | Vertical offset |
| `center` | Horizontal offset | Vertical offset |
| `right-center` | Left (from right edge) | Vertical offset |
| `bottom-left` | Right | Up (from bottom edge) |
| `bottom-center` | Horizontal offset | Up (from bottom edge) |
| `bottom-right` | Left (from right edge) | Up (from bottom edge) |

#### `width` (int or string)
**Default: 1**

Panel width. Can be specified as:
- **Integer**: Terminal columns (e.g., `80`)
- **String with "px"**: Pixels (e.g., `"1200px"`)

```toml
width = 80           # 80 terminal columns
width = "1200px"     # 1200 pixels
```

#### `height` (int or string)
**Default: 1**

Panel height. Can be specified as:
- **Integer**: Terminal lines (e.g., `24`)
- **String with "px"**: Pixels (e.g., `"600px"`)

```toml
height = 24          # 24 terminal lines
height = "600px"     # 600 pixels
```

### Behavior

#### `hide_on_focus_loss` (bool)
**Default: false**

Hide panel when it loses keyboard focus.

```toml
hide_on_focus_loss = true
```

**Note:** When enabled, `focus_policy` is automatically set to `"on-demand"`.

#### `focus_policy` (string)
**Default: "not-allowed"**

Controls keyboard focus behavior.

**Valid values:**
- `"not-allowed"` - Panel never receives keyboard focus (status displays)
- `"on-demand"` - Panel can receive focus when clicked (interactive widgets)
- `"exclusive"` - Panel always has focus when visible (rarely used)

```toml
focus_policy = "on-demand"
```

#### `output_name` (string)
**Default: "DP-2"**

Target monitor name for panel placement.

```toml
output_name = "DP-2"
```

**CRITICAL:** Always use DP-2. Never use DP-1 as it will cause system failure.

Query available monitors:
```bash
hyprctl monitors -j
```

### Metadata

#### `metadata` (map)
**Optional, only meaningful in prism sources**

Custom key-value pairs for prism-specific configuration. Used in `prism.toml` or standalone `.toml` files.

```toml
[metadata]
api_key = "your-api-key"
location = "San Francisco"
refresh_interval = 300
```

**Important:** Metadata in shine.toml `[prisms.*]` sections is ignored. Only metadata from prism sources (prism.toml, standalone .toml) is preserved during configuration merge.

## Complete Configuration Examples

### Status Bar (Full Width Top)

```toml
[prisms.bar]
enabled = true
origin = "top-center"
height = "30px"
width = "1920px"
position = "0,0"
output_name = "DP-2"
focus_policy = "not-allowed"
```

### Chat Panel (Bottom Center)

```toml
[prisms.chat]
enabled = true
origin = "bottom-center"
height = 10
width = 80
position = "0,10"
output_name = "DP-2"
focus_policy = "on-demand"
hide_on_focus_loss = true
```

### Clock Widget (Top-Right Corner)

```toml
[prisms.clock]
enabled = true
origin = "top-right"
width = "150px"
height = "30px"
position = "0,0"
output_name = "DP-2"
focus_policy = "not-allowed"
```

### System Info (Custom Position)

```toml
[prisms.sysinfo]
enabled = true
origin = "top-left"
width = "200px"
height = "100px"
position = "10,40"
output_name = "DP-2"
focus_policy = "not-allowed"
```

### Spotify Player (Bottom with Offset)

```toml
[prisms.spotify]
enabled = true
origin = "bottom-center"
width = "600px"
height = "120px"
position = "0,10"
output_name = "DP-2"
focus_policy = "on-demand"
hide_on_focus_loss = false
```

## How Configuration Works

### 1. Configuration Loading

Shine loads `~/.config/shine/shine.toml` and discovers prisms from configured paths.

```go
// From pkg/config/loader.go
cfg, err := config.Load("~/.config/shine/shine.toml")
```

### 2. Prism Discovery

The system searches configured paths for three types of prism configurations:

**Type 1: Full Package** (directory with prism.toml and binary)
```
~/.config/shine/prisms/weather/
├── prism.toml
└── shine-weather
```

**Type 2: Data Directory** (directory with prism.toml, binary in PATH)
```
~/.config/shine/prisms/weather/
├── prism.toml
└── config.json

# Binary: ~/.local/bin/shine-weather
```

**Type 3: Standalone Configuration** (standalone .toml file, binary in PATH)
```
~/.config/shine/prisms/weather.toml

# Binary: ~/.local/bin/shine-weather
```

### 3. Configuration Merge

If a prism is discovered AND has configuration in shine.toml, the configs are merged:

```go
// From pkg/config/discovery.go
merged = MergePrismConfigs(discoveredPrism.Config, userConfig)
```

**Merge priority:**
- User settings in shine.toml override prism defaults
- Metadata ALWAYS comes from prism source (never from shine.toml)
- `enabled` field is OR'ed (true in either enables the prism)

### 4. Panel Creation

PrismConfig is converted to panel.Config:

```go
// From pkg/config/types.go
panelCfg := prismConfig.ToPanelConfig()
```

This handles:
- Origin parsing
- Dimension parsing (int vs "px" strings)
- Position parsing
- Focus policy mapping
- Default values

### 5. Margin Calculation

Margins are calculated automatically from origin, position, and panel size:

```go
// From pkg/panel/config.go
top, left, bottom, right, err := config.calculateMargins()
```

**Users don't set margins directly.** The system computes them based on:
- Monitor resolution (via hyprctl)
- Panel dimensions
- Origin point
- Position offset

### 6. Kitty Remote Control Launch

Finally, configuration is translated to Kitty remote control arguments:

```go
// From pkg/panel/config.go
args := config.ToRemoteControlArgs(binaryPath)
```

Generates:
```bash
kitty @ launch --type=os-panel \
    --os-panel edge=top \
    --os-panel columns=1200px \
    --os-panel lines=30px \
    --os-panel margin-left=360 \
    --os-panel margin-top=0 \
    --os-panel focus-policy=not-allowed \
    --os-panel output-name=DP-2 \
    --title shine-bar \
    /path/to/shine-bar
```

## Migration from Old Format

If you have old configurations, update them as follows:

### Field Renames

| Old Field | New Field | Notes |
|-----------|-----------|-------|
| `edge` | `origin` | Values changed (see origin docs) |
| `lines` | `height` | Now supports "px" suffix |
| `columns` | `width` | Now supports "px" suffix |
| `lines_pixels` | `height = "Npx"` | Merged into height field |
| `columns_pixels` | `width = "Npx"` | Merged into width field |
| `margin_*` | (removed) | Margins calculated automatically |

### Old Format Example

```toml
[bar]
enabled = true
edge = "top"
lines_pixels = 30
columns_pixels = 1920
margin_top = 0
```

### New Format

```toml
[prisms.bar]
enabled = true
origin = "top-center"
height = "30px"
width = "1920px"
position = "0,0"
```

## Troubleshooting

### Panel Not Appearing

1. **Check enabled status:**
   ```toml
   enabled = true
   ```

2. **Verify output name:**
   ```toml
   output_name = "DP-2"
   ```

3. **Check binary exists:**
   ```bash
   which shine-myprism
   ```

4. **Test configuration loading:**
   ```bash
   shine  # Check console output for errors
   ```

### Wrong Position

1. **Verify origin matches intent:**
   - Use semantic origins (top-left, bottom-center) for clarity

2. **Check position format:**
   ```toml
   position = "100,50"  # Correct: integers, comma-separated
   position = "100px,50px"  # WRONG: no px suffix allowed
   ```

3. **Understand coordinate system for your origin:**
   - Review coordinate behavior table above

### Size Issues

1. **Confirm pixel values have "px" suffix:**
   ```toml
   width = "600px"  # Correct
   width = 600px    # WRONG: needs quotes
   ```

2. **Check monitor resolution:**
   ```bash
   hyprctl monitors -j | grep -A5 '"name": "DP-2"'
   ```

3. **Verify dimensions are positive:**
   ```toml
   width = 100   # Correct
   width = 0     # WRONG: will use default (1)
   ```

## Advanced Topics

### Dynamic Origin Selection

Use different origins for different panel types:

```toml
# Full-width bar
[prisms.bar]
origin = "top-center"
width = "1920px"

# Corner widget
[prisms.clock]
origin = "top-right"
width = "200px"

# Floating panel
[prisms.chat]
origin = "center"
width = 80
height = 20
```

### Multi-Monitor Setup

Target specific monitors by name:

```toml
[prisms.primary-bar]
enabled = true
origin = "top-center"
output_name = "DP-2"

[prisms.secondary-bar]
enabled = true
origin = "top-center"
output_name = "HDMI-A-1"
```

### Prism Discovery Paths

Configure custom search paths:

```toml
[core]
path = [
    "~/my-prisms",           # Custom directory
    "~/.local/share/shine/bin",
    "/usr/lib/shine/bin",
]
```

### Conditional Enabling

Enable prisms based on environment or use case:

```toml
# Development setup
[prisms.debug-panel]
enabled = true

# Production setup
# [prisms.debug-panel]
# enabled = false
```

## Best Practices

1. **Use pixels for precise sizing:**
   ```toml
   width = "600px"  # Preferred
   width = 60       # Only for relative sizing
   ```

2. **Choose semantic origins:**
   ```toml
   origin = "top-right"    # Clear intent
   origin = "center"       # Less clear without context
   ```

3. **Always specify DP-2:**
   ```toml
   output_name = "DP-2"  # Explicit is better
   ```

4. **Test positioning incrementally:**
   ```toml
   position = "0,0"    # Start here
   position = "10,0"   # Adjust as needed
   ```

5. **Use appropriate focus policies:**
   ```toml
   focus_policy = "not-allowed"  # Status displays
   focus_policy = "on-demand"    # Interactive widgets
   ```

## See Also

- [Prism Developer Guide](PRISM_DEVELOPER_GUIDE.md) - Creating custom prisms
- [README.md](../README.md) - General Shine documentation
- [pkg/config/types.go](../pkg/config/types.go) - Configuration structure source
- [pkg/panel/config.go](../pkg/panel/config.go) - Panel configuration implementation
