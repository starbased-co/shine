# Code Tour: prismctl IPC System

## Overview

The IPC (Inter-Process Communication) system in `prismctl` provides a **Unix socket-based JSON protocol** for controlling prism supervisors at runtime. This is the **primary mechanism** for external tools (like `shine` CLI or `shinectl`) to interact with running prism instances.

**Key Design Principles:**

- **Deterministic socket paths**: `/run/user/{uid}/shine/prism-{instance}.sock`
- **Line-delimited JSON**: Simple, debuggable protocol
- **Synchronous request/response**: One command per connection
- **Thread-safe supervisor access**: All operations lock the supervisor mutex

## Architecture

```
External Client (shine CLI, shinectl)
         │
         │ Unix Socket Connection
         ⮟
    ipcServer.serve() ← listener loop
         │
         │ Accept connection
         ⮟
    handleConnection() ← parse JSON command
         │
         ├─🠶 handleStart()   ─→ supervisor.start()
         ├─🠶 handleKill()    ─→ supervisor.killPrism()
         ├─🠶 handleStatus()  ─→ supervisor state (locked)
         └─🠶 handleStop()    ─→ supervisor.shutdown()
```

## Core Data Structures

### Command Protocol

[`cmd/prismctl/ipc.go#L15-19`](cmd/prismctl/ipc.go#L15-19)

```go
// ipcCommand represents a command sent via IPC
type ipcCommand struct {
    Action string `json:"action"` // "start", "kill", "status", "stop"
    Prism  string `json:"prism"`  // Prism name for start/kill action
}
```

**Keep in Mind:**

- Only 4 actions supported: `start`, `kill`, `status`, `stop`
- `Prism` field is **required** for `start` and `kill`, **ignored** for others
- Unknown actions return error response (no silent failures)

### Response Protocol

[`cmd/prismctl/ipc.go#L22-26`](cmd/prismctl/ipc.go#L22-26)

```go
// ipcResponse represents a response from IPC
type ipcResponse struct {
    Success bool   `json:"success"`
    Message string `json:"message"`
    Data    any    `json:"data,omitempty"` // Polymorphic data field
}
```

**Design Decision:** The `Data` field uses `any` (interface{}) to support different response types:

- `status` action → `statusResponse` struct
- Other actions → typically `nil`

### Status Response Structure

[`cmd/prismctl/ipc.go#L28-40`](cmd/prismctl/ipc.go#L28-40)

```go
// statusResponse represents the status command response
type statusResponse struct {
    Foreground string        `json:"foreground"`  // Current visible prism
    Background []string      `json:"background"`  // All backgrounded prisms
    Prisms     []prismStatus `json:"prisms"`      // Detailed per-prism info
}

// prismStatus represents individual prism status
type prismStatus struct {
    Name  string `json:"name"`
    PID   int    `json:"pid"`
    State string `json:"state"` // "foreground" or "background"
}
```

**Data Model:** Status provides **three views** of the same data:

1. **Foreground** (string): Quick access to active prism name
2. **Background** ([]string): List of backgrounded prism names
3. **Prisms** ([]prismStatus): Full details including PIDs

This redundancy optimizes for different client use cases without multiple roundtrips.

## Server Lifecycle

### Initialization

[`cmd/prismctl/ipc.go#L52-91`](cmd/prismctl/ipc.go#L52-91)

```go
func newIPCServer(instance string, supervisor *supervisor) (*ipcServer, error) {
    // Use XDG runtime directory
    uid := os.Getuid()
    runtimeDir := fmt.Sprintf("/run/user/%d/shine", uid)

    // Create directory if needed
    if err := os.MkdirAll(runtimeDir, 0700); err != nil {
        return nil, fmt.Errorf("failed to create runtime directory: %w", err)
    }

    // Extract basename from instance path (handles "./test/fixtures/test-prism" -> "test-prism")
    instanceName := filepath.Base(instance)

    // Socket path
    socketPath := filepath.Join(runtimeDir, fmt.Sprintf("prism-%s.sock", instanceName))

    // Remove stale socket if exists
    _ = os.Remove(socketPath)

    // Create Unix socket listener
    listener, err := net.Listen("unix", socketPath)
    if err != nil {
        return nil, fmt.Errorf("failed to create Unix socket: %w", err)
    }

    // Set socket permissions to user-only
    if err := os.Chmod(socketPath, 0600); err != nil {
        listener.Close()
        return nil, fmt.Errorf("failed to set socket permissions: %w", err)
    }

    log.Printf("IPC server listening on: %s", socketPath)

    return &ipcServer{
        socketPath: socketPath,
        listener:   listener,
        supervisor: supervisor,
        stopCh:     make(chan struct{}),
    }, nil
}
```

**Critical Implementation Details:**

1. **Socket Path Derivation:**
   - Uses `filepath.Base(instance)` to extract prism name from paths
   - Example: `./test/fixtures/clock` → socket at `/run/user/1000/shine/prism-clock.sock`
   - This makes socket paths **deterministic** and **discoverable**

2. **Security:**
   - Runtime directory: `0700` permissions (user-only)
   - Socket file: `0600` permissions (user read/write only)
   - No group or world access to IPC

3. **Stale Socket Handling:**
   - Always removes existing socket before binding (Line 69)
   - Prevents "address already in use" errors on restart
   - Safe because sockets are namespaced by UID

4. **Error Handling:**
   - MkdirAll is **not fatal** if directory exists
   - Listener cleanup on chmod failure (defensive programming)

### Server Loop

[`cmd/prismctl/ipc.go#L93-120`](cmd/prismctl/ipc.go#L93-120)

```go
// serve starts accepting IPC connections
func (s *ipcServer) serve() {
    s.wg.Add(1)
    defer s.wg.Done()

    for {
        select {
        case <-s.stopCh:
            return
        default:
        }

        conn, err := s.listener.Accept()
        if err != nil {
            select {
            case <-s.stopCh:
                return // Graceful shutdown
            default:
                log.Printf("Error accepting connection: %v", err)
                continue // Log and continue on transient errors
            }
        }

        // Handle connection in goroutine
        s.wg.Add(1)
        go s.handleConnection(conn)
    }
}
```

**Concurrency Model:**

- **Main loop**: Single-threaded accept loop
- **Connection handling**: Each connection gets its own goroutine
- **WaitGroup tracking**: Ensures all handlers complete before shutdown
- **Graceful shutdown**: `stopCh` signals termination, `wg.Wait()` ensures cleanup

> **⚠️ CRITICAL**: The nested `select` on `stopCh` (Lines 107-110) is **essential**. It prevents error spam during shutdown when `listener.Close()` causes `Accept()` to fail.

## Connection Handling

### Request Processing

[`cmd/prismctl/ipc.go#L122-154`](cmd/prismctl/ipc.go#L122-154)

```go
// handleConnection processes a single IPC connection
func (s *ipcServer) handleConnection(conn net.Conn) {
    defer s.wg.Done()
    defer conn.Close()

    reader := bufio.NewReader(conn)
    line, err := reader.ReadString('\n')
    if err != nil {
        s.sendError(conn, "failed to read command")
        return
    }

    // Parse command
    var cmd ipcCommand
    if err := json.Unmarshal([]byte(strings.TrimSpace(line)), &cmd); err != nil {
        s.sendError(conn, fmt.Sprintf("invalid JSON: %v", err))
        return
    }

    // Process command
    switch cmd.Action {
    case "start":
        s.handleStart(conn, cmd)
    case "kill":
        s.handleKill(conn, cmd)
    case "status":
        s.handleStatus(conn)
    case "stop":
        s.handleStop(conn)
    default:
        s.sendError(conn, fmt.Sprintf("unknown action: %s", cmd.Action))
    }
}
```

**Protocol Design:**

1. **Line-delimited JSON**: Read until `\n` (enables simple debugging with `echo | nc`)
2. **Single command per connection**: No streaming, no pipelining
3. **Fail-fast validation**: JSON parse errors return immediately
4. **Explicit error messages**: Include parse error details for debugging

**Why line-delimited?**

- Debuggable: `echo '{"action":"status"}' | nc -U /path/to/socket`
- Simple: No length prefixes or chunking needed
- Sufficient: Commands are small (<1KB typically)

## Command Handlers

### Start Handler (Idempotent Launch/Resume)

[`cmd/prismctl/ipc.go#L156-171`](cmd/prismctl/ipc.go#L156-171)

```go
// handleStart processes a start command (idempotent launch/resume)
func (s *ipcServer) handleStart(conn net.Conn, cmd ipcCommand) {
    if cmd.Prism == "" {
        s.sendError(conn, "prism name required for start action")
        return
    }

    log.Printf("IPC: Received start request for %s", cmd.Prism)

    if err := s.supervisor.start(cmd.Prism); err != nil {
        s.sendError(conn, fmt.Sprintf("start failed: %v", err))
        return
    }

    s.sendSuccess(conn, "prism started/resumed", nil)
}
```

**Idempotency:** The `supervisor.start()` method handles both:

- **Launch**: Fork new process if prism not running
- **Resume**: Switch surface to existing prism if already running

This makes the `start` command safe to call repeatedly without side effects.

### Kill Handler

[`cmd/prismctl/ipc.go#L173-188`](cmd/prismctl/ipc.go#L173-188)

```go
// handleKill processes a kill command
func (s *ipcServer) handleKill(conn net.Conn, cmd ipcCommand) {
    if cmd.Prism == "" {
        s.sendError(conn, "prism name required for kill action")
        return
    }

    log.Printf("IPC: Received kill request for %s", cmd.Prism)

    if err := s.supervisor.killPrism(cmd.Prism); err != nil {
        s.sendError(conn, fmt.Sprintf("kill failed: %v", err))
        return
    }

    s.sendSuccess(conn, "prism killed", nil)
}
```

**Termination Semantics:**

- Sends `SIGTERM` to target prism process
- Removes from supervisor's prism list
- If killed prism was foreground, switches to next in MRU order

See [`cmd/prismctl/supervisor.go`](cmd/prismctl/supervisor.go) for full termination logic.

### Status Handler (Read-Only Query)

[`cmd/prismctl/ipc.go#L190-223`](cmd/prismctl/ipc.go#L190-223)

```go
// handleStatus processes a status query
func (s *ipcServer) handleStatus(conn net.Conn) {
    s.supervisor.mu.Lock()
    defer s.supervisor.mu.Unlock()

    // Build status response
    foreground := ""
    background := []string{}
    prisms := []prismStatus{}

    for i, p := range s.supervisor.prismList {
        state := "background"
        if i == 0 {
            state = "foreground"
            foreground = p.name
        } else {
            background = append(background, p.name)
        }

        prisms = append(prisms, prismStatus{
            Name:  p.name,
            PID:   p.pid,
            State: state,
        })
    }

    data := statusResponse{
        Foreground: foreground,
        Background: background,
        Prisms:     prisms,
    }

    s.sendSuccess(conn, "status ok", data)
}
```

**Thread Safety:**

- Acquires supervisor mutex for consistent snapshot
- Reads `prismList` (MRU-ordered slice)
- **Index 0 = foreground** (architectural invariant)

> **⚠️ IMPORTANT**: This handler **must not** call other supervisor methods while holding the lock. The lock is only for reading `prismList` safely.

**MRU Invariant:**
The supervisor maintains `prismList` in MRU (Most Recently Used) order:

- Index 0: Foreground (currently visible)
- Index 1+: Background (in order of last use)

This invariant is **critical** for status reporting and surface switching.

### Stop Handler (Graceful Shutdown)

[`cmd/prismctl/ipc.go#L225-232`](cmd/prismctl/ipc.go#L225-232)

```go
// handleStop processes a stop command
func (s *ipcServer) handleStop(conn net.Conn) {
    log.Printf("IPC: Received stop request")
    s.sendSuccess(conn, "stopping", nil)

    // Trigger shutdown
    go s.supervisor.shutdown()
}
```

**Shutdown Sequence:**

1. Send success response **before** initiating shutdown (client doesn't wait)
2. Launch shutdown in goroutine (non-blocking)
3. Supervisor shutdown will eventually call `ipcServer.stop()`

**Why goroutine?**

- Prevents deadlock: shutdown may need to close this connection
- Allows response to be sent before supervisor begins termination
- Client gets acknowledgment immediately

## Response Helpers

### Success Response

[`cmd/prismctl/ipc.go#L234-242`](cmd/prismctl/ipc.go#L234-242)

```go
// sendSuccess sends a success response
func (s *ipcServer) sendSuccess(conn net.Conn, message string, data any) {
    resp := ipcResponse{
        Success: true,
        Message: message,
        Data:    data,
    }
    s.sendResponse(conn, resp)
}
```

### Error Response

[`cmd/prismctl/ipc.go#L244-251`](cmd/prismctl/ipc.go#L244-251)

```go
// sendError sends an error response
func (s *ipcServer) sendError(conn net.Conn, message string) {
    resp := ipcResponse{
        Success: false,
        Message: message,
    }
    s.sendResponse(conn, resp)
}
```

### JSON Marshaling

[`cmd/prismctl/ipc.go#L253-265`](cmd/prismctl/ipc.go#L253-265)

```go
// sendResponse sends a JSON response
func (s *ipcServer) sendResponse(conn net.Conn, resp ipcResponse) {
    data, err := json.Marshal(resp)
    if err != nil {
        log.Printf("Error marshaling response: %v", err)
        return
    }

    data = append(data, '\n') // Line-delimited protocol
    if _, err := conn.Write(data); err != nil {
        log.Printf("Error writing response: %v", err)
    }
}
```

**Error Handling Strategy:**

- Marshal errors: Log only (client gets disconnected, no partial response)
- Write errors: Log only (connection might be broken)
- No retries or buffering (single-shot protocol)

**Line Delimiter:** The `\n` appended at Line 261 completes the line-delimited protocol, enabling clients to use `ReadString('\n')` or similar.

## Server Shutdown

[`cmd/prismctl/ipc.go#L267-277`](cmd/prismctl/ipc.go#L267-277)

```go
// stop stops the IPC server
func (s *ipcServer) stop() {
    close(s.stopCh)       // Signal serve() loop to exit
    s.listener.Close()    // Break Accept() call
    s.wg.Wait()           // Wait for all handlers to complete

    // Clean up socket file
    _ = os.Remove(s.socketPath)

    log.Printf("IPC server stopped")
}
```

**Shutdown Sequence:**

1. **Close stopCh**: Signals main loop to exit (breaks Line 100 select)
2. **Close listener**: Causes pending `Accept()` to return error (unblocks Line 105)
3. **Wait for handlers**: All goroutines spawned by `serve()` complete
4. **Remove socket**: Cleanup filesystem (best-effort, ignore errors)

**Correctness:**

- No new connections accepted after `stopCh` close
- Existing handlers run to completion
- Socket file removed before function returns

> **⚠️ CRITICAL**: The order is **essential**. Closing listener before stopCh would cause error logs. Removing socket before wg.Wait() could allow reconnection.

## Integration Points

### Supervisor Interaction

The IPC server is a **thin adapter** over the supervisor:

```
IPC Command         Supervisor Method           Mutex Required?
-----------         -----------------           ---------------
start              → supervisor.start()         ✓ (internal)
kill               → supervisor.killPrism()     ✓ (internal)
status             → supervisor.mu.Lock()       ✓ (explicit)
stop               → supervisor.shutdown()      ✓ (internal)
```

All supervisor methods handle their own locking **except** `handleStatus()`, which must manually lock to read `prismList`.

### Socket Discovery

External clients discover sockets via:

1. **By prism name** (deterministic):

   ```bash
   SOCK=/run/user/$(id -u)/shine/prism-clock.sock
   ```

2. **By glob pattern** (enumeration):

   ```bash
   ls /run/user/$(id -u)/shine/prism-*.sock
   ```

3. **Via `shine` CLI** (abstraction):

   ```bash
   shine status        # Queries all prism sockets
   shine start clock   # Queries clock-specific socket
   ```

## Testing Patterns

### Manual Testing

```bash
# Start command
echo '{"action":"start","prism":"clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.sock

# Status query
echo '{"action":"status"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.sock

# Kill command
echo '{"action":"kill","prism":"clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.sock

# Stop supervisor
echo '{"action":"stop"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.sock
```

### Using socat (Alternative)

```bash
# socat provides better error handling
echo '{"action":"status"}' | socat - UNIX-CONNECT:/run/user/$(id -u)/shine/prism-panel-0.sock
```

### Automated Tests

See [`cmd/prismctl/ipc_test.go`](cmd/prismctl/ipc_test.go) (if exists) for unit tests.

**Test Strategy:**

- Mock supervisor for handler tests
- Use `net.Pipe()` for connection simulation
- Test malformed JSON, missing fields, unknown actions
- Verify response structure with `json.Unmarshal`

## Common Pitfalls

### 1. Socket Path Mismatch

**Problem:** Client uses wrong socket path (e.g., prism name vs. panel name).

**Solution:** Always derive socket path from **prism instance name**, not panel name:

```go
instanceName := filepath.Base(instance)  // Handles paths correctly
socketPath := filepath.Join(runtimeDir, fmt.Sprintf("prism-%s.sock", instanceName))
```

### 2. Missing Mutex Lock

**Problem:** Reading `supervisor.prismList` without locking in `handleStatus()`.

**Solution:** Always acquire `supervisor.mu.Lock()` before accessing shared state (Line 192).

### 3. Deadlock on Shutdown

**Problem:** Calling `supervisor.shutdown()` synchronously in `handleStop()`.

**Solution:** Use `go supervisor.shutdown()` to avoid blocking the response (Line 231).

### 4. Stale Socket Detection

**Problem:** Old socket file exists after crash, preventing bind.

**Solution:** Always `os.Remove(socketPath)` before `net.Listen()` (Line 69).

## Security Considerations

1. **UID Isolation**: Sockets in `/run/user/{uid}` prevent cross-user access
2. **File Permissions**: `0600` on socket (user-only read/write)
3. **Directory Permissions**: `0700` on runtime directory (user-only access)
4. **No Authentication**: Assumes Unix permissions are sufficient (same-user trust)
5. **Input Validation**: JSON schema enforced, unknown actions rejected

**Threat Model:** Trusts all processes running as the same UID. Does **not** defend against:

- Malicious processes running as the user
- Privilege escalation (relies on OS)
- Denial of service (unbounded connections)

## Future Enhancements

From the known limitations in CLAUDE.md:

1. **Authentication**: Add shared secret or capability tokens
2. **Rate Limiting**: Prevent DoS via connection floods
3. **Streaming Protocol**: Multi-command pipelining for batch operations
4. **Binary Protocol**: Consider Protocol Buffers or msgpack for efficiency
5. **Connection Pooling**: Reuse connections in `shine` CLI
6. **Async Notifications**: Publish events (prism started/stopped) to subscribers

## Summary

The IPC system provides a **simple, debuggable, secure** communication channel for controlling prism supervisors:

- ✅ **Deterministic**: Socket paths derived from prism names
- ✅ **Secure**: UID-isolated, file permissions enforced
- ✅ **Debuggable**: Line-delimited JSON, simple netcat testing
- ✅ **Thread-safe**: Proper mutex usage, WaitGroup coordination
- ✅ **Graceful**: Shutdown waits for handlers, removes socket
- ✅ **Idempotent**: `start` command safe to repeat

**Core Invariant:** Index 0 of `supervisor.prismList` is **always** the foreground prism (visible to user). All IPC operations preserve this invariant.

_Tokens: Input: 3 | Output: 5880 | Cache Creation: 48385_
