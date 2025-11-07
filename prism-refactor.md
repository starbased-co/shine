# Prism Configuration Refactoring - Implementation Summary

## Overview
Implemented dual-file infrastructure with unified data structure for prism configuration. Prisms can now be organized in **four flexible ways** while maintaining the same configuration schema.

## Key Changes

### 1. Unified PrismConfig Structure (pkg/config/types.go)
- Added `Version` field for semantic versioning
- Added `Metadata` map for prism identification (description, author, license, etc.)
- Added `ResolvedPath` internal field for binary location after discovery
- Documented that metadata is ONLY valid in prism sources, not shine.toml
- Consistent structure used in both shine.toml and prism.toml

### 2. Four-Tier Configuration System (pkg/config/discovery.go)
New system supports four prism organization methods:

**Type 1: Inline Configuration** (no separate file)
```toml
# ~/.config/shine/shine.toml
[prisms.bar]
enabled = true
anchor = "top"
height = "30px"
# No discovery, no separate files, binary via PATH
```

**Type 2: Full Package** (directory with binary)
```
~/.config/shine/prisms/weather/
├── prism.toml          # Config + metadata
├── shine-weather       # Binary included
└── assets/             # Optional resources
```

**Type 3: Data Directory** (directory without binary)
```
~/.config/shine/prisms/spotify/
├── prism.toml          # Config + metadata
└── config.json         # Prism-specific data
# Binary found via PATH lookup
```

**Type 4: Standalone Config** (single .toml file)
```
~/.config/shine/prisms/clock.toml  # Just config file
# Binary found via PATH lookup
```

### 3. Configuration Merge Strategy
- Prism sources (prism.toml, standalone .toml) provide defaults
- User config in shine.toml overrides user-configurable settings
- Metadata ALWAYS comes from prism source during merge
- Any metadata in shine.toml [prisms.*] is simply ignored during merge

### 4. Updated Core Configuration
- Added `~/.config/shine/prisms` to default search paths
- Prisms directory is checked first for discovery
- Falls back to binary-only locations

### 5. Test Coverage
Comprehensive tests added in discovery_test.go:
- TestDiscoverDirectoryPrism (Type 2 & 3: directories)
- TestDiscoverStandalonePrism (Type 4: standalone .toml)
- TestDiscoverPrisms (all discovered types: 2, 3, 4)
- TestMergePrismConfigs (merge logic with metadata preservation)
- TestMetadataFromPrismSourceTakesPriority (metadata always from prism source)
- All existing tests updated and passing

## Benefits

✅ **Consistency**: Same data structure everywhere
✅ **Flexibility**: Choose organization level per prism
✅ **Self-Contained**: Prisms can ship with defaults and assets
✅ **User Control**: Easy overrides without touching prism files
✅ **Portability**: Prism directories are self-describing
✅ **Discovery**: Auto-discover prisms in multiple formats
✅ **Simplicity**: No metadata merge complexity

## Usage Examples

### Inline Configuration (Simple)
```toml
# ~/.config/shine/shine.toml
[prisms.clock]
enabled = true
anchor = "top-right"
width = "150px"
height = "30px"
```

### Full Package (Complex, Distributable)
```toml
# ~/.config/shine/prisms/weather/prism.toml
name = "weather"
version = "1.0.0"
path = "shine-weather"
enabled = true
anchor = "top-right"
width = "400px"
height = "30px"

[metadata]
description = "Weather widget"
author = "Your Name"
license = "MIT"
```

### User Override
```toml
# ~/.config/shine/shine.toml
[prisms.weather]
enabled = true          # Enable it
width = "500px"         # Override: wider
margin_right = 200      # Override: reposition
# All other settings from weather/prism.toml
```

## File Changes
- pkg/config/types.go - Enhanced PrismConfig structure
- pkg/config/discovery.go - NEW: Four-tier configuration system (Types 2-4 discovery)
- pkg/config/loader.go - Updated to use new discovery + handle Type 1 inline
- pkg/config/discovery_test.go - NEW: Comprehensive tests for Types 2-4
- pkg/config/loader_test.go - Updated tests for Type 1 + integration
- examples/shine.toml - Updated with four-tier structure docs
- examples/prism.toml - Updated to show unified structure (Types 2-4)

## Four Configuration Types Summary

| Type | Location | Binary | Metadata | Use Case |
|------|----------|--------|----------|----------|
| **1. Inline** | `shine.toml [prisms.*]` | PATH | ❌ No | Simple, quick configs |
| **2. Full Package** | `prisms/name/prism.toml` | Bundled | ✅ Yes | Distributable, self-contained |
| **3. Data Directory** | `prisms/name/prism.toml` | PATH | ✅ Yes | Config + data files, external binary |
| **4. Standalone** | `prisms/name.toml` | PATH | ✅ Yes | Single-file organization |

**Key Points:**
- All 4 types use the same `PrismConfig` structure
- Types 2-4 are discovered automatically from prism directories
- Type 1 requires manual configuration in shine.toml
- Metadata is only meaningful in Types 2-4 (prism sources)
- During merge, metadata always comes from prism source
- User overrides in shine.toml work for all types

## Migration Notes
No migration needed - project is unreleased. All backward compatibility code removed.
