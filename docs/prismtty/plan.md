# `prismctl` Architecture - Shine Implementation Plan

## Executive Summary

**Validated Architecture**: Expert agents (kitty-kat, charm-dev) have confirmed the design is viable and performant.

**Critical Finding**: Terminal state management is the #1 technical requirement. All other aspects are straightforward.

**Timeline Estimate**:

- Phase 1 (MVP): 2-3 days
- Phase 2 (Production): 2-3 days
- Phase 3 (Polish): 1-2 days

---

## Phase 1: MVP Foundation (Days 1-3)

### Goal: Prove the core supervisor pattern works

**Deliverables:**

1. `prismctl` binary with terminal state management
2. Hot-swap capability via IPC
3. Basic signal handling (SIGCHLD, SIGTERM, SIGWINCH)
4. Manual testing proof-of-concept

**Implementation Priority:**

```
cmd/prismctl/
├── main.go                    # Entry point, signal setup
├── supervisor.go              # Core supervisor logic
├── terminal.go                # Terminal state reset (CRITICAL)
├── ipc.go                     # Unix socket IPC server
└── signals.go                 # Signal handling (SIGCHLD, SIGTERM, SIGWINCH)
```

**Critical Code Requirements:**

1. **Terminal State Reset** (highest priority)

   ```go
   func (p *Prismctl) resetTerminalState() error {
       // Reset termios to canonical mode
       // Send visual reset sequences
       // Must be called after EVERY child exit
   }
   ```

2. **Sequential Child Handling**

   ```go
   func (p *Prismctl) hotSwap(newBinary string) error {
       // 1. SIGTERM old child
       // 2. Wait for exit (5s timeout, then SIGKILL)
       // 3. CRITICAL: resetTerminalState()
       // 4. 10ms stabilization delay
       // 5. fork/exec new child
   }
   ```

3. **Signal Orchestration**
   ```go
   // Setup before first child
   signal.Notify(sigCh, syscall.SIGCHLD, syscall.SIGTERM, syscall.SIGWINCH)
   ```

**Test Criteria:**

- [ ] Launch prismctl → shine-clock renders correctly
- [ ] Hot-swap clock → spotify → no visual corruption
- [ ] Rapid swaps (10x in 10s) → stable
- [ ] Kill child with SIGKILL → prismctl recovers
- [ ] Close Kitty panel → clean shutdown

**Success Metric**: All 5 test cases pass reliably

---

## Phase 2: Integration with shinectl (Days 4-6)

### Goal: Connect the control layer

**Deliverables:**

1. `shinectl` service manager
2. Config parsing for prism.toml
3. Panel lifecycle management via Kitty remote control
4. IPC between shinectl ↔ prismctl

**Implementation Priority:**

```
cmd/shinectl/
├── main.go                    # Service daemon
├── config.go                  # prism.toml parser (reuse existing)
├── panel_manager.go           # Kitty panel lifecycle
└── ipc_client.go              # Talk to prismctl instances

pkg/ipc/
└── protocol.go                # Shared IPC protocol
```

**Key Features:**

1. **Panel Spawning**

   ```go
   func (s *Shinectl) spawnPanel(prism PrismConfig) (panelID int, err error) {
       // kitten @ launch --type=window prismctl <prism-name>
       // Parse window ID from output
       // Store panelID → prismctl socket mapping
   }
   ```

2. **IPC Protocol**

   ```go
   type IPCCommand struct {
       Action string `json:"action"` // "start", "stop", "status"
       Prism  string `json:"prism"`  // Binary path
   }
   ```

3. **Config Hot-Reload**
   ```go
   // shinectl receives SIGHUP → reload config → update panels
   ```

**Test Criteria:**

- [ ] shinectl spawns 3 panels from config
- [ ] Each panel runs correct prism
- [ ] Hot-reload config changes panels
- [ ] shinectl restart preserves panels (Kitty keeps them alive)

**Success Metric**: `shine start` → 3 working panels with correct prisms

---

## Phase 3: User Experience & Polish (Days 7-9)

### Goal: Production-ready CLI and features

**Deliverables:**

1. `shine` user-facing CLI
2. Crash recovery with auto-restart
3. Health monitoring and logging
4. Documentation

**Implementation Priority:**

```
cmd/shine/
├── main.go                    # CLI entry point
├── commands.go                # start, stop, reload, status
└── output.go                  # Rich CLI output

pkg/supervisor/
└── health.go                  # Health checks, metrics
```

**Features:**

1. **User CLI**

   ```bash
   shine start              # Start/resume service
   shine stop               # Graceful shutdown
   shine reload             # Hot-reload config
   shine status             # Show panel status
   shine logs [panel-id]    # View panel logs
   ```

2. **Crash Recovery**

   ```go
   func (p *Prismctl) monitorChild() {
       exitCode := <-p.childExit
       if exitCode != 0 && p.config.AutoRestart {
           time.Sleep(p.config.RestartDelay)
           p.forkExec(p.currentPrism)
       }
   }
   ```

3. **Audit Logging**
   ```go
   // Log to ~/.local/share/shine/logs/panel-<id>.log
   // Format: [timestamp] [level] event: data
   ```

**Test Criteria:**

- [ ] All shine commands work
- [ ] Crash recovery restarts prism
- [ ] Logs are readable and useful
- [ ] `shine status` shows accurate info

**Success Metric**: Production-ready for daily use

---

## Critical Path Dependencies

```
Phase 1: prismctl (standalone)
         ↓
Phase 2: shinectl integration
         ↓
Phase 3: shine CLI + polish
```

**Blockers:**

- Phase 2 depends on Phase 1 terminal state reset working perfectly
- Phase 3 can partially parallelize with Phase 2

---

## Risk Mitigation

| Risk                            | Impact | Mitigation                                        |
| ------------------------------- | ------ | ------------------------------------------------- |
| Terminal corruption after crash | High   | Comprehensive reset function (provided by agents) |
| Race conditions in hot-swap     | Medium | Serialize all swaps, single state machine         |
| Zombie processes                | Medium | Robust SIGCHLD handling with unix.Wait4()         |
| IPC socket conflicts            | Low    | Use PID in socket name, cleanup on exit           |

---

## Technical Specifications

### prismctl Requirements (CRITICAL)

**Must implement:**

1. Terminal state reset function (from checklist lines 40-66)
2. SIGCHLD handling with zombie reaping
3. SIGTERM forwarding to child
4. SIGWINCH forwarding to child
5. Sequential hot-swap with stabilization delay
6. 5-second grace period for SIGTERM before SIGKILL

**Code Template Provided**: Yes (lines 265-295 in checklist)

### Prism Development Guidelines

**Template for all prisms:**

```go
func main() {
    p := tea.NewProgram(model{})
    if _, err := p.Run(); err != nil {
        os.Exit(1)
    }
    os.Exit(0)
}
```

**DO NOT:**

- Use `tea.WithoutCatchPanics()`
- Use `tea.WithoutSignalHandler()`
- Add custom signal handlers

---

## Success Criteria

### Phase 1 Complete When:

- ✅ 5/5 test cases pass
- ✅ No terminal corruption in any scenario
- ✅ Hot-swap latency <100ms

### Phase 2 Complete When:

- ✅ shinectl manages 3+ panels correctly
- ✅ Config hot-reload works
- ✅ IPC communication is reliable

### Phase 3 Complete When:

- ✅ All shine commands functional
- ✅ Crash recovery tested and working
- ✅ Documentation complete

---

## Immediate Next Actions

1. **Create prismctl skeleton** (30 min)
   - `cmd/prismctl/main.go` with argument parsing
   - Basic logging setup

2. **Implement terminal.go** (2 hours)
   - Copy reset function from checklist
   - Test in isolation with test script

3. **Implement supervisor.go** (4 hours)
   - Child process management
   - Hot-swap logic with terminal reset
   - Basic error handling

4. **Implement signals.go** (2 hours)
   - SIGCHLD, SIGTERM, SIGWINCH handlers
   - Signal channel setup

5. **Integration test** (1 hour)
   - Manual testing with shine-clock
   - Verify terminal state across swaps

**Day 1 Goal**: Working prismctl with manual hot-swap capability

---

## Design Decisions ✅ RESOLVED

### 1. Config Format - Binary Resolution
**Decision**: Reference binaries **by name**, utilize existing PATH lookup logic to acquire executable path.

**Rationale**:
- Simplifies prism.toml configuration
- Leverages standard Unix PATH mechanism
- Allows prisms to be installed anywhere in PATH
- Consistent with user expectations (`shine-clock` vs `/usr/bin/shine-clock`)

**Implementation**: Use `exec.LookPath(prismName)` before fork/exec.

---

### 2. Socket Naming Convention
**Decision**: Use XDG runtime directory with structured naming:

```
/run/user/{uid}/shine/shine-{component}.{pid}.sock
/run/user/{uid}/shine/prism-{component}.{pid}.sock
```

**Examples**:
- `shinectl` main service: `/run/user/1000/shine/shine-service.1234.sock`
- `prismctl` for bar panel: `/run/user/1000/shine/prism-bar.5678.sock`
- `prismctl` for panel 2: `/run/user/1000/shine/prism-panel.9012.sock`

**Rationale**:
- `/run/user/{uid}` is XDG standard for runtime files (systemd user services)
- Automatic cleanup on logout (tmpfs)
- Per-user isolation
- PID suffix prevents conflicts on restart
- Component name allows multiple instances with semantic naming

**Implementation**:
```go
socketPath := fmt.Sprintf("/run/user/%d/shine/%s-%s.%d.sock",
    os.Getuid(), "prism", component, os.Getpid())
```

---

### 3. Logging Strategy
**Decision**: Dual logging approach:

**Per-Prism Logs**:
- Individual file per prism: `~/.local/share/shine/logs/prism-{component}-{pid}.log`
- OR systemd user journal with identifier (if available)
- Captures prism-specific output and errors

**Shine Service Log**:
- Central log: `~/.local/share/shine/logs/shine.log`
- OR systemd user journal for shinectl
- Captures service-level events (panel launches, config reloads, IPC events)

**Format**: Structured JSON lines for easy parsing
```json
{"timestamp":"2025-11-07T16:00:00Z","level":"info","component":"prism-bar","event":"child_started","pid":3000}
```

**Rationale**:
- Separate logs prevent mixing of concerns
- Individual prism logs aid debugging specific widgets
- Central log provides system overview
- Journal integration for systemd users
- JSON format enables log aggregation tools

---

### 4. Auto-Restart Configuration
**Decision**: Opt-in per prism, Docker Compose-style restart policies.

**prism.toml Format**:
```toml
[[prism]]
name = "shine-clock"
restart = "unless-stopped"  # always | on-failure | unless-stopped | no

[[prism]]
name = "shine-spotify"
restart = "on-failure"
restart_delay = "5s"
max_restarts = 10
```

**Restart Policies**:
- `no`: Never restart (default)
- `on-failure`: Restart only if exit code != 0
- `unless-stopped`: Always restart unless explicitly stopped via IPC
- `always`: Restart unconditionally (even on clean exit)

**Additional Options**:
- `restart_delay`: Time to wait before restart (default: 1s)
- `max_restarts`: Maximum restart attempts per hour (default: unlimited)

**Rationale**:
- Familiar to users of Docker/systemd
- Flexibility for different prism behaviors
- Safe default (no auto-restart)
- Prevents restart loops with max_restarts

---

## Implementation Notes

### Socket Directory Setup
```go
func ensureSocketDir() error {
    uid := os.Getuid()
    dir := fmt.Sprintf("/run/user/%d/shine", uid)
    return os.MkdirAll(dir, 0700)  // User-only permissions
}
```

### Log Directory Setup
```go
func ensureLogDir() error {
    dir := filepath.Join(os.Getenv("HOME"), ".local/share/shine/logs")
    return os.MkdirAll(dir, 0755)
}
```

### Binary Resolution
```go
func resolvePrismBinary(name string) (string, error) {
    // Use PATH lookup
    path, err := exec.LookPath(name)
    if err != nil {
        return "", fmt.Errorf("prism not found in PATH: %s", name)
    }
    return path, nil
}
```

---

## Review Checkpoint

**All design decisions resolved.** Ready to proceed with implementation.

**Phase 1 approved**: Begin prismctl development immediately.

**Timeline confirmed**: 6-9 days to production-ready system.

_Tokens: Input: 10 | Output: 2596 | Cache Creation: 4866 | Cache Read: 117778_
