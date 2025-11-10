# Shine - Unified Vision & Roadmap (PTY-per-Process Architecture)

**Status**: Architectural Redesign - PTY Multiplexing
**Date**: 2025-11-08
**Current State**: SIGSTOP/SIGCONT supervisor (Phase 2/3 complete)
**Target State**: PTY-per-process multiplexer with background processing

---

## Executive Summary

Shine's current architecture uses **SIGSTOP/SIGCONT** for suspend/resume, which provides instant swaps but **freezes background processes completely**. This prevents useful background work like data fetching, state updates, and computation.

This document proposes **PTY-per-Process Multiplexing**: a kitty-native approach where each prism gets its own PTY, enabling:
- ✅ **Background processing** while not visible
- ✅ **Instant swap** (<10ms) by switching relay target
- ✅ **Full kitty compatibility** (graphics, keyboard, unicode)
- ✅ **Zero protocol translation** (unlike tmux)
- ✅ **State preservation** in-memory across swaps

**This is NOT tmux!** We avoid all the problems Kovid Goyal documented with tmux by being kitty-native (single terminal type, pure passthrough, zero translation).

---

## Vision Statement

**Shine will become a production-grade Kitty panel manager with PTY-per-process multiplexing, providing instant hot-swap between continuously-running prisms while maintaining full kitty protocol compatibility and zero terminal corruption.**

### Core Principles

1. **Kitty-Native**: Uses kitty's features, doesn't fight them (TERM=xterm-kitty passthrough)
2. **Zero Translation**: Dumb relay, not smart translation (avoids tmux's bugs)
3. **Background Processing**: All prisms run continuously (no SIGSTOP freezing)
4. **Instant Swap**: <10ms context switch by relay retargeting
5. **Zero Corruption**: Terminal state always pristine through reset protocol
6. **Single Terminal Type**: Kitty only (no multi-terminal translation nightmare)

---

## Why NOT tmux's Approach

### Kovid Goyal's Critique of tmux

From kitty issue discussions:

> "tmux cannot work *correctly* with multiple terminals. That is because it has to *translate* escape codes it receives into escape codes a particular terminal understands and different terminal understand different escape codes. tmux does not do that different translation."

**tmux problems**:
- Sets `TERM=tmux-256color` (hides kitty from apps)
- Translates escape codes per client terminal type (buggy)
- Breaks kitty keyboard protocol
- Breaks kitty graphics protocol
- Wastes CPU on protocol translation
- Causes rendering bugs

### How prismctl Differs

| **Aspect** | **tmux** | **prismctl** |
|------------|----------|--------------|
| **TERM variable** | `tmux-256color` | `xterm-kitty` (passthrough) |
| **Protocol translation** | YES (buggy) | NO (pure relay) |
| **Multi-terminal support** | YES (goal) | NO (kitty only) |
| **Escape code parsing** | YES | NO (zero interpretation) |
| **Kitty protocols** | Broken | Full support |
| **Philosophy** | Universal | Kitty-native |

**Key insight**: prismctl is a **dumb relay**, not a smart translator. We pass bytes unchanged from child PTY to real PTY. Zero parsing, zero modification, zero bugs.

---

## PTY-per-Process Architecture

### High-Level Diagram

```
┌─────────────────────────────────────────┐
│ Kitty Terminal Emulator                 │
│ TERM=xterm-kitty                        │
└──────────────┬──────────────────────────┘
               │ Real PTY (master/slave)
          ┌────┴─────┐
          │Real_PTY_M│ (kitty owns master)
          └────┬─────┘
               │ pty pair
          ┌────┴─────┐
          │Real_PTY_S│ (/dev/pts/5)
          └────┬─────┘
               │ inherited as stdin/stdout/stderr
┌──────────────┴──────────────────────────┐
│ prismctl (Supervisor + PTY Multiplexer) │
│ TERM=xterm-kitty (inherited)            │
│                                          │
│  ┌──────────────────────────┐           │
│  │ I/O Relay Loop           │           │
│  │ - Real_PTY_S ↔ FG_PTY_M  │           │
│  │ - Pure passthrough       │           │
│  │ - Zero translation       │           │
│  └──────────────────────────┘           │
│  ┌──────────────────────────┐           │
│  │ Terminal State Manager   │           │
│  │ - Saves initial termios  │           │
│  │ - Resets on swap         │           │
│  └──────────────────────────┘           │
│  ┌──────────────────────────┐           │
│  │ PTY Manager              │           │
│  │ - Allocates child PTYs   │           │
│  │ - Syncs terminal sizes   │           │
│  │ - Forwards SIGWINCH      │           │
│  └──────────────────────────┘           │
└─┬────────────┬────────────┬─────────────┘
  │ owns       │ owns       │ owns
  │            │            │
┌─┴────┐   ┌──┴────┐   ┌───┴────┐
│PTY1_M│   │PTY2_M │   │PTY3_M  │ ← FOREGROUND (relayed)
└─┬────┘   └──┬────┘   └───┬────┘
  │ pair      │ pair       │ pair
┌─┴────┐   ┌──┴────┐   ┌───┴────┐
│PTY1_S│   │PTY2_S │   │PTY3_S  │
└─┬────┘   └──┬────┘   └───┬────┘
  │ std*      │ std*       │ std*
┌─┴────────┐ ┌┴──────────┐ ┌┴──────────┐
│ clock    │ │ chat      │ │ weather   │
│ PID 123  │ │ PID 124   │ │ PID 125   │
│ RUNNING  │ │ RUNNING   │ │ RUNNING   │
│ isatty✓  │ │ isatty✓   │ │ isatty✓   │
│ TERM=    │ │ TERM=     │ │ TERM=     │
│ xterm-   │ │ xterm-    │ │ xterm-    │
│ kitty    │ │ kitty     │ │ kitty     │
└──────────┘ └───────────┘ └───────────┘
Background    Background    Foreground
processing    processing    (visible)
```

### Component Responsibilities (Updated)

**prismctl** (Supervisor + Multiplexer)
- Allocates PTY pair for each child prism
- Relays I/O between Real PTY and foreground child PTY
- Syncs terminal size to all child PTYs on resize
- Forwards SIGWINCH to foreground child
- Manages MRU list for swap ordering
- Resets terminal state on swap (prevents corruption)
- Zero escape code translation (pure passthrough)
- IPC server for swap/status commands

**Child Prisms** (Bubble Tea TUIs)
- Each gets dedicated PTY (appears as real terminal)
- `isatty()` returns true → renders normally
- Sees `TERM=xterm-kitty` (full kitty features)
- Keeps running when not foreground (background processing!)
- Receives SIGWINCH when foreground (for resize)

---

## How It Works: Technical Details

### 1. Child PTY Allocation

```go
func (s *supervisor) launchPrismWithPTY(prismName string) error {
    // Create dedicated PTY pair for this prism
    childPTY_M, childPTY_S, _ := pty.Open()

    // Sync terminal size from Real PTY
    realWinsize, _ := unix.IoctlGetWinsize(
        int(s.realPTY_S_FD),
        unix.TIOCGWINSZ,
    )
    unix.IoctlSetWinsize(childPTY_M, unix.TIOCSWINSZ, realWinsize)

    // Fork/exec with child PTY as stdin/stdout/stderr
    cmd := exec.Command(prismName)
    cmd.Stdin = childPTY_S
    cmd.Stdout = childPTY_S
    cmd.Stderr = childPTY_S
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid:  true,  // New session
        Setctty: true,  // Make PTY controlling terminal
        Ctty:    int(childPTY_S.Fd()),
    }

    cmd.Start()
    childPTY_S.Close()  // Parent keeps master only

    // Track instance
    instance := &prismInstance{
        name:      prismName,
        pid:       cmd.Process.Pid,
        ptyMaster: childPTY_M,
        state:     prismBackground,
    }

    s.prismList = append(s.prismList, instance)
    return nil
}
```

**Result**: Child sees real PTY, `isatty()` = true, `TERM=xterm-kitty`

### 2. I/O Relay Loop

```go
func (s *supervisor) relayLoop() {
    for {
        s.mu.Lock()
        fg := s.getCurrentForeground()
        s.mu.Unlock()

        if fg == nil {
            time.Sleep(10 * time.Millisecond)
            continue
        }

        // Bidirectional relay (dumb passthrough!)
        go io.Copy(fg.ptyMaster, os.Stdin)   // Real PTY → child
        go io.Copy(os.Stdout, fg.ptyMaster)  // child → Real PTY

        // Relay switches automatically when foreground changes
    }
}
```

**Result**: Bytes flow unchanged: kitty ↔ Real PTY ↔ prismctl ↔ child PTY ↔ prism

### 3. Hot-Swap (Instant!)

```go
func (s *supervisor) swapToForeground(targetIdx int) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    target := s.prismList[targetIdx]

    // 1. Update MRU list (move target to front)
    if len(s.prismList) > 0 {
        s.prismList[0].state = prismBackground
    }
    target.state = prismForeground

    s.prismList = append(
        s.prismList[:targetIdx],
        s.prismList[targetIdx+1:]...,
    )
    s.prismList = append([]prismInstance{target}, s.prismList...)

    // 2. Sync terminal size to new foreground
    realWinsize, _ := unix.IoctlGetWinsize(
        int(s.realPTY_S_FD),
        unix.TIOCGWINSZ,
    )
    unix.IoctlSetWinsize(
        int(target.ptyMaster.Fd()),
        unix.TIOCSWINSZ,
        realWinsize,
    )

    // 3. Send SIGWINCH to trigger redraw
    unix.Kill(target.pid, unix.SIGWINCH)

    // Relay loop automatically switches to new foreground!
    // Swap time: ~5-10ms

    return nil
}
```

**Result**: Near-instant context switch, no process restart

### 4. SIGWINCH Propagation

```go
func (s *supervisor) handleRealSIGWINCH() {
    // prismctl received SIGWINCH from kitty

    realWinsize, _ := unix.IoctlGetWinsize(
        int(s.realPTY_S_FD),
        unix.TIOCGWINSZ,
    )

    s.mu.Lock()
    defer s.mu.Unlock()

    // Sync size to ALL child PTYs
    for _, prism := range s.prismList {
        unix.IoctlSetWinsize(
            int(prism.ptyMaster.Fd()),
            unix.TIOCSWINSZ,
            realWinsize,
        )

        // Send SIGWINCH to each child
        unix.Kill(prism.pid, unix.SIGWINCH)
    }
}
```

**Result**: Resize works correctly, Bubble Tea receives WindowSizeMsg

---

## Architecture Comparison

### Old: SIGSTOP/SIGCONT (Current)

```
prismctl → Real PTY → prism (foreground, running)
        ↘ Real PTY → prism (background, SIGSTOP - FROZEN)
        ↘ Real PTY → prism (background, SIGSTOP - FROZEN)
```

**Limitations**:
- ❌ Background processes completely frozen (no CPU)
- ❌ Can't fetch data while suspended
- ❌ Can't update state while suspended
- ✅ Instant resume (~50ms)
- ✅ Low memory (suspended processes)

### New: PTY-per-Process

```
prismctl → Real PTY ↔ child PTY1 → prism (background, RUNNING)
        ↘           ↔ child PTY2 → prism (background, RUNNING)
        ↘           ↔ child PTY3 → prism (foreground, RUNNING, RELAYED)
```

**Benefits**:
- ✅ Background processes keep running
- ✅ Can fetch data, update state, compute
- ✅ Even faster swap (~5-10ms, relay switch)
- ✅ State always current (no stale data)
- ⚠️ Higher memory (all processes active)

---

## What We Get

### ✅ Background Processing

```go
// shine-weather running in background:
for {
    select {
    case <-ticker.C:
        // Fetch new weather data every 5 minutes
        // Even when not visible!
        m.updateWeather()
    case msg := <-m.msgs:
        // Process Bubble Tea messages
        m.Update(msg)
    }
}
```

**Use cases**:
- Weather app fetches data while clock is visible
- Chat client maintains connection while calendar shown
- System monitor updates stats while weather displayed
- Music player controls playback while visualizer hidden

### ✅ Instant Swap with Current State

```
User swaps from clock → weather:
  T+0ms:  User presses hotkey
  T+2ms:  prismctl switches relay target
  T+3ms:  prismctl syncs terminal size
  T+4ms:  prismctl sends SIGWINCH to weather
  T+8ms:  Bubble Tea redraws weather
  T+10ms: Weather app visible with LATEST data

Total: 10ms (vs 50ms SIGCONT, vs 200ms restart)
```

### ✅ Full Kitty Protocol Support

**All these work perfectly**:

```bash
# Inside any prism (even background!)
$ echo $TERM
xterm-kitty  # ← Sees kitty!

$ kitty +kitten icat image.png
# Works! Graphics protocol passes through!

$ echo -e "\x1b[?u"
# Reports kitty keyboard protocol support!

# Unicode, ligatures, true color, all work!
```

**Why**: Zero translation, pure passthrough, `TERM=xterm-kitty` everywhere

### ✅ Zero tmux Issues

**tmux problem**: Translate `xterm-kitty` codes → `tmux-256color` → client terminal
**prismctl**: `xterm-kitty` codes → unchanged → `xterm-kitty` (kitty)

No translation = no bugs!

---

## Implementation Phases

### Phase 4A: PTY Infrastructure (NEW)

**Goal**: Replace SIGSTOP/SIGCONT with PTY allocation

**Tasks**:
1. **Add PTY library dependency**
   - `go get github.com/creack/pty`
   - Provides cross-platform PTY allocation

2. **Implement PTY manager** (`cmd/prismctl/pty_manager.go`)
   - `allocatePTY()` - creates master/slave pair
   - `syncTerminalSize(master, slave)` - sync winsize
   - `closePTY(master)` - cleanup on exit

3. **Modify `supervisor.go`**
   - Replace `exec.Command` with PTY-based launch
   - Track `ptyMaster` FD in `prismInstance` struct
   - Remove SIGSTOP/SIGCONT logic

4. **Test: Single prism**
   - Launch one prism with dedicated PTY
   - Verify `isatty()` = true inside prism
   - Verify `TERM=xterm-kitty`
   - Test kitty graphics protocol

**Success criteria**:
- ✅ Single prism renders correctly
- ✅ `echo $TERM` shows `xterm-kitty`
- ✅ `kitty +kitten icat` works
- ✅ Resize triggers redraw

### Phase 4B: I/O Relay (NEW)

**Goal**: Implement relay loop for foreground PTY

**Tasks**:
1. **Create relay loop** (`cmd/prismctl/relay.go`)
   - Bidirectional `io.Copy` goroutines
   - Real PTY ↔ foreground child PTY
   - Context cancellation on foreground change

2. **Integrate with supervisor**
   - Start relay on first prism launch
   - Switch relay target on swap
   - Cancel old relay, start new relay

3. **Handle relay errors**
   - EOF on child PTY (process exit)
   - Write errors (terminal gone)
   - Graceful fallback

4. **Test: Multi-prism**
   - Launch 3 prisms with separate PTYs
   - Verify all run simultaneously
   - Verify only foreground visible

**Success criteria**:
- ✅ Launch 3 prisms, all processes running
- ✅ Only foreground shows output
- ✅ Background processes keep working

### Phase 4C: Hot-Swap (NEW)

**Goal**: Instant swap via relay retargeting

**Tasks**:
1. **Implement swap logic**
   - Update MRU list (move target to front)
   - Sync terminal size to new foreground
   - Send SIGWINCH for redraw
   - Switch relay target

2. **Add swap IPC command**
   - Update `cmd/prismctl/ipc.go`
   - `{"action":"swap","target":"shine-weather"}`

3. **Measure swap latency**
   - Instrument swap timing
   - Target: <10ms

4. **Test: Rapid swaps**
   - Swap between 3 prisms rapidly
   - Verify no corruption
   - Verify state preservation

**Success criteria**:
- ✅ Swap completes in <10ms
- ✅ No terminal corruption
- ✅ Background state preserved

### Phase 4D: SIGWINCH Propagation (NEW)

**Goal**: Manual SIGWINCH handling for resize

**Tasks**:
1. **Install SIGWINCH handler**
   - Modify `cmd/prismctl/signals.go`
   - Add `handleSIGWINCH()` function

2. **Implement size sync**
   - Get size from Real PTY via `ioctl(TIOCGWINSZ)`
   - Set size on all child PTYs via `ioctl(TIOCSWINSZ)`
   - Send SIGWINCH to all children

3. **Optimize: Foreground-only**
   - Option: Only sync size to foreground
   - Background processes get synced on swap
   - Reduces overhead

4. **Test: Resize**
   - Resize kitty window
   - Verify foreground redraws immediately
   - Swap to background, verify correct size

**Success criteria**:
- ✅ Resize kitty window → prism redraws
- ✅ Bubble Tea receives WindowSizeMsg
- ✅ Swap to background → correct size

### Phase 4E: PTY Cleanup (NEW)

**Goal**: Proper PTY resource management

**Tasks**:
1. **Track PTY FDs**
   - Store master FDs in `prismInstance`
   - Close master on process exit
   - Prevent FD leaks

2. **Handle child exit**
   - Detect child process termination
   - Close associated PTY master
   - Remove from prism list
   - Reset terminal if was foreground

3. **Shutdown cleanup**
   - Close all PTY masters on prismctl exit
   - Restore Real PTY terminal state
   - Kill all child processes gracefully

4. **Test: Exit scenarios**
   - Child crash → PTY cleanup
   - Child clean exit → PTY cleanup
   - prismctl shutdown → all PTYs closed

**Success criteria**:
- ✅ No FD leaks (check with `lsof`)
- ✅ Clean shutdown
- ✅ Terminal restored on exit

### Phase 5: Polish & Optimization

**Tasks**:
1. **Memory-based eviction**
   - Monitor memory usage per prism
   - Kill LRU background prisms over threshold
   - Restart on swap if needed

2. **Metrics collection**
   - Swap frequency
   - Relay throughput (bytes/sec)
   - Memory usage per prism
   - Swap latency histogram

3. **Performance tuning**
   - Buffer sizes for relay
   - Relay goroutine optimization
   - Reduce allocations

4. **Documentation**
   - Architecture guide
   - PTY management internals
   - Difference from tmux
   - Troubleshooting guide

---

## Migration from Current Architecture

### Backward Compatibility

**Breaking change**: Prisms will now **keep running** when not foreground.

**Impact**:
- Positive: Background work now possible!
- Neutral: Memory usage slightly higher
- Negative: None (pure upgrade)

**Migration steps**:
1. Users update binaries (`go build`)
2. Restart shinectl (`shine stop && shine start`)
3. Prisms now run continuously

**No config changes required!**

---

## Success Metrics

### Performance Targets

- **Swap latency**: <10ms (vs 50ms SIGCONT)
- **Relay overhead**: <1% CPU per prism
- **Memory overhead**: <5MB per background prism
- **Zero corruption**: 100% of swaps clean

### Feature Validation

- ✅ Background processing works (data fetching)
- ✅ Instant swap feels instantaneous
- ✅ Kitty graphics protocol works in prisms
- ✅ Kitty keyboard protocol works in prisms
- ✅ Resize triggers redraw correctly
- ✅ No tmux-style protocol bugs

---

## Risks & Mitigation

### Risk 1: Complexity

**Risk**: PTY management more complex than SIGSTOP

**Mitigation**:
- Use battle-tested `github.com/creack/pty` library
- Implement incrementally (Phases 4A-4E)
- Extensive testing at each phase

### Risk 2: Memory Usage

**Risk**: All prisms running → higher memory

**Mitigation**:
- Implement eviction policy (kill LRU after N background)
- Monitor per-prism memory usage
- Make eviction threshold configurable

### Risk 3: Relay Bugs

**Risk**: I/O relay could drop bytes or deadlock

**Mitigation**:
- Use proven `io.Copy` pattern
- Add comprehensive error handling
- Test with stress scenarios (rapid swaps, large output)

---

## Conclusion

**The PTY-per-process architecture provides the best of all worlds**:

1. **Background processing** (vs SIGSTOP freezing)
2. **Instant swap** (faster than SIGCONT)
3. **Full kitty compatibility** (vs tmux breaking protocols)
4. **State preservation** (in-memory, always current)
5. **Kitty-native design** (uses kitty's features correctly)

**We avoid tmux's problems** by:
- Single terminal type (no translation)
- Pure passthrough (no parsing)
- Kitty-only (no multi-terminal support)
- TERM=xterm-kitty everywhere

**This is the architecture Kovid Goyal would approve of**: kitty-native, protocol-preserving, zero translation, uses kitty's features properly.

**Next Action**: Proceed with Phase 4A (PTY Infrastructure) implementation.

---

**Document Status**: Approved Architecture
**Implementation Lead**: TBD
**Target Completion**: Phase 4 within 2 weeks
**Dependencies**: `github.com/creack/pty` library
