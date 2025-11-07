# Phase 3: Advanced Features - Quick Start

**Status**: ✅ COMPLETED
**For**: Users and developers wanting to use Phase 3 features

## What's New in Phase 3?

Phase 3 adds four major production-ready features:

1. **Manifest-Based Discovery** - Structured metadata for prisms
2. **Hot Reload** - Automatic config reload on changes
3. **Enhanced Validation** - Comprehensive binary checking
4. **Lifecycle Management** - Centralized prism control

## Quick Start

### 1. Manifest-Based Discovery

Create a prism with a manifest file:

```bash
# Create prism directory
mkdir -p ~/.config/shine/prisms/weather

# Create manifest
cat > ~/.config/shine/prisms/weather/prism.toml <<EOF
[prism]
name = "weather"
version = "1.0.0"
binary = "shine-weather"

[metadata]
description = "Weather widget"
author = "Your Name"
license = "MIT"
EOF

# Copy your binary
cp /path/to/shine-weather ~/.config/shine/prisms/weather/
```

Enable in config:

```toml
# ~/.config/shine/shine.toml
[core]
discovery_mode = "auto"  # Try manifest first, fall back to convention

[prisms.weather]
enabled = true
edge = "top-right"
columns_pixels = 300
```

### 2. Hot Reload (Automatic)

Just edit your config and save - shine will detect changes automatically:

```bash
# Edit config
vim ~/.config/shine/shine.toml

# Change something, save - prisms reload automatically
# (1 second detection latency)
```

### 4. Lifecycle Management

For advanced use cases (custom launchers, monitoring tools):

```go
import (
    "github.com/starbased-co/shine/pkg/prism"
    "github.com/starbased-co/shine/pkg/panel"
    "github.com/starbased-co/shine/pkg/config"
)

// Setup
prismMgr := prism.NewManagerWithMode(dirs, true, prism.DiscoveryAuto)
panelMgr := panel.NewManager()
lifecycleMgr := prism.NewLifecycleManager(prismMgr, panelMgr)

// Launch
cfg := &config.PrismConfig{Name: "weather", Enabled: true, /* ... */}
lifecycleMgr.Launch("weather", cfg)

// Monitor
status, _ := lifecycleMgr.Health("weather")
fmt.Printf("Running: %v, Uptime: %v\n", status.Running, status.Uptime)

// Reload
lifecycleMgr.Reload("weather", updatedCfg)
```

## Discovery Modes

Choose how shine finds prisms:

| Mode         | Behavior                              | Use Case                          |
| ------------ | ------------------------------------- | --------------------------------- |
| `convention` | Use shine-\* naming only              | Fastest, simple setups            |
| `manifest`   | Require prism.toml files              | Structured, curated collections   |
| `auto`       | Try manifest, fall back to convention | Best of both worlds (recommended) |

Set in config:

```toml
[core]
discovery_mode = "auto"  # convention | manifest | auto
```

## Migration from Phase 1/2

**Good news: No changes required!**

- All Phase 3 features are opt-in
- Existing configs work as-is
- Convention-based discovery still default
- No breaking changes

To use new features:

1. Add `discovery_mode = "auto"` to `[core]` (optional)
2. Create manifest files for prisms you want structured (optional)
3. That's it!

## Examples

### Example 1: Simple Manifest Prism

```bash
# Directory structure
~/.config/shine/prisms/weather/
├── prism.toml          # Manifest
└── shine-weather       # Binary

# prism.toml
[prism]
name = "weather"
version = "1.0.0"
binary = "shine-weather"

[metadata]
description = "Weather widget"

# shine.toml
[core]
discovery_mode = "auto"

[prisms.weather]
enabled = true
edge = "top-right"
columns_pixels = 300
```

### Example 2: Mixed Setup

```bash
# Some prisms with manifests
~/.config/shine/prisms/
├── weather/
│   ├── prism.toml
│   └── shine-weather
└── custom/
    ├── prism.toml
    └── shine-custom

# Some without (convention-based)
~/.local/bin/
├── shine-bar
└── shine-clock

# Both work with discovery_mode = "auto"
```

### Example 3: Validation in Build Script

```bash
#!/bin/bash
# build-and-validate.sh

set -e

echo "Building prism..."
go build -o shine-myprism

echo "Validating binary..."
cat > validate.go <<EOF
package main
import (
    "fmt"
    "os"
    "github.com/starbased-co/shine/pkg/prism"
)
func main() {
    result, _ := prism.Validate("./shine-myprism")
    if !result.Valid {
        fmt.Printf("❌ Validation failed: %v\n", result.Errors)
        os.Exit(1)
    }
    fmt.Printf("✅ Valid binary\n")
    for _, w := range result.Warnings {
        fmt.Printf("⚠️  %s\n", w)
    }
    fmt.Printf("Capabilities: %v\n", result.Capabilities)
}
EOF

go run validate.go
rm validate.go

echo "Installing..."
mkdir -p ~/.config/shine/prisms/myprism
cp shine-myprism ~/.config/shine/prisms/myprism/
echo "✨ Done!"
```

## Troubleshooting

### Manifest not found

```
Error: prism weather not found via manifest discovery
```

**Solution**: Ensure directory name matches prism name:

```bash
~/.config/shine/prisms/weather/  # ← must match "weather"
├── prism.toml
```

### Validation warnings

```
Warning: Binary appears to be a script (shebang detected)
```

**Action**: This is informational. Scripts work fine, but native binaries are preferred for performance.

### Hot reload not working

**Check**:

1. Config file is writable
2. Using absolute path or proper expansion
3. Waiting at least 1 second after save

## Performance

All Phase 3 features have minimal overhead:

- **Manifest discovery**: One extra file read per prism directory
- **Hot reload**: 1Hz polling (one stat() per second)
- **Validation**: On-demand only (not automatic)
- **Lifecycle manager**: No background overhead

## Security

Phase 3 maintains security:

- ✅ Manifests validated before use
- ✅ Binary paths checked for traversal
- ✅ Executable permissions required
- ✅ User controls prism directories
- ⚠️ Validation is advisory (doesn't block execution)

## Learn More

- **Full Details**: See `/home/starbased/dev/projects/shine/docs/PHASE_3_SUMMARY.md`
- **Architecture**: See `/home/starbased/dev/projects/shine/docs/PRISM_SYSTEM_DESIGN.md` Section 12
- **Examples**: See `/home/starbased/dev/projects/shine/examples/prisms-manifest/`

## Support

For issues or questions:

1. Check this README
2. See Phase 3 Summary document
3. Review test files for API examples
4. Open GitHub issue

## Credits

Implemented as part of the Shine prism system Phase 3.
All features designed for backward compatibility and ease of use.
