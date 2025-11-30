# prismctl ipc

IPC protocol and command reference for hot-swapping prisms.

## USAGE

prismctl uses JSON-RPC 2.0 over Unix sockets for all IPC communication.

```bash
echo '{"jsonrpc":"2.0","method":"<method>","params":{...},"id":1}' | socat - UNIX-CONNECT:<socket-path>
```

## RPC METHODS

### prism/up

Start or bring a prism to foreground (idempotent).

**Request:**
```json
{"jsonrpc":"2.0","method":"prism/up","params":{"name":"shine-clock"},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":{"pid":12345,"state":"fg"},"id":1}
```

Behavior:
- If prism is not running, starts it in foreground
- If prism is running in background, brings it to foreground
- If prism is already foreground, no-op (idempotent)
- If different prism is foreground, suspends it and switches to requested prism

### prism/down

Stop a prism and remove it from the MRU list.

**Request:**
```json
{"jsonrpc":"2.0","method":"prism/down","params":{"name":"shine-clock"},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":{"stopped":true},"id":1}
```

Behavior:
- Sends SIGTERM to prism process
- Waits 20ms for graceful shutdown, then sends SIGKILL
- Removes prism from MRU list
- If was foreground, automatically brings next MRU prism to foreground

### prism/fg

Bring a running prism to foreground.

**Request:**
```json
{"jsonrpc":"2.0","method":"prism/fg","params":{"name":"shine-chat"},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":{"ok":true,"was_fg":false},"id":1}
```

Behavior:
- Switches I/O relay to specified prism
- Suspends current foreground prism (SIGSTOP)
- Resumes target prism (SIGCONT)
- Returns `was_fg:true` if prism was already foreground (idempotent)

### prism/bg

Send a foreground prism to background.

**Request:**
```json
{"jsonrpc":"2.0","method":"prism/bg","params":{"name":"shine-clock"},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":{"ok":true,"was_bg":false},"id":1}
```

Behavior:
- Suspends prism with SIGSTOP
- Disconnects I/O relay
- Returns `was_bg:true` if prism was already background (idempotent)

### prism/list

List all managed prisms and their states.

**Request:**
```json
{"jsonrpc":"2.0","method":"prism/list","params":{},"id":1}
```

**Response:**
```json
{
  "jsonrpc":"2.0",
  "result":{
    "prisms":[
      {"name":"shine-clock","pid":12345,"state":"fg","uptime_ms":5432100,"restarts":0},
      {"name":"shine-chat","pid":12346,"state":"bg","uptime_ms":3210000,"restarts":1}
    ]
  },
  "id":1
}
```

### service/health

Check supervisor health status.

**Request:**
```json
{"jsonrpc":"2.0","method":"service/health","params":{},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":{"healthy":true,"prism_count":3},"id":1}
```

### service/shutdown

Graceful shutdown of prismctl supervisor.

**Request:**
```json
{"jsonrpc":"2.0","method":"service/shutdown","params":{"graceful":true},"id":1}
```

**Response:**
```json
{"jsonrpc":"2.0","result":{"shutting_down":true},"id":1}
```

Behavior:
- Terminates foreground prism (SIGTERM)
- Terminates all background prisms (SIGTERM)
- Closes IPC socket
- Exits prismctl process

## EXAMPLES

### Check supervisor health

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-clock.sock
$ echo '{"jsonrpc":"2.0","method":"service/health","params":{},"id":1}' | socat - UNIX-CONNECT:$SOCK
```

### Start a prism

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-bar.sock
$ echo '{"jsonrpc":"2.0","method":"prism/up","params":{"name":"shine-spotify"},"id":1}' | socat - UNIX-CONNECT:$SOCK
```

### List all prisms

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-chat.sock
$ echo '{"jsonrpc":"2.0","method":"prism/list","params":{},"id":1}' | socat - UNIX-CONNECT:$SOCK
```

### Stop a prism

```bash
$ SOCK=/run/user/$(id -u)/shine/prism-clock.sock
$ echo '{"jsonrpc":"2.0","method":"prism/down","params":{"name":"shine-clock"},"id":1}' | socat - UNIX-CONNECT:$SOCK
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
