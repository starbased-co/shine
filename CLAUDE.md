# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Shine** is a TUI-based desktop shell toolkit for Hyprland that leverages Kitty's layer shell support and the Charm ecosystem (Bubble Tea, Bubbles, Lip Gloss). Write desktop panels as Bubble Tea TUIs that run in GPU-accelerated Kitty panels.

**Key Innovation**: No Wayland code required - Kitty handles all layer shell integration via `kitten panel`.

**Current Status**: Phase 2/3 complete (prismtty architecture). Production-ready for managing multiple TUI panels.

## Core Architecture

### Three-Tier System

```
shine (user CLI)
  ↓
shinectl (service manager)
  ↓ spawns panels via Kitty remote control
prismctl (panel supervisor)
  ↓ manages process lifecycle
shine-* prisms (Bubble Tea TUIs)
```

**shine**: User-facing CLI (`start`, `stop`, `reload`, `status`, `logs`)
**shinectl**: Background service that spawns/monitors panels, reads `~/.config/shine/prism.toml`
**prismctl**: Panel supervisor with suspend/resume, MRU ordering, crash recovery
**Prisms**: Individual TUI applications (e.g., `shine-clock`, `shine-chat`, `shine-bar`)

### Critical Behaviors

- **prismctl suspend/resume**: Uses SIGSTOP/SIGCONT for zero-overhead backgrounding
- **MRU ordering**: Most recently used prism stays foreground, others suspended
- **Crash recovery**: Restart policies (no, on-failure, unless-stopped, always)
- **Hot-reload**: SIGHUP to shinectl reloads config without disrupting panels
- **IPC**: JSON over Unix sockets (`/run/user/{uid}/shine/prism-{component}.{pid}.sock`)

## IPC Socket Management

**IMPORTANT**: When sending IPC commands, always find the current socket first:

```bash
# Find the most recent socket
SOCK=$(ls -t /run/user/$(id -u)/shine/prism-*.sock | head -1)

# Or find by component name
SOCK=$(ls -t /run/user/$(id -u)/shine/prism-test-prism.*.sock | head -1)

# Then use it for commands
echo '{"action":"status"}' | socat - UNIX-CONNECT:$SOCK
```

**Why this matters**: Each prismctl instance creates a socket with its PID in the name. When you restart prismctl, the old socket is removed and a new one is created. Always look up the current socket before running parallel IPC commands to avoid "No such file or directory" errors.

## Development Commands

### Build

```bash
# Build all binaries
go build -o bin/shine ./cmd/shine
go build -o bin/shinectl ./cmd/shinectl
go build -o bin/prismctl ./cmd/prismctl

# Build specific prism
go build -o bin/shine-chat ./cmd/shine-chat
go build -o bin/shine-clock ./cmd/shine-clock
go build -o bin/shine-bar ./cmd/shine-bar
```

### Testing

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./pkg/config ./pkg/panel ./pkg/prism

# Run with coverage
go test -cover ./...

# Integration tests (requires Kitty with remote control)
go test -v ./pkg/panel -tags=integration
```

### Running Locally

```bash
# Install to PATH for testing
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
./bin/shine-chat

# Test prismctl with a prism
./bin/prismctl panel-test shine-clock

# Hot-reload config
pkill -HUP shinectl

# Test IPC manually
echo '{"action":"status"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
echo '{"action":"start","prism":"shine-clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

## Code Organization

### Package Structure

```
cmd/
  shine/          # User CLI (commands.go, output.go, main.go)
  shinectl/       # Service manager (config.go, ipc_client.go, panel_manager.go, main.go)
  prismctl/       # Panel supervisor (supervisor.go, pty_manager.go, ipc.go, terminal.go, signals.go)
  shine-*/        # Example prisms (chat, clock, bar, sysinfo)

pkg/
  config/         # Configuration system (types.go, loader.go, discovery.go, watcher.go)
  panel/          # Panel lifecycle (manager.go, config.go, remote.go)
  prism/          # Prism management (manager.go, prism.go, manifest.go, validate.go)
```

### Key Files

**pkg/config/types.go**: Core config structures (`Config`, `CoreConfig`, `PrismConfig`)
**pkg/prism/manager.go**: Prism lifecycle (Launch, Stop, Reload, Health)
**pkg/panel/manager.go**: Kitty panel spawning via remote control API
**cmd/prismctl/supervisor.go**: Process supervisor with suspend/resume and MRU

### Configuration System

Three ways to configure prisms:

1. **Inline** in `~/.config/shine/shine.toml` (`[prisms.chat]`)
2. **Directory** with `prism.toml` + binary (e.g., `~/.config/shine/prisms/weather/`)
3. **Standalone** `.toml` files (e.g., `~/.config/shine/prisms/clock.toml`)

User config in `shine.toml` OVERRIDES prism defaults from `prism.toml`.
Metadata (description, author, license) comes ONLY from prism sources, never shine.toml.

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
vim cmd/shine-clock/main.go

# Rebuild
go build -o bin/shine-clock ./cmd/shine-clock
cp bin/shine-clock ~/.local/bin/

# Hot-swap in running panel (via prismctl IPC)
echo '{"action":"start","prism":"shine-clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
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
./bin/shine-clock
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
- 10ms stabilization delay after resume (SIGCONT)
- 20ms shutdown grace period (SIGTERM → SIGKILL)
- Terminal state restoration requires exact sequencing

### Socket Naming Convention

```
/run/user/{uid}/shine/prism-{component}.{pid}.sock  # prismctl sockets
```

Example: `/run/user/1000/shine/prism-panel-0.12345.sock`

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
{"action": "start", "prism": "shine-clock"}
{"action": "kill", "prism": "shine-clock"}
{"action": "status"}
{"action": "stop"}
```

**Response**:
```json
{"success": true, "foreground": "shine-clock", "background": ["shine-chat"]}
{"success": false, "error": "prism not found"}
```

### Terminal State Management

prismctl preserves terminal state when suspending:
1. Save current terminal attributes (termios)
2. SIGSTOP to suspend process
3. On resume: restore attributes, SIGCONT, 10ms stabilization

**CRITICAL**: Never modify terminal state outside prismctl supervisor code.

## Documentation References

- **README.md**: User-facing overview, installation, usage
- **docs/QUICKSTART.md**: 5-minute getting started guide
- **docs/PHASE2-3-IMPLEMENTATION.md**: Detailed implementation report for Phase 2 & 3
- **docs/configuration.md**: Complete configuration reference
- **examples/shine.toml**: Fully commented example config
- **examples/prism.toml**: Prism configuration example
- **docs/llms/**: LLM-optimized documentation (Charm ecosystem, Hyprland, etc.)

## Known Limitations

From architecture (Phase 4 future work):
- No eviction policy - unlimited suspended prisms
- No persistence - MRU list lost on prismctl restart
- No prism tagging (pin/evict)
- No memory limits - background prisms consume full memory
- No per-prism logs yet (only shinectl.log)
- Config reload requires manual SIGHUP (no `shine reload` IPC yet)

## Version Information

- **Current Version**: 0.2.0 (from cmd/shine/main.go)
- **Go Version**: 1.25.1
- **Dependencies**: Charm Bracelet (bubbletea, bubbles, lipgloss), Kitty integration
