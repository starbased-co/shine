# Shine Project - Phase 2 Handoff

**Date**: 2025-11-01
**Phase Completed**: Phase 1 - Prototype
**Commit**: `f5f5839` - feat: implement shine - Hyprland TUI desktop shell toolkit

---

## Project Overview

**Shine** is a Hyprland Wayland Layer Shell TUI Desktop Shell Toolkit for building desktop components (panels, docks, widgets) using Bubble Tea TUI framework running inside Kitty panels.

**Core Concept**: Instead of reimplementing Wayland layer shell in Go, we leverage Kitty's battle-tested `kitten panel` command which provides GPU-accelerated rendering and layer shell integration. Shine provides a clean Go API for configuring and managing these panels.

---

## Phase 1 Accomplishments ✅

### Working Features

1. **Panel Configuration System**
   - Ported LayerShellConfig from Kitty's Python implementation
   - Type-safe Go structs with TOML serialization
   - Supports all layer shell features: layers, edges, focus policies, margins, exclusive zones

2. **Corner Positioning**
   - Custom corner modes: `top-left`, `top-right`, `bottom-left`, `bottom-right`
   - Dynamic margin calculation based on monitor resolution
   - Queries Hyprland via `hyprctl monitors -j` to calculate positioning
   - Tested on 2560×1440 (DP-2)

3. **Component Architecture**
   - `cmd/shine`: Main launcher with binary path resolution
   - `cmd/shine-chat`: Bubble Tea chat widget (example component)
   - `cmd/shinectl`: Remote control utility (toggle, show, hide)
   - `pkg/panel`: Configuration, manager, remote control client
   - `pkg/config`: TOML configuration loading

4. **Widget Behavior**
   - Ctrl+C protection in widgets (persistent desktop components)
   - Single instance mode with toggle visibility
   - Remote control via Unix sockets
   - Configurable focus policies

5. **Testing**
   - 11 unit + integration tests passing
   - Test coverage for config parsing, margin calculation, kitten args generation

### Verified Runtime Behavior

- ✅ 400×300px chat panel in top-right corner of DP-2
- ✅ Stays visible (not hide-on-focus-loss for persistent widgets)
- ✅ Ctrl+C ignored in widget (doesn't accidentally close)
- ✅ Remote control socket works at `/tmp/shine-chat.sock`
- ✅ Single instance mode prevents duplicates
- ✅ Dynamic positioning adapts to monitor resolution

---

## Architecture Decisions

### Why Kitty Panel?

**Decision**: Use `kitten panel` instead of implementing Wayland layer shell in Go.

**Rationale**:
- Kitty's GPU-accelerated text rendering is exceptional
- Layer shell implementation is complex and well-tested in Kitty
- Saves months of development time
- Cross-compositor compatibility (Sway, Hyprland, etc.)

**Trade-off**: Dependency on Kitty being installed.

### Corner Positioning Strategy

**Problem**: Kitty panel's `edge` parameter anchors to edges (e.g., `edge=top` spans full width).

**Solution**:
- For corners, use `edge=top` or `edge=bottom` (based on vertical position)
- Calculate `margin-left` dynamically: `monitor_width - panel_width - margin_right`
- Query Hyprland for monitor resolution at launch time

**Code**: `pkg/panel/config.go:201-281` (getMonitorResolution + ToKittenArgs)

### Remote Control Protocol

**Implementation**: Use `-o listen_on=unix:/tmp/shine-{component}.sock` (NOT `--listen-on`)

**Note**: Kitty expects `listen_on` as a configuration option via `-o`, not a CLI flag. This caused initial confusion (see commit fixing this).

---

## Current Configuration

### Example: Chat Widget in Top-Right

```toml
# ~/.config/shine/shine.toml
[chat]
enabled = true
edge = "top-right"              # Custom corner mode
lines_pixels = 300              # Height in pixels
columns_pixels = 400            # Width in pixels
margin_top = 10
margin_right = 10
single_instance = true
hide_on_focus_loss = false      # Persistent widget
focus_policy = "on-demand"
output_name = "DP-2"            # Target monitor
```

### Usage

```bash
# Launch all enabled panels
./bin/shine

# Toggle chat visibility
./bin/shinectl toggle chat

# Stop everything
killall shine
```

---

## Known Limitations & Issues

### 1. Monitor Resolution Detection

**Current**: Queries Hyprland at launch time only
**Issue**: If monitor resolution changes (hotplug, config change), margins become incorrect
**Solution**: Add monitor change detection via Hyprland IPC or periodic re-query

### 2. Kitty Dependency

**Current**: Hard requirement on Kitty being installed
**Issue**: Users without Kitty can't use shine
**Solution**: Document requirement clearly, consider fallback detection

### 3. Corner Positioning Accuracy

**Current**: Works but relies on calculated margins
**Issue**: Slight positioning drift if Kitty's interpretation differs
**Better**: Native layer shell anchor support (would require Kitty upstream changes)

### 4. Remote Control Limited

**Current**: Only toggle visibility implemented
**Missing**: Show, hide, reload config, query state
**Files**: `cmd/shinectl/main.go:35-47`, `pkg/panel/remote.go:23-42`

### 5. No Component Hot Reload

**Current**: Must restart `shine` to reload config changes
**Desired**: `shinectl reload` should re-read TOML and update panels

---

## Code Structure

```
shine/
├── cmd/
│   ├── shine/main.go           # Launcher (95 lines)
│   │   └── findComponentBinary() - Locates component binaries
│   ├── shine-chat/main.go      # Example Bubble Tea component (124 lines)
│   └── shinectl/main.go        # Remote control CLI (73 lines)
│
├── pkg/
│   ├── panel/
│   │   ├── config.go           # LayerShellConfig + ToKittenArgs (354 lines)
│   │   │   ├── Edge enums (top, bottom, corners)
│   │   │   ├── getMonitorResolution() - Hyprland query
│   │   │   └── ToKittenArgs() - Converts to kitten panel CLI
│   │   ├── manager.go          # Panel lifecycle management (129 lines)
│   │   └── remote.go           # Remote control client (102 lines)
│   │
│   └── config/
│       ├── types.go            # TOML config structures (88 lines)
│       └── loader.go           # Config loading/saving (86 lines)
│
├── examples/
│   └── shine.toml              # Example configuration
│
├── docs/
│   └── llms/man/kitty.md       # Kitty manual reference (32k lines)
│
├── PLAN.md                     # Original implementation plan
├── IMPLEMENTATION.md           # Implementation report
└── README.md                   # User documentation
```

---

## Phase 2 Roadmap

### Priority 1: Core Features

#### 1.1 Additional Components

**Goal**: Build more desktop components beyond chat example.

**Candidates**:
- Status bar (workspace indicator, clock, system stats)
- Dock (application launcher with icons)
- Notification center
- Media player controls
- Calendar/agenda widget

**Implementation**:
- Each component is a standalone Bubble Tea app in `cmd/shine-{name}/`
- Add corresponding config section in TOML
- Register in `cmd/shine/main.go`

**Example**:
```go
// cmd/shine-bar/main.go
package main

import (
    tea "github.com/charmbracelet/bubbletea"
    "github.com/charmbracelet/lipgloss"
)

type model struct {
    width int
    time  string
}

func (m model) View() string {
    return lipgloss.JoinHorizontal(
        lipgloss.Top,
        renderWorkspaces(),
        renderSpacer(m.width),
        renderClock(m.time),
    )
}
```

#### 1.2 Complete Remote Control

**Current**: Only `toggle` implemented
**Add**: `show`, `hide`, `reload`, `list`, `query`

**Files to modify**:
- `pkg/panel/remote.go` - Add methods
- `cmd/shinectl/main.go` - Add commands

**Example**:
```go
// pkg/panel/remote.go
func (rc *RemoteControl) Show() error {
    return rc.execute(map[string]interface{}{
        "cmd": "resize-os-window",
        "action": "show",
    })
}

func (rc *RemoteControl) QueryState() (*PanelState, error) {
    // Send 'ls' command, parse JSON response
}
```

#### 1.3 Config Hot Reload

**Goal**: `shinectl reload` updates panels without restart

**Approach**:
1. Main `shine` process watches config file (fsnotify)
2. On change, re-read TOML
3. Update panels via remote control
4. Restart only if component list changed

**Implementation**:
```go
// cmd/shine/main.go
func watchConfig(cfg *config.Config, mgr *panel.Manager) {
    watcher, _ := fsnotify.NewWatcher()
    watcher.Add(configPath)

    for event := range watcher.Events {
        if event.Op&fsnotify.Write == fsnotify.Write {
            newCfg := config.Load(configPath)
            applyConfigChanges(cfg, newCfg, mgr)
        }
    }
}
```

### Priority 2: User Experience

#### 2.1 Component Registry

**Goal**: Plugin-like architecture for components

**Current**: Hard-coded component checks in `main.go`
**Better**: Registry pattern

```go
// pkg/components/registry.go
type Component interface {
    Name() string
    DefaultConfig() ComponentConfig
    Launch(cfg ComponentConfig) (tea.Model, error)
}

type Registry struct {
    components map[string]Component
}

func (r *Registry) Register(c Component) {
    r.components[c.Name()] = c
}
```

#### 2.2 Declarative Widgets (Phase 2+)

**Goal**: Define simple widgets in TOML without writing Go code

**Example**:
```toml
[[widget]]
name = "clock"
type = "text"
command = "date +%H:%M"
refresh_interval = "1s"
style = "bold cyan"
position = "top-right"
```

**Implementation**: Widget engine that parses TOML and generates Bubble Tea components dynamically.

#### 2.3 Theme System

**Goal**: User-customizable colors and styles

```toml
[theme]
background = "#1e1e2e"
foreground = "#cdd6f4"
accent = "#89b4fa"
font_size = 12
```

**Apply**: Pass theme to all components via context or environment.

### Priority 3: Stability & Polish

#### 3.1 Error Handling

**Current**: Basic error messages, some panics
**Improve**:
- Graceful degradation (if Kitty not found, show helpful message)
- Retry logic for Hyprland queries
- Better validation of TOML config

#### 3.2 Logging

**Add**: Structured logging with levels

```go
import "github.com/charmbracelet/log"

logger := log.NewWithOptions(os.Stderr, log.Options{
    Level: log.DebugLevel,
})
logger.Info("Launching panel", "name", "chat", "pid", pid)
```

#### 3.3 Documentation

**Expand**:
- Component development guide
- Configuration reference (all options documented)
- Troubleshooting section
- Example configurations for common setups

---

## Development Workflow

### Setup

```bash
# Clone and build
cd ~/dev/projects/shine
go build -o bin/shine ./cmd/shine
go build -o bin/shinectl ./cmd/shinectl
go build -o bin/shine-chat ./cmd/shine-chat

# Run tests
./test.sh
# OR
go test ./...

# Install globally (optional)
sudo cp bin/shine /usr/local/bin/
sudo cp bin/shinectl /usr/local/bin/
sudo cp bin/shine-chat /usr/local/bin/
```

### Adding a New Component

1. **Create component binary**:
   ```bash
   mkdir -p cmd/shine-bar
   # Implement Bubble Tea app in cmd/shine-bar/main.go
   ```

2. **Add to launcher**:
   ```go
   // cmd/shine/main.go
   if cfg.Bar != nil && cfg.Bar.Enabled {
       barBinary, _ := findComponentBinary("shine-bar")
       panelCfg := cfg.Bar.ToPanelConfig()
       mgr.Launch("bar", panelCfg, barBinary)
   }
   ```

3. **Add config struct**:
   ```go
   // pkg/config/types.go
   type BarConfig struct {
       Enabled         bool   `toml:"enabled"`
       Edge            string `toml:"edge"`
       // ... other fields
   }

   type Config struct {
       Chat *ChatConfig `toml:"chat"`
       Bar  *BarConfig  `toml:"bar"` // Add here
   }
   ```

4. **Build and test**:
   ```bash
   go build -o bin/shine-bar ./cmd/shine-bar
   ./bin/shine  # Launches new component if enabled
   ```

### Testing Corner Positioning

```bash
# Test different corners
sed -i 's/edge = ".*"/edge = "top-left"/' ~/.config/shine/shine.toml
./bin/shine

# Test different monitor
sed -i 's/output_name = ".*"/output_name = "DP-1"/' ~/.config/shine/shine.toml
./bin/shine

# Test different sizes
sed -i 's/lines_pixels = .*/lines_pixels = 500/' ~/.config/shine/shine.toml
sed -i 's/columns_pixels = .*/columns_pixels = 600/' ~/.config/shine/shine.toml
./bin/shine
```

---

## Technical Gotchas

### 1. Kitty Config Options vs CLI Flags

**Wrong**: `--listen-on=unix:/tmp/socket`
**Right**: `-o listen_on=unix:/tmp/socket`

Kitty configuration options must be passed via `-o key=value`, not as flags. This applies to `allow_remote_control` and `listen_on`.

### 2. Corner Positioning Calculation

The calculation happens in `ToKittenArgs()`:
```go
// For top-right:
marginLeft = monitorWidth - panelWidth - marginRight

// Then use edge=top (not edge=none)
```

Using `edge=none` doesn't anchor to any edges, so margins don't position correctly. We use `edge=top` or `edge=bottom` with calculated left/right margins.

### 3. Binary Path Resolution

Components must be in PATH or in the same directory as `shine`. The `findComponentBinary()` function checks:
1. PATH (via `exec.LookPath`)
2. Same directory as `shine` binary

### 4. Ctrl+C Handling

The chat widget has Ctrl+C removed from its quit keys:
```go
// cmd/shine-chat/main.go:96-100
case tea.KeyEsc:
    // Clear current input on Esc
    m.textarea.Reset()
// Removed: case tea.KeyCtrlC
```

For new components, follow this pattern to prevent accidental closure.

---

## Resources

### Documentation
- **Kitty Panel**: `docs/llms/man/kitty.md` (lines 30490-30705)
- **Layer Shell Protocol**: `docs/llms/research/git-miner/kitty-wayland-panel.md`
- **Bubble Tea**: `docs/llms/man/charm/bubbletea.md`

### External References
- Kitty panel docs: https://sw.kovidgoyal.net/kitty/kittens/panel/
- Bubble Tea: https://github.com/charmbracelet/bubbletea
- Layer shell protocol: https://github.com/swaywm/wlr-protocols

### Example Projects
- `hypr-dock` (147 stars): https://github.com/darkmaster420/hypr-dock
  - Uses GTK3 + layer shell for Hyprland dock
  - Good reference for layer shell configuration patterns

---

## Next Session Checklist

When starting Phase 2:

- [ ] Review this handoff document
- [ ] Read `PLAN.md` for original architecture decisions
- [ ] Run `./bin/shine` to verify Phase 1 still works
- [ ] Run tests: `./test.sh`
- [ ] Check Hyprland version: `hyprctl version`
- [ ] Check Kitty version: `kitty --version`
- [ ] Decide on first Phase 2 feature (recommendation: status bar component)
- [ ] Update `PLAN.md` with Phase 2 tasks
- [ ] Create branch: `git checkout -b phase-2-{feature}`

---

## Questions & Support

**Project Lead**: starbased (s@starbased.net)
**Repository**: (add when pushed)
**Issues**: (add when repository created)

**Common Questions**:

**Q**: Why not use GTK + layer shell directly?
**A**: Kitty's GPU rendering is superior, and we wanted pure TUI (no GTK widgets). Shine focuses on terminal-based UI.

**Q**: Can components communicate with each other?
**A**: Not yet. Phase 2+ could add IPC between components via Unix sockets or shared memory.

**Q**: What about non-Hyprland compositors?
**A**: Should work on any compositor supporting layer shell (Sway, River). Tested only on Hyprland so far.

**Q**: Performance concerns?
**A**: Kitty is extremely efficient. Each panel is a separate process, so CPU usage scales linearly with component count.

---

## Final Notes

Phase 1 demonstrates the core concept works beautifully:
- Clean Go API wrapping Kitty panel
- Dynamic positioning with monitor awareness
- Bubble Tea components as desktop widgets
- Type-safe configuration with TOML

The foundation is solid. Phase 2 is about expanding the component library and improving UX (hot reload, better remote control, component registry).

**Most Important**: Keep the Unix philosophy - each component is a standalone binary that does one thing well. Shine just orchestrates them.

Good luck! 🚀

---

**Handoff Date**: 2025-11-01
**Phase 1 Completion**: ✅
**Ready for Phase 2**: ✅
