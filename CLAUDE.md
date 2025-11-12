# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Shine** is a TUI-based desktop shell toolkit for Hyprland that leverages Kitty's layer shell support and the Charm ecosystem (Bubble Tea, Bubbles, Lip Gloss). Write desktop panels as Bubble Tea TUIs that run in GPU-accelerated Kitty panels.

**Key Innovation**: No Wayland code required - Kitty handles all layer shell integration via `kitten panel`.

## Core Architecture

### The Optical Metaphor

**Prisms** are the Kitty panels equipped with `prismctl`. Each prism can hold multiple light sources (TUI child processes) and refract one to the user at a time through its surface.

- **Panel** = The Kitty layer shell window spawned by `shinectl`
- **Prism** = Panel + `prismctl` supervisor (the refracting container)
- **Light Sources** = TUI child processes (e.g., `clock`, `chat`, `bar`) that beam their output
- **Surface** = The bidirectional I/O mechanism that refracts light from the active source to the user

```
┌─────────────────┐
│      shine      │ <- User: shine start/stop/status
└────────┬────────┘
         │ IPC (Unix socket: /run/user/{uid}/shine/shine.*.sock)
         │
┌────────┴────────┐
│     shinectl    │ <- Service manager: reads prism.toml config
└────────┬────────┘
         │ kitten @ launch --type=os-panel prismctl panel-0
         │
┌────────┴────────┐
│      kitty      │ <- Layer shell panel window (positioned via --os-panel flags)
└────────┬────────┘
        ↓│↑ Real PTY
┌────────┴────────┐
│      pty_M      │ <- Real terminal PTY master (Kitty's PTY)
└────────┬────────┘
        ↓│↑ Surface (bidirectional)
┌────────┴────────┐      ┌──────────────┐
│      pty_S      │<────>│  prismctl    │ <- Prism supervisor (IPC: prism-{instance}.sock)
└─────────────────┘      └─┬────┬────┬──┘    Foreground TUI visible to user
                    ┌──────┘    │    └──────┐
               ┌────┴─────┐┌────┴─────┐┌────┴─────┐
               │   PTY1   ││   PTY2   ││  *PTY3   │ <- Child PTYs (* = active/foreground)
               └────┬─────┘└────┬─────┘└────┬─────┘
               ┌────┴─────┐┌────┴─────┐┌────┴─────┐
               │  clock   ││  wabar   ││   app3   │ <- TUI processes (light sources)
               └──────────┘└──────────┘└──────────┘
                background   background   FOREGROUND
```

**Flow explanation:**

1. User runs `shine start` → spawns `shinectl` in background
2. `shinectl` reads `prism.toml` and spawns Kitty panels via remote control
3. Each Kitty panel launches `prismctl` as its command (the prism supervisor)
4. `prismctl` allocates PTYs and forks TUI child processes (light sources)
5. The surface relays I/O between Real PTY and the active child's PTY
6. MRU ordering: most recent child stays foreground, others continue running in background
7. Hot-swap: `prismctl` can switch which light source is refracted through the surface

### Three-Tier System

```
shine (user CLI)
  ↓ sends IPC commands
shinectl (service manager)
  ↓ spawns Kitty panels via remote control
kitten panel [prismctl panel-name]
  ↓ prismctl is the prism (container + supervisor)
Light sources (TUI processes)
  ↓ one source at a time is refracted through the surface
User's terminal
```

**shine**: User-facing CLI (`start`, `stop`, `reload`, `status`, `logs`)
**shinectl**: Service manager that reads config and launches `kitten panel` processes with `prismctl` as the command
**prismctl**: The prism - supervisor that manages multiple TUI child processes (light sources) and refracts one to the user via the surface
**Light Sources**: Individual TUI applications (e.g., `clock`, `chat`, `bar`) that run as children of prismctl

### Critical Behaviors

- **Background processing**: All TUI light sources continue running, only I/O relay switches
- **MRU ordering**: Most recently used light source gets I/O relay, others continue in background
- **Crash recovery**: Restart policies (no, on-failure, unless-stopped, always)
- **Hot-reload**: SIGHUP to shinectl reloads config without disrupting panels
- **IPC**: JSON over Unix sockets (`/run/user/{uid}/shine/prism-{instance}.sock`)

## IPC Socket Management

**IMPORTANT**: Socket paths are deterministic based on prism name:

```bash
# Direct socket path by prism name (instance)
SOCK=/run/user/$(id -u)/shine/prism-clock.sock

# Use it for commands
echo '{"action":"status"}' | socat - UNIX-CONNECT:$SOCK

# Or find all prismctl sockets
ls /run/user/$(id -u)/shine/prism-*.sock
```

## Development Commands

### Build

```bash
# Build all binaries
make build

# Or individually:
go build -o bin/shine ./cmd/shine
go build -o bin/shinectl ./cmd/shinectl
go build -o bin/prismctl ./cmd/prismctl

# Build specific prism
go build -o bin/chat ./cmd/prisms/chat
go build -o bin/clock ./cmd/prisms/clock
go build -o bin/bar ./cmd/prisms/bar
```

### Testing

```bash
# Run all tests
make test

# Or:
go test ./cmd/... ./pkg/...

# Run with verbose output
go test -v ./pkg/config ./pkg/panel

# Run with coverage
go test -cover ./...

# Integration tests (requires Kitty with remote control)
make test-integration
```

### Running Locally

```bash
# Install to PATH for testing
make install

# Or add bin/ to PATH temporarily
export PATH="$PWD/bin:$PATH"

# Start service
shine start

# Check status
shine status

# View logs
shine logs shinectl

# Stop service
shine stop
```

### Manual Testing

```bash
# Test prism standalone (requires TTY)
./bin/chat

# Test prismctl with a prism
./bin/prismctl panel-test clock

# Hot-reload config
pkill -HUP shinectl

# Test IPC manually
echo '{"action":"status"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
echo '{"action":"start","prism":"clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

### Help System Testing

```bash
# Test human-readable help
shine help start
shine help list
shine help categories

# Test JSON output (for tooling integration)
shine help start --json
shine help --json names
shine help --json categories

# Test shell completion
source examples/completion.bash
shine <TAB>
shine help <TAB>
```

## Code Organization

### Package Structure

```
cmd/
  shine/          # User CLI
    main.go         # CLI entry point and command routing
    commands.go     # Command implementations (start, stop, status, etc.)
    output.go       # Terminal output formatting
    help.go         # Help rendering and display logic
    help_metadata.go # Structured help metadata and registry
    help/           # Markdown help content
      usage.md      # Main help page
      start.md      # Per-command help pages
      stop.md
      status.md
      reload.md
      logs.md
  shinectl/       # Service manager (config.go, ipc_client.go, panel_manager.go, main.go)
  prismctl/       # Panel supervisor (supervisor.go, surface.go, pty_manager.go, ipc.go, terminal.go, signals.go)
  prisms/         # Example prisms
    bar/          # Status bar prism
    chat/         # Chat panel prism
    clock/        # Clock widget prism
    sysinfo/      # System info widget prism

pkg/
  config/         # Configuration system (types.go, loader.go, discovery.go, watcher.go)
  panel/          # Panel configuration (config.go positioning logic, remote.go control helpers)
```

### Key Files

**pkg/config/types.go**: Core config structures (`Config`, `CoreConfig`, `PrismConfig`)
**pkg/config/discovery.go**: Prism discovery from configured directories (three types)
**pkg/panel/config.go**: Panel positioning logic (origin, margins, dimensions)
**cmd/shinectl/config.go**: Hybrid config (positioning + restart policies)
**cmd/shinectl/panel_manager.go**: Spawns Kitty panels via remote control, health monitoring
**cmd/prismctl/supervisor.go**: Process supervisor with surface switching and MRU
**cmd/prismctl/surface.go**: Bidirectional I/O surface (Real PTY ↔ child PTY)
**cmd/shine/help_metadata.go**: Help system registry and metadata structures
**cmd/shine/help.go**: Help rendering (Glamour), JSON output, topic generation

### Help System Architecture

The CLI help system uses a **hybrid approach**:

**Markdown Files** (cmd/shine/help/\*.md)

- Long-form content with examples and troubleshooting
- Embedded at compile-time via `//go:embed`
- Rendered beautifully with Glamour

**Structured Metadata** (help_metadata.go)

- `CommandHelp` struct with name, category, synopsis, usage, related commands
- `helpRegistry` map for centralized command metadata
- Enables programmatic access and multiple output formats

**Multiple Output Formats**

- Human-readable: Glamour-rendered markdown (`shine help start`)
- Listings: Generated from metadata (`shine help list`, `shine help categories`)
- Machine-readable: JSON output (`shine help start --json`)

**Use Cases**

- User documentation: Rich terminal help with examples
- Shell completion: `shine help --json names` provides command list
- IDE integration: JSON metadata for hover text and autocomplete
- Future: Man page generation, HTML docs, interactive TUI help browser

### Configuration System

Three ways to configure prisms:

1. **Inline** in `~/.config/shine/shine.toml` (`[prisms.chat]`)
2. **Directory** with `prism.toml` + binary (e.g., `~/.config/shine/prisms/weather/`)
3. **Standalone** `.toml` files (e.g., `~/.config/shine/prisms/clock.toml`)

User config in `shine.toml` OVERRIDES prism defaults from `prism.toml`.

## Common Development Tasks

### Adding a New Prism

```bash
# Create boilerplate
shinectl new-prism my-widget

# Navigate to prism directory
cd ~/.config/shine/prisms/my-widget

# Build and install
make build
make install

# Add to config
vim ~/.config/shine/shine.toml
# Add [prisms.my-widget] section with enabled = true

# Reload shine
pkill -HUP shinectl
```

### Modifying Existing Prism

```bash
# Edit prism code
vim cmd/prisms/clock/main.go

# Rebuild
go build -o bin/clock ./cmd/prisms/clock
cp bin/clock ~/.local/bin/

# Hot-swap in running panel (via prismctl IPC)
echo '{"action":"start","prism":"clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

### Debugging Panel Issues

```bash
# Check shinectl logs
tail -f ~/.local/share/shine/logs/shinectl.log

# Verify sockets exist
ls -la /run/user/$(id -u)/shine/

# Test Kitty remote control
kitty @ ls

# Run prism standalone to isolate issues
./bin/clock
```

### Configuration Hot-Reload Testing

```bash
# Start service
shine start

# Modify config
vim ~/.config/shine/prism.toml

# Reload (sends SIGHUP to shinectl)
pkill -HUP shinectl

# Verify changes
shine status
```

## Important Constraints

### Kitty Remote Control

- Must have `allow_remote_control yes` in `~/.config/kitty/kitty.conf`
- Must have `listen_on unix:/tmp/@mykitty` (or similar socket)
- Panel manager auto-detects socket via env vars or pgrep

### prismctl Timing Constants

**DO NOT MODIFY** these without careful testing (from prismctl implementation):

- 10ms stabilization delay after surface switch
- 20ms shutdown grace period (SIGTERM → SIGKILL)
- Terminal state restoration requires exact sequencing

### Socket Naming Convention

```
/run/user/{uid}/shine/prism-{instance}.sock  # prismctl sockets
```

Where `{instance}` is the prism name (e.g., "clock", "bar", "spotify")

Example: `/run/user/1000/shine/prism-clock.sock`

### Restart Policies

From `prism.toml`:

- `restart = "no"` - Never restart (default)
- `restart = "on-failure"` - Restart only on non-zero exit
- `restart = "unless-stopped"` - Always restart unless explicitly stopped
- `restart = "always"` - Restart unconditionally

Additional settings:

- `restart_delay` - Delay before restart (e.g., "5s", "500ms")
- `max_restarts` - Max restarts per hour (0 = unlimited)

## Prism Development Guidelines

### Bubble Tea Pattern

All prisms are standard Bubble Tea applications:

```go
type model struct {
    // Your state
}

func (m model) Init() tea.Cmd { ... }
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { ... }
func (m model) View() string { ... }

func main() {
    p := tea.NewProgram(initialModel())
    if _, err := p.Run(); err != nil {
        log.Fatal(err)
    }
}
```

### Required Prism Interface

Prisms MUST:

1. Be executable binaries in PATH or configured paths
2. Accept no required CLI arguments (config via files if needed)
3. Exit cleanly on SIGTERM/SIGINT
4. Handle terminal resize (SIGWINCH forwarded by prismctl)

### Prism Manifest (prism.toml)

Example:

```toml
name = "shine-weather"
version = "1.0.0"
enabled = true
origin = "top-right"
width = "200px"
height = "100px"

[metadata]
description = "Weather widget with forecasts"
author = "your-name"
license = "MIT"
```

## Project-Specific Patterns

### Error Handling

- Use `fmt.Errorf("context: %w", err)` for wrapping
- Log errors before returning in manager/supervisor code
- Rich CLI output via lipgloss in `cmd/shine/output.go`

### IPC Protocol

JSON messages over Unix sockets:

**Request**:

```json
{"action": "start", "prism": "clock"}
{"action": "kill", "prism": "clock"}
{"action": "status"}
{"action": "stop"}
```

**Response**:

```json
{"success": true, "foreground": "clock", "background": ["chat"]}
{"success": false, "error": "prism not found"}
```

### Terminal State Management

prismctl preserves terminal state when switching surfaces:

1. Save current terminal attributes (termios)
2. Reset terminal to canonical mode before switching
3. On switch: restore attributes, sync PTY size, 10ms stabilization

**CRITICAL**: Never modify terminal state outside prismctl supervisor code.

## Documentation References

- **README.md**: User-facing overview, installation, usage
- **docs/QUICKSTART.md**: 5-minute getting started guide
- **docs/configuration.md**: Complete configuration reference
- **cmd/shine/help/*.md**: Embedded help content rendered with Glamour
- **cmd/shine/help_metadata.go**: Structured metadata for help system
- **examples/shine.toml**: Fully commented example config
- **examples/prism.toml**: Prism configuration example
- **examples/completion.zsh**: zsh shell completion script
- **examples/completion.bash**: bash shell completion script
- **docs/llms/**: LLM-optimized documentation (Charm ecosystem, Hyprland, etc.)
- **docs/tour.md**: Comprehensive tour of the codebase
- **docs/ipc.md**: IPC protocol documentation

## Known Limitations

- No eviction policy - unlimited background light sources
- No persistence - MRU list lost on prismctl restart
- No light source tagging (pin/evict)
- No memory limits - background light sources consume full memory
- No per-light-source logs yet (only shinectl.log)
- Config reload requires manual SIGHUP (no `shine reload` IPC yet)

## Version Information

- **Current Version**: 0.2.0 (from cmd/shine/main.go)
- **Go Version**: 1.25.1
- **Dependencies**: Charm Bracelet (bubbletea, bubbles, lipgloss), Kitty integration
