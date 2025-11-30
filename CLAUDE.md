# I am the `shine` assistant

**CRITICAL FOR ALL AGENTS**: When building Go code in this project:

- **ALWAYS** use `make build` or `go build -o bin/<name> ./cmd/<name>`
- **NEVER** run bare `go build ./cmd/...` - this places binaries in the project root
- Binaries belong in `bin/` directory ONLY

## Project Overview

**Shine** is a TUI-based desktop shell toolkit for Hyprland that leverages Kitty's layer shell support and the Charm ecosystem (Bubble Tea, Bubbles, Lip Gloss). Write desktop panels as Bubble Tea TUIs that run in GPU-accelerated Kitty panels.

**Key Innovation**: No Wayland code required - Kitty handles all layer shell integration via `kitten panel`.

## Core Architecture

### Terminology

- **Panel** = The Kitty layer shell window spawned by `shined`
- **Prism** = Panel + `prismctl` supervisor process
- **Child Processes** = TUI applications (e.g., `clock`, `chat`, `bar`) managed by the supervisor
- **Mirror** = The bidirectional I/O relay between the real PTY and the active child's PTY

```
┌─────────────────┐
│      shine      │ <- User: shine start/stop/status
└────────┬────────┘
         │ JSON-RPC 2.0 over Unix socket
         │
┌────────┴────────┐
│     shined    │ <- Service manager: reads shine.toml config
└────────┬────────┘
         │ kitten @ launch --type=os-panel prismctl {instance}
         │
┌────────┴────────┐
│      kitty      │ <- Layer shell panel window
└────────┬────────┘
        ↓│↑ Real PTY
┌────────┴────────┐
│      pty_M      │ <- Real terminal PTY master (Kitty's PTY)
└────────┬────────┘
        ↓│↑ Mirror (bidirectional)
┌────────┴────────┐      ┌──────────────┐
│      pty_S      │<────>│  prismctl    │ <- PTY supervisor (prism-{instance}.sock)
└─────────────────┘      └─┬────┬────┬──┘
                    ┌──────┘    │    └──────┐
               ┌────┴─────┐┌────┴─────┐┌────┴─────┐
               │   PTY1   ││   PTY2   ││  *PTY3   │ <- Child PTYs (* = foreground)
               └────┬─────┘└────┬─────┘└────┬─────┘
               ┌────┴─────┐┌────┴─────┐┌────┴─────┐
               │  clock   ││  wabar   ││   app3   │ <- TUI child processes
               └──────────┘└──────────┘└──────────┘
                background   background   FOREGROUND
```

### Three-Tier System

| Component    | Binary         | Role                   | Socket                                        |
| ------------ | -------------- | ---------------------- | --------------------------------------------- |
| **shine**    | `cmd/shine`    | User CLI               | Connects to shine.sock, prism-\*.sock         |
| **shined** | `cmd/shined` | Service manager daemon | `/run/user/{uid}/shine/shine.sock`            |
| **prismctl** | `cmd/prismctl` | PTY multiplexer        | `/run/user/{uid}/shine/prism-{instance}.sock` |
| **Prisms**   | `cmd/prisms/*` | TUI applications       | N/A                                           |

### Critical Behaviors

- **Background processing**: All TUI child processes continue running, only I/O relay switches
- **MRU ordering**: Most recently used child gets I/O relay, others continue in background
- **Hot-reload**: SIGHUP to shined reloads config without disrupting panels
- **State files**: Binary mmap files for lock-free status reads

## IPC Protocol

**Transport**: JSON-RPC 2.0 over Unix sockets via `creachadair/jrpc2`

### Socket Paths

```bash
# shined socket
/run/user/$(id -u)/shine/shine.sock

# prismctl sockets (per instance)
/run/user/$(id -u)/shine/prism-{instance}.sock

# State files (mmap, binary format)
/run/user/$(id -u)/shine/shined.state
/run/user/$(id -u)/shine/prism-{instance}.state
```

### RPC Methods

**prismctl (prism-{instance}.sock)**:

- `prism/up {name}` → Start/resume prism, returns `{pid, state}`
- `prism/down {name}` → Stop prism, returns `{stopped}`
- `prism/fg {name}` → Bring to foreground, returns `{ok, was_fg}`
- `prism/bg {name}` → Send to background, returns `{ok, was_bg}`
- `prism/list {}` → List all prisms with state
- `service/health {}` → Health check
- `service/shutdown {graceful}` → Shutdown

**shined (shine.sock)**:

- `panel/list {}` → List all panels
- `panel/spawn {config}` → Spawn new panel
- `panel/kill {instance}` → Kill panel
- `service/status {}` → Aggregated status
- `config/reload {}` → Reload configuration

### Notifications (prismctl → shined)

- `prism/started {panel, name, pid}`
- `prism/stopped {panel, name, exit_code}`
- `prism/crashed {panel, name, exit_code, signal}`
- `foreground/changed {panel, from, to}`

## Development Commands

### Build

**IMPORTANT**: Always use `make build` or specify `-o bin/` to avoid placing binaries in the project root.

```bash
# Build all binaries (preferred)
make build

# Or individually (ALWAYS use -o bin/):
go build -o bin/shine ./cmd/shine
go build -o bin/shined ./cmd/shined
go build -o bin/prismctl ./cmd/prismctl

# Build specific prism
go build -o bin/clock ./cmd/prisms/clock

# NEVER run bare `go build ./cmd/...` - creates binaries in project root
```

### Testing

```bash
make test              # Unit tests
make test-integration  # Integration tests (requires PTY)
make test-all          # All tests
go test -v ./pkg/...   # Verbose package tests
go test -cover ./...   # With coverage
```

### Running Locally

```bash
make install                    # Install to ~/.local/bin
export PATH="$PWD/bin:$PATH"    # Or add bin/ to PATH

shine start                     # Start service
shine status                    # Check status
shine logs shined             # View logs
shine stop                      # Stop service
```

## Code Organization

### Package Structure

```
cmd/
  shine/              # User CLI
    main.go             # Entry point
    commands.go         # Command implementations
    output.go           # Terminal output formatting
    help.go             # Help rendering (Glamour)
    help_metadata.go    # Structured help metadata
    help/               # Embedded markdown help files
  shined/           # Service manager
    main.go             # Entry point
    config.go           # Config loading
    panel_manager.go    # Spawns Kitty panels
    panel_handlers.go   # RPC handlers
    ipc_server.go       # RPC server
    state.go            # State management
    notifications.go    # Notification handling
  prismctl/           # PTY supervisor
    main.go             # Entry point
    supervisor.go       # Process supervision
    mirror.go           # I/O relay
    pty_manager.go      # PTY allocation
    handlers.go         # RPC handlers
    ipc.go              # RPC server
    state.go            # State management
    notifications.go    # Notification sending
    terminal.go         # Terminal state
    signals.go          # Signal handling
  prisms/             # Example prisms
    bar/                # Status bar
    chat/               # Chat panel
    clock/              # Clock widget
    sysinfo/            # System info

pkg/
  config/             # Configuration system
    types.go            # Config structures
    loader.go           # TOML loading
    discovery.go        # Prism discovery
    validation.go       # Config validation
    watcher.go          # File watching
  panel/              # Panel configuration
    config.go           # Positioning logic
    remote.go           # Kitty remote control
  rpc/                # RPC infrastructure
    types.go            # Request/response types
    client.go           # RPC client
    server.go           # RPC server
    errors.go           # Error handling
  state/              # State management
    types.go            # State structures
    reader.go           # Mmap reader
    writer.go           # Mmap writer
    mmap.go             # Mmap utilities
  paths/              # Path utilities
    paths.go            # XDG paths, socket paths
  help/               # Help system
    registry.go         # Help registry
    render.go           # Glamour rendering
```

### Key Files

- `pkg/config/types.go`: Core config structures (`Config`, `CoreConfig`, `PrismConfig`)
- `pkg/config/discovery.go`: Prism discovery from configured directories
- `pkg/rpc/types.go`: All RPC request/response types and notifications
- `pkg/state/types.go`: Binary state structures for mmap files
- `cmd/prismctl/supervisor.go`: Process supervisor with mirror switching and MRU
- `cmd/prismctl/mirror.go`: Bidirectional I/O relay
- `cmd/shined/panel_manager.go`: Spawns Kitty panels via remote control

### Configuration System

Three ways to configure prisms:

1. **Inline** in `~/.config/shine/shine.toml` (`[prisms.chat]`)
2. **Directory** with `prism.toml` + binary (e.g., `~/.config/shine/prisms/weather/`)
3. **Standalone** `.toml` files (e.g., `~/.config/shine/prisms/clock.toml`)

User config in `shine.toml` OVERRIDES prism defaults from `prism.toml`.

## Runtime Paths

| Resource   | Location                         |
| ---------- | -------------------------------- |
| Config     | `~/.config/shine/shine.toml`     |
| Prism dirs | `~/.config/shine/prisms/{name}/` |
| Logs       | `~/.local/share/shine/logs/`     |
| Sockets    | `/run/user/{uid}/shine/*.sock`   |
| State      | `/run/user/{uid}/shine/*.state`  |

## Important Constraints

### Build Artifacts

**NEVER** run bare `go build ./cmd/...` - this places binaries in the project root.

Always use:

- `make build` (preferred)
- `go build -o bin/<name> ./cmd/<name>`

### Kitty Remote Control

- Must have `allow_remote_control yes` in `~/.config/kitty/kitty.conf`
- Must have `listen_on unix:/tmp/@mykitty` (or similar socket)
- Panel manager auto-detects socket via env vars or pgrep

### prismctl Timing Constants

**DO NOT MODIFY** these without careful testing:

- 10ms stabilization delay after mirror swap
- 20ms shutdown grace period (SIGTERM → SIGKILL)
- Terminal state restoration requires exact sequencing

### Prism Interface

Prisms MUST:

1. Be executable binaries in PATH or configured paths
2. Accept no required CLI arguments
3. Exit cleanly on SIGTERM/SIGINT
4. Handle terminal resize (SIGWINCH forwarded by prismctl)

## Documentation References

- **docs/configuration.md**: Configuration reference
- **docs/rpc-tour.md**: RPC protocol documentation
- **cmd/shine/help/\*.md**: Embedded CLI help (Glamour-rendered)
- **docs/llms/**: LLM-optimized documentation (Charm ecosystem, Hyprland, etc.)

## Known Limitations

- No eviction policy - unlimited background child processes
- No persistence - MRU list lost on prismctl restart
- No per-prism resource limits
- No built-in prism lifecycle hooks

## Version Information

- **Current Version**: 0.2.0
- **Go Version**: 1.25.1
- **Dependencies**: Charm Bracelet (bubbletea, bubbles, lipgloss, glamour), Kitty, jrpc2
