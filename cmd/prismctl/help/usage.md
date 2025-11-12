# prismctl - Prism Process Supervisor

Manages the lifecycle of a single prism TUI process with hot-swap capability.

## USAGE

```bash
prismctl <prism-name> [component-name]
```

## ARGUMENTS

```text
prism-name      Name of the prism binary to run (e.g., shine-clock)
component-name  Optional component identifier for IPC socket naming
                (default: same as prism-name)
```

## BEHAVIOR

prismctl provides:
- Terminal state management and cleanup
- Hot-swap capability via IPC commands
- Signal handling (SIGCHLD, SIGTERM, SIGWINCH)
- Process suspend/resume with SIGSTOP/SIGCONT
- MRU (Most Recently Used) ordering
- Crash recovery with restart policies

## IPC SOCKET

The IPC socket is created at:
```text
/run/user/{uid}/shine/prism-{instance}.sock
```

Where `{instance}` is the prism name (second argument, defaults to prism-name).

## EXAMPLES

```bash
$ prismctl shine-clock
```

```bash
$ prismctl shine-spotify music-panel
```

```bash
$ echo '{"action":"status"}' | socat - UNIX-CONNECT:/run/user/$(id -u)/shine/prism-*.sock
```

## FILES

```text
Logs:    ~/.local/share/shine/logs/prismctl.log
Sockets: /run/user/{uid}/shine/prism-*.sock
```

## LEARN MORE
  Use `prismctl help ipc` for IPC command reference.
  Use `prismctl help signals` for signal handling details.
  See the Shine documentation for full reference.
