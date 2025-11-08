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
┌────────────────────────────────────────────────────────┐
│  shine (launcher)                                      │
│  ├─ Loads ~/.config/shine/shine.toml                   │
│  ├─ Discovers prisms from configured paths             │
│  ├─ Manages prism lifecycle                            │
│  └─ Launches prisms via Kitty remote control API       │
└────────────────────────────────────────────────────────┘
                        │
                        ├─ kitty @ launch --type=os-panel ──> shine-prism (Bubble Tea TUI)
                        │
                        └─ IPC ────> Kitty socket
                                            │
                                            └─ shinectl toggle prism
```

## Prerequisites

- **Kitty** >= 0.36.0 with remote control and OS panel support
- **Hyprland** (or other Wayland compositor with wlr-layer-shell)
- **Go** >= 1.21

Verify prerequisites:

```bash
# Check Kitty version
kitty --version

# Check Hyprland (if using Hyprland)
hyprctl version

# Ensure Kitty has remote control enabled in ~/.config/kitty/kitty.conf:
# allow_remote_control yes
# listen_on unix:/tmp/@mykitty
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
[core]
path = [
    "~/.local/share/shine/bin",
    "~/.config/shine/bin",
    "~/.config/shine/prisms",
]

[prisms.chat]
enabled = true
origin = "bottom-center"
height = 10
width = 80
position = "0,10"
hide_on_focus_loss = true
focus_policy = "on-demand"
output_name = "DP-2"
```

### Configuration Options

| Option               | Type          | Default         | Description                                                                                                                                  |
| -------------------- | ------------- | --------------- | -------------------------------------------------------------------------------------------------------------------------------------------- |
| `enabled`            | bool          | `false`         | Enable/disable prism                                                                                                                         |
| `origin`             | string        | `"center"`      | Anchor point: `top-left`, `top-center`, `top-right`, `left-center`, `center`, `right-center`, `bottom-left`, `bottom-center`, `bottom-right` |
| `width`              | int or string | `1`             | Width in columns (int) or pixels (string with "px")                                                                                          |
| `height`             | int or string | `1`             | Height in lines (int) or pixels (string with "px")                                                                                           |
| `position`           | string        | `"0,0"`         | Offset from origin as "x,y" in pixels                                                                                                        |
| `hide_on_focus_loss` | bool          | `false`         | Hide panel when it loses focus                                                                                                               |
| `focus_policy`       | string        | `"not-allowed"` | Focus policy: `not-allowed`, `exclusive`, `on-demand`                                                                                        |
| `output_name`        | string        | `"DP-2"`        | Target monitor name                                                                                                                          |
| `path`               | string        | `""`            | Custom binary name or path (defaults to `shine-{name}`)                                                                                      |

See [docs/configuration.md](docs/configuration.md) for complete configuration documentation.

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
│   ├── shine-chat/     # Chat TUI component (example)
│   └── shinectl/       # Control utility & prism generator
├── pkg/
│   ├── config/         # Configuration system
│   │   ├── types.go    # Config structures (CoreConfig, PrismConfig)
│   │   ├── loader.go   # TOML loading
│   │   ├── discovery.go # Prism discovery
│   │   └── watcher.go  # Config file watching
│   ├── panel/          # Panel management
│   │   ├── config.go   # Panel config & CLI args
│   │   ├── manager.go  # Panel lifecycle via Kitty remote control
│   │   └── remote.go   # Remote control client
│   └── prism/          # Prism management
│       ├── manager.go  # Prism lifecycle
│       ├── prism.go    # Prism types
│       ├── manifest.go # Prism metadata
│       └── validate.go # Prism validation
├── examples/
│   ├── shine.toml      # Example config
│   └── prisms/         # Example prisms (weather, spotify, sysmonitor)
└── docs/
    ├── configuration.md       # Configuration guide
    ├── PRISM_DEVELOPER_GUIDE.md # Prism development guide
    └── llms/                  # LLM-optimized documentation
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
# Test prism standalone (requires TTY)
./bin/shine-chat

# Test with shine launcher
./bin/shine

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

## Creating Custom Prisms

Shine makes it easy to create your own custom widgets (prisms). A prism is a standard Bubble Tea application that follows simple conventions.

### Quick Start

```bash
# Create a new prism from template
shinectl new-prism my-widget

# Navigate and build
cd ~/.config/shine/prisms/my-widget
make build
make install

# Configure in shine.toml
# Add [prisms.my-widget] section

# Launch shine
shine
```

### Example Prisms

The `examples/prisms/` directory includes three complete examples:

1. **weather** - Weather display with auto-refresh and icons
2. **spotify** - Music player with playback controls and progress bar
3. **sysmonitor** - System resource monitor with CPU/memory/disk usage

Each example is a fully-functional, documented prism demonstrating different capabilities.

### Learn More

See the [Prism Developer Guide](docs/PRISM_DEVELOPER_GUIDE.md) for complete documentation on:

- Prism interface requirements
- Development workflow
- Best practices
- Advanced topics (API integration, state persistence, etc.)
- Example walkthroughs
- Troubleshooting

## Roadmap

### Phase 1 (Complete)

- [x] Chat component with Bubble Tea
- [x] Panel configuration system
- [x] Panel manager
- [x] Remote control client
- [x] TOML configuration
- [x] Basic tests

### Phase 2 (Complete)

- [x] Prism system architecture
- [x] Prism discovery and management
- [x] Developer tooling (`shinectl new-prism`)
- [x] Example prisms (weather, spotify, sysmonitor)
- [x] Comprehensive developer guide

### Phase 3 (Future)

- [ ] Hot reload configuration
- [ ] IPC event bus for inter-prism communication
- [ ] Prism marketplace/repository
- [ ] Theming system
- [ ] Advanced prism templates

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
