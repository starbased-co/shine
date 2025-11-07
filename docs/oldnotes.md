# `shine` Notes

```toml
[core]
prism_dirs = [
    "/usr/lib/shine/prisms",      # Built-in prisms (shipped with Shine)
    "~/.config/shine/prisms",      # User prisms
]
# tree /usr/lib/shine/prisms

# All widgets are prisms - no distinction between built-in and user-provided
[prisms.bar]
enabled = true
edge = "top"

[prisms.chat]
enabled = true

[prisms.weather]
enabled = true

```

```go
// pkg/prism/prism.go
type Prism struct {
    Name     string          // "bar", "spotify"
    Binary   string          // Abs path to executable
    Config   *PrismConfig    // TOML config
    Process  *exec.Cmd       // Command Process
    WindowID string          // Hyprland window ID
}

// pkg/prism/manager.go
type PrismManager struct {
    prismDirs []string                 // Search paths
    registry  map[string]*Prism        // Loaded prisms
    // ... discovery and lifecycle logic
}
```

```text
┌──────────────────┐
│   shine.toml     │  1. TOML file on disk
│  [core]          │
│  [prisms.bar]    │
│  [prisms.chat]   │
└────────┬─────────┘
         │ config.Load() + toml.Unmarshal()
         ▼
┌──────────────────┐
│  Config struct   │  2. Parsed into Go structs
│  ├─ Core         │
│  └─ Prisms       │
│     map[string]  │
│     *PrismConfig │
└────────┬─────────┘
         │ prism.Manager.LoadFromConfig()
         ▼
┌──────────────────┐
│  Prism struct    │  3. Created by PrismManager
│  ├─ Name         │  ← From map key
│  ├─ Binary       │  ← Found by discovery
│  ├─ Config       │  ← From PrismConfig
│  ├─ Process      │  ← Set at launch
│  └─ WindowID     │  ← Set after launch
└────────┬─────────┘
         │ prism.Launch()
         ▼
┌──────────────────┐
│   Prism $PID     │  4. Executable running
└──────────────────┘

┌─────────────────────────────────────────┐
│  Application Layer                      │  Your TUI Binary
│  ├─ Bubble Tea (TUI framework)          │  Your Go code
│  └─ Lipgloss (styling)                  │
├─────────────────────────────────────────┤
│  Terminal Emulator Layer                │
│  └─ Kitty                               │  ANSI → Pixels
├─────────────────────────────────────────┤
│  Wayland Protocol Layer                 │
│  ├─ wl_surface (base surface)           │
│  ├─ wl_buffer (pixel data)              │
│  └─ zwlr_layer_shell_v1 (positioning)   │  IPC with compositor
├─────────────────────────────────────────┤
│  Compositor Layer                       │
│  └─ Hyprland                            │  Window management
├─────────────────────────────────────────┤
│  Graphics Layer                         │
│  └─ Mesa/DRM                            │  GPU
├─────────────────────────────────────────┤
│  Monitor                                │
│  └─ DP-2                                │
└─────────────────────────────────────────┘
```
