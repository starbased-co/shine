# `shine ⬘`

![Discord](https://img.shields.io/discord/1418762336982007960?style=social&logo=discord&logoColor=%235865F2&label=Share%20your%20shine%20%E2%AC%98!%20Join%20starbased%40HQ&link=https%3A%2F%2Fdiscord.gg%2XBvrkZfrQC)

`shine` orchestrates [kitty panels](https://sw.kovidgoyal.net/kitty/kittens/panel/) as Wayland layer shell surfaces, leaning on kitty's client connection to the wayland compositor to offer a configuration based workflow for reproducible and sharable desktop shells. Each kitten panel is driven by a dedicated PTY managed by the integrated multiplexer `prismctl`, which also serves as the control plane for the terminal applications—called prisms—running within. `shine` is built around the Charm ecosystem ([Bubble Tea](https://github.com/charmbracelet/bubbletea), [Lip Gloss](https://github.com/charmbracelet/lipgloss), [Bubbles](https://github.com/charmbracelet/bubbles)), but prismctl itself is process-agnostic: anything that speaks PTY works. The prism just defines what to run and where to put it; the process inside can be Go, Rust, Python, or a shell script.

## Usage

```bash
shine start    # Start the service
shine status   # Check status
shine stop     # Stop
```

Panels are configured in `~/.config/shine/shine.toml`:

```toml
[prisms.clock]
path = "shine-clock"
origin = "top-right"
width = "200px" # todo: percentage units, other units?
height = "60px"
enabled = true
```

## Overview

```
                      ┌─────────────────┐
                      │      shine      │ <- $ shine start/stop/status
                      └────────┬────────┘
                              ↓│↑ JSON-RPC 2.0 ⇆
┌──────────────────────────────┴──────────────────────┐
│  shined                                             │
│  ├─ Loads ~/.config/shine/shine.toml                │
│  │  └─ Discovers prism.toml defs in prisms/         │
│  ├─ Manages prism lifecycle                         │
│  └─ Launches prisms via Kitty remote control API    │
└──────────────────────┬──────────────────────────────┘
    unix:@shine.sock ⇆ │ ⇆ unix:@mykitty
                       │ kitten @ launch --type=os-panel prismctl {instance}
              ┌────────┴────────┐
              │      kitty      │ <- wl_surface
              └────────┬────────┘
                      ↓│↑ stdio
              ┌────────┴────────┐
 /dev/ptmx -> │      pty_M      │
              └────────┬────────┘
                      ↓│↑
              ┌────────┴────────┐     ┌──────────────┐
/dev/pts/n -> │      pty_S      ├─────┤   prismctl   │ <- prism-{instance}.sock
              └─────────────────┘     └─┬────┬─────┬─┘
                                 ┌──────┘    │     └─────┐
                            ┌────┴─────┐┌────┴─────┐┌────┴─────┐
                            │   PTY1   ││   PTY2   ││  *PTY3   │ <- (* = foreground)
                            └────┬─────┘└────┬─────┘└────┬─────┘
                            ┌────┴─────┐┌────┴─────┐┌────┴─────┐
                            │  clock   ││  wabar   ││   app3   │ <- prisms/bin/
                            └──────────┘└──────────┘└──────────┘
```
