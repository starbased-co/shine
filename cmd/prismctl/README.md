# prismctl - Prism Process Supervisor

**Status**: Phase 1 MVP Complete ✓
**Version**: 1.0.0-phase1

## Overview

`prismctl` is a lightweight process supervisor for Shine prism programs (Bubble Tea TUI widgets). It provides:

- **Terminal state management** - Prevents corruption between prism swaps
- **Hot-swap capability** - Switch prisms without closing panels
- **Signal handling** - SIGCHLD, SIGTERM, SIGWINCH forwarding
- **IPC control** - JSON-based Unix socket API

## Usage

```bash
# Basic usage
prismctl <prism-name> [component-name]

# Examples
prismctl shine-clock
prismctl shine-spotify music-panel

# Get help
prismctl --help
```

## Architecture

```
prismctl (supervisor - this program)
├── IPC server (Unix socket)
├── Signal handling (SIGCHLD, SIGWINCH, SIGTERM)
├── Terminal state management (critical!)
└── Hot-swap orchestration
    ↓ fork/exec
  Child Process (Bubble Tea prism)
```

## IPC Protocol

Socket location: `/run/user/{uid}/shine/prism-{instance}.sock`

### Commands

**Swap to new prism**:
```json
{"action":"swap","prism":"shine-sysinfo"}
```

**Query status**:
```json
{"action":"status"}
```

**Stop prismctl**:
```json
{"action":"stop"}
```

### Response Format

```json
{
  "success": true,
  "message": "operation completed",
  "data": {...}
}
```

## Testing

See test suite in `../../test/integration/` for automated tests.

Quick test:
```bash
# Terminal 1
prismctl shine-clock test

# Terminal 2 (find socket first)
ls /run/user/$(id -u)/shine/

# Send swap command
echo '{"action":"swap","prism":"shine-sysinfo"}' | \
  socat - UNIX-CONNECT:/run/user/$(id -u)/shine/prism-test.<pid>.sock
```

## Implementation Files

- `main.go` - Entry point and initialization
- `terminal.go` - Terminal state save/reset/restore (CRITICAL)
- `signals.go` - Signal orchestration
- `supervisor.go` - Child process lifecycle and hot-swap
- `ipc.go` - Unix socket IPC server

## Requirements

- Linux (uses `golang.org/x/sys/unix`)
- Real TTY (stdin must be a terminal)
- Prism binaries in PATH

## Limitations (Phase 1)

- No automatic crash recovery (manual swap required)
- No persistent logging (stderr only)
- Single prism per instance

Phase 2+ will add automatic restart, health monitoring, and integration with shinectl.

## Documentation

- Architecture: See `../../docs/tour.md` for complete system overview
- IPC Protocol: `../../docs/ipc.md`
- Configuration: `../../docs/configuration.md`

## License

See project LICENSE file.
