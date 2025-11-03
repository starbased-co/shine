# Shine

**Hyprland Wayland Layer Shell TUI Desktop Shell Toolkit**

Shine is a toolkit for building beautiful TUI-based desktop shell components for Hyprland using the Charm ecosystem (Bubble Tea, Bubbles, Lip Gloss). Instead of implementing Wayland layer shell bindings from scratch, we leverage Kitty's battle-tested GPU-accelerated terminal with built-in layer shell support via `kitten panel`.

## Status

**Phase 1: Prototype** - Single component (chat) with basic panel management.

## Features

- **Pure Bubble Tea Components**: Write desktop panels using familiar TUI libraries
- **GPU-Accelerated Rendering**: Powered by Kitty's OpenGL engine
- **Zero Wayland Code**: Kitty handles all layer shell integration
- **Remote Control**: Toggle panel visibility programmatically
- **TOML Configuration**: Simple, user-friendly configuration

## Architecture

```
┌─────────────────────────────────────────────────┐
│  shine (launcher)                               │
│  ├─ Loads ~/.config/shine/shine.toml            │
│  ├─ Manages panel lifecycle                     │
│  └─ Launches components via kitten panel        │
└─────────────────────────────────────────────────┘
                        │
                        ├─ spawn ──> kitten panel ──> shine-chat (Bubble Tea TUI)
                        │
                        └─ IPC ────> Unix socket (remote control)
                                            │
                                            └─ shinectl toggle chat
```

## Prerequisites

- **Kitty** >= 0.36.0 with `kitten panel` support
- **Hyprland** (or other Wayland compositor with wlr-layer-shell)
- **Go** >= 1.21

Verify prerequisites:

```bash
# Check Kitty version
kitty --version

# Check if kitten panel exists
kitten panel --help

# Check Hyprland (if using Hyprland)
hyprctl version
```

## Installation

### Build from Source

```bash
# Clone repository
git clone https://github.com/starbased-co/shine.git
cd shine

# Build binaries
go build -o bin/shine ./cmd/shine
go build -o bin/shinectl ./cmd/shinectl
go build -o bin/shine-chat ./cmd/shine-chat

# Verify build
ls -lh bin/
```

### Install to PATH

```bash
# Copy to ~/.local/bin (ensure it's in your PATH)
cp bin/shine bin/shinectl bin/shine-chat ~/.local/bin/

# Or add bin/ to PATH temporarily
export PATH="$PWD/bin:$PATH"
```

## Configuration

Create `~/.config/shine/shine.toml`:

```bash
mkdir -p ~/.config/shine
cp examples/shine.toml ~/.config/shine/shine.toml
```

Example configuration:

```toml
[chat]
enabled = true
edge = "bottom"
lines = 10
margin_left = 10
margin_right = 10
margin_bottom = 10
hide_on_focus_loss = true
focus_policy = "on-demand"
```

### Configuration Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `enabled` | bool | `false` | Enable/disable component |
| `edge` | string | `"top"` | Panel edge: `top`, `bottom`, `left`, `right`, `center`, `center-sized`, `background` |
| `lines` | int | `1` | Height in terminal lines |
| `columns` | int | `1` | Width in terminal columns (for left/right edges) |
| `lines_pixels` | int | `0` | Height in pixels (overrides `lines`) |
| `columns_pixels` | int | `0` | Width in pixels (overrides `columns`) |
| `margin_top` | int | `0` | Top margin in pixels |
| `margin_left` | int | `0` | Left margin in pixels |
| `margin_bottom` | int | `0` | Bottom margin in pixels |
| `margin_right` | int | `0` | Right margin in pixels |
| `hide_on_focus_loss` | bool | `false` | Hide panel when it loses focus |
| `focus_policy` | string | `"not-allowed"` | Focus policy: `not-allowed`, `exclusive`, `on-demand` |
| `output_name` | string | `""` | Target monitor name (empty = primary) |

## Usage

### Launch All Panels

```bash
shine
```

This will:
1. Load configuration from `~/.config/shine/shine.toml`
2. Launch all enabled components
3. Stay running as orchestrator process
4. Handle Ctrl+C gracefully

### Control Panels

```bash
# Toggle chat visibility
shinectl toggle chat

# Show chat (future)
shinectl show chat

# Hide chat (future)
shinectl hide chat
```

### Hyprland Keybindings

Add to `~/.config/hypr/hyprland.conf`:

```conf
# Toggle chat panel
bind = SUPER, C, exec, shinectl toggle chat

# Future: Reload shine config
# bind = SUPER SHIFT, R, exec, shinectl reload
```

## Development

### Project Structure

```
shine/
├── cmd/
│   ├── shine/          # Main launcher
│   ├── shine-chat/     # Chat TUI component
│   └── shinectl/       # Control utility
├── pkg/
│   ├── panel/          # Panel management
│   │   ├── config.go   # LayerShellConfig
│   │   ├── manager.go  # Panel lifecycle
│   │   └── remote.go   # Remote control client
│   └── config/         # Configuration
│       ├── types.go    # Config structures
│       └── loader.go   # TOML loading
├── examples/
│   └── shine.toml      # Example config
└── docs/
    └── llms/           # LLM-optimized documentation
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run with verbose output
go test -v ./pkg/config ./pkg/panel

# Run with coverage
go test -cover ./...
```

### Building

```bash
# Build all binaries
go build -o bin/shine ./cmd/shine
go build -o bin/shinectl ./cmd/shinectl
go build -o bin/shine-chat ./cmd/shine-chat

# Build with debug info
go build -gcflags="all=-N -l" -o bin/shine ./cmd/shine
```

### Testing Manually

```bash
# Test shine-chat standalone (requires TTY)
./bin/shine-chat

# Test with kitty panel directly
kitten panel --edge=bottom --lines=10 \
    --margin-left=10 --margin-right=10 \
    --listen-on=unix:/tmp/test.sock \
    ./bin/shine-chat

# Test remote control
./bin/shinectl toggle chat
```

## Troubleshooting

### Panel doesn't launch

```bash
# Check if shine is running
ps aux | grep shine

# Check kitten panel support
kitten panel --help

# Run shine with verbose output
./bin/shine
```

### Remote control fails

```bash
# Check if socket exists
ls -la /tmp/shine-chat.sock

# Check if panel is running with remote control
ps aux | grep shine-chat

# Test socket manually
echo '{"cmd":"resize-os-window","action":"toggle-visibility"}' | nc -U /tmp/shine-chat.sock
```

### Panel not visible on Sway

Sway renders its background over panels on the background layer. Disable Sway's background:

```
output * bg #000000 solid_color
```

## Roadmap

### Phase 1 (Current)
- [x] Chat component with Bubble Tea
- [x] Panel configuration system
- [x] Panel manager
- [x] Remote control client
- [x] TOML configuration
- [x] Basic tests

### Phase 2 (Future)
- [ ] Additional components (status bar, dock, widgets)
- [ ] Hot reload configuration
- [ ] Declarative widget system
- [ ] IPC event bus
- [ ] Component marketplace
- [ ] Theming system

## Documentation

- [PLAN.md](PLAN.md) - Complete development plan
- [docs/llms/research/git-miner/kitty-wayland-panel.md](docs/llms/research/git-miner/kitty-wayland-panel.md) - Kitty layer shell research
- [docs/llms/man/charm/](docs/llms/man/charm/) - Charm ecosystem documentation

## License

See [LICENSE](LICENSE) file.

## Contributing

Contributions welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

## Acknowledgments

- [Kitty Terminal](https://sw.kovidgoyal.net/kitty/) - For the amazing layer shell support
- [Charm Bracelet](https://charm.sh/) - For the beautiful TUI libraries (Bubble Tea, Bubbles, Lip Gloss)
- [Hyprland](https://hyprland.org/) - For the fantastic Wayland compositor
