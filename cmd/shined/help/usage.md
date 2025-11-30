# shined - Shine Service Manager

Background service that spawns and manages prismctl panel supervisors.

## USAGE

```bash
shined [options]
```

## OPTIONS

```text
-config PATH    Path to shine.toml (default: ~/.config/shine/shine.toml)
-version        Print version and exit
-help           Show this help message
```

## BEHAVIOR

shined is a long-running daemon that:
- Reads configuration from shine.toml
- Spawns Kitty panels via remote control API
- Launches prismctl supervisors for each panel
- Monitors panel health (30-second interval)
- Handles configuration reloads via SIGHUP

## SIGNALS

```text
SIGHUP          Reload configuration and update panels
SIGTERM/SIGINT  Graceful shutdown of all panels
```

## EXAMPLES

```bash
$ shined
```

```bash
$ shined -config ~/.config/shine/my-config.toml
```

```bash
$ pkill -HUP shined
```

## FILES

```text
Config:  ~/.config/shine/shine.toml
Logs:    ~/.local/share/shine/logs/shined.log
Sockets: /run/user/{uid}/shine/prism-*.sock
```

## LEARN MORE
  Use `shine start` to launch shined as a service.
  Use `shine logs shined` to view logs.
  See shine.toml documentation for configuration options.
