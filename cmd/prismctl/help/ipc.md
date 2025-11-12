# prismctl ipc

IPC protocol and command reference for hot-swapping prisms.

## USAGE

```bash
echo '{"action":"<action>"}' | socat - UNIX-CONNECT:<socket-path>
```

## IPC COMMANDS

### start

Start or resume a prism (idempotent).

```json
{"action":"start","prism":"shine-clock"}
```

Behavior:
- If prism is already running, brings it to foreground
- If different prism is running, suspends it and starts new one
- If no prism is running, starts the specified prism

### kill

Kill the current prism and resume next in MRU list.

```json
{"action":"kill","prism":"shine-clock"}
```

Behavior:
- Terminates the specified prism process
- Automatically resumes most recently used prism
- Removes killed prism from MRU list

### status

Query current supervisor status.

```json
{"action":"status"}
```

Response:
```json
{
  "success": true,
  "foreground": "shine-clock",
  "background": ["shine-chat", "shine-spotify"]
}
```

### stop

Graceful shutdown of prismctl supervisor.

```json
{"action":"stop"}
```

Behavior:
- Terminates foreground prism
- Terminates all background prisms
- Closes IPC socket
- Exits prismctl process

## EXAMPLES

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-clock.sock
$ echo '{"action":"status"}' | socat - UNIX-CONNECT:$SOCK
```

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-bar.sock
$ echo '{"action":"start","prism":"shine-spotify"}' | socat - UNIX-CONNECT:$SOCK
```

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-chat.sock
$ echo '{"action":"kill","prism":"shine-chat"}' | socat - UNIX-CONNECT:$SOCK
```

## SOCKET PATHS

Socket paths are deterministic based on prism name (instance):

```bash
# Direct path by prism name
SOCK=/run/user/$(id -u)/shine/prism-clock.sock

# Or list all prismctl sockets
ls /run/user/$(id -u)/shine/prism-*.sock
```

Each prism instance has a unique socket. When prismctl restarts, the old socket
is removed and a new one is created at the same path.

## LEARN MORE
  Use `prismctl help usage` for main usage.
  Use `prismctl help signals` for signal handling.
  See IPC protocol documentation for advanced usage.
