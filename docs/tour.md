# Session: <command-message>codetour is running…</command-message>

<command-name>/codetour</command-name>
...
_2025-11-09 21:01:29_

---

## 🤖 Assistant

_2025-11-10T05:01:28Z_

# Shine Desktop Shell: Architecture Tour

Welcome to this comprehensive code tour of the **Shine** desktop shell toolkit. I'll walk you through how the three binaries (`shine`, `shinectl`, `prismctl`) interact with the shared `pkg` code to create a TUI-based panel system for Hyprland.

## Overview: The Three-Tier Architecture

Shine implements a **three-tier supervisor pattern** where each layer has clear responsibilities:

```
┌─────────────────────────────────────────┐
│  shine (User CLI)                       │  ← User-facing command interface
│  cmd/shine/                             │
└────────────┬────────────────────────────┘
             │ IPC over Unix sockets
             ↓
┌─────────────────────────────────────────┐
│  shinectl (Service Manager)             │  ← Configuration & panel lifecycle
│  cmd/shinectl/                          │
└────────────┬────────────────────────────┘
             │ Spawns via Kitty remote control
             ↓
┌─────────────────────────────────────────┐
│  prismctl (Panel Supervisor)            │  ← Process supervision & hot-swap
│  cmd/prismctl/                          │
└────────────┬────────────────────────────┘
             │ Fork/exec with PTY
             ↓
┌─────────────────────────────────────────┐
│  shine-* prisms (Bubble Tea TUIs)       │  ← Individual widget applications
│  cmd/shine-{bar,clock,chat}/           │
└─────────────────────────────────────────┘
```

All three binaries share core abstractions from `pkg/`:

- **`pkg/config`** - Configuration loading, discovery, merging
- **`pkg/panel`** - Kitty panel management & remote control
- **`pkg/prism`** - Prism lifecycle tracking

---

## Part 1: Shared Foundation (`pkg/`)

### Configuration System (`pkg/config/`)

The configuration system handles three sources of prism configuration:

#### [`pkg/config/types.go`](pkg/config/types.go#L6-150)

```go
// Config represents the main shine configuration
type Config struct {
    Core   *CoreConfig             `toml:"core"`
    Prisms map[string]*PrismConfig `toml:"prisms"`
}

// PrismConfig is the unified configuration for ALL prisms
type PrismConfig struct {
    Name         string      `toml:"name"`
    Enabled      bool        `toml:"enabled"`

    // Positioning (origin + offset)
    Origin   string      `toml:"origin,omitempty"`   // "top-left", "center", etc.
    Position string      `toml:"position,omitempty"` // "x,y" offset in pixels
    Width    interface{} `toml:"width,omitempty"`    // int or "200px"
    Height   interface{} `toml:"height,omitempty"`   // int or "100px"

    // Metadata (ONLY from prism sources, never from user config)
    Metadata map[string]interface{} `toml:"metadata,omitempty"`

    // Internal: resolved binary path
    ResolvedPath string `toml:"-"`
}
```

**[✏️ CHANGE]** It's not that important, i've removed the emphasis from the comments

> **⚠️ CRITICAL**: The `Metadata` field is **ALWAYS** sourced from prism manifests (`.toml` files), **NEVER** from user config in `shine.toml`. This separation ensures prism authors control metadata.

#### Configuration Discovery [`pkg/config/discovery.go`](pkg/config/discovery.go#L31-78)

Three discovery modes for finding prisms:

1. **Type 1: Directory with `prism.toml` + binary** (e.g., `~/.config/shine/prisms/weather/`)
2. **Type 2: Data directory** - `prism.toml` references binary via PATH
3. **Type 3: Standalone `.toml` files** (e.g., `clock.toml` references `shine-clock` in PATH)

```go
// DiscoverPrisms searches configured directories
func DiscoverPrisms(prismDirs []string) (map[string]*DiscoveredPrism, error) {
    // Walks directories looking for:
    // - Subdirectories with prism.toml
    // - Standalone .toml files
    // Returns map of prism name → discovered config + binary path
}
```

#### Configuration Merging [`pkg/config/discovery.go#L196-263`](pkg/config/discovery.go#L196-263)

```go
// MergePrismConfigs: User config OVERRIDES prism defaults
// BUT metadata ALWAYS comes from prism source
func MergePrismConfigs(prismSource, userConfig *PrismConfig) *PrismConfig {
    merged := &PrismConfig{}

    // Runtime state from user
    merged.Enabled = userConfig.Enabled || prismSource.Enabled

    // Layout from user (if set), otherwise prism default
    if userConfig.Origin != "" {
        merged.Origin = userConfig.Origin
    } else {
        merged.Origin = prismSource.Origin
    }

    // Metadata ALWAYS from source (NEVER from user config)
    merged.Metadata = prismSource.Metadata

    return merged
}
```

---

### Panel Management (`pkg/panel/`)

The panel package abstracts **Kitty's layer shell integration** via remote control API.

#### [`pkg/panel/config.go`](pkg/panel/config.go#L216-260)

**[➕ ADD]** We to include a sort of override parameter or rather like a direct args parameter maybe just call it kitten args or panel args and where users can just supply a option string or an argument string which will take precedence over any of the values in the config.

```go
// Config represents layer shell panel configuration
type Config struct {
    // Layer shell properties
    Type        LayerType   // background, panel, top, overlay
    Origin      Origin      // top-left, top-center, center, etc.
    FocusPolicy FocusPolicy // not-allowed, exclusive, on-demand

    // Size (cells or pixels)
    Width  Dimension // e.g., Dimension{Value: 200, IsPixels: true}
    Height Dimension

    // Position offset from origin (in pixels)
    Position Position // e.g., Position{X: 10, Y: 50}

    // Output (CRITICAL: Must be DP-2 for this setup)
    OutputName string

    **[➕ ADD]**
    // grab-keyboard=no --listen-on 1234
    KittyArgs map[string]string
}
```

> **⚠️ CRITICAL**: Default output is hardcoded to `"DP-2"` (see line 258). This is environment-specific and should be configurable.

#### Margin Calculation [`pkg/panel/config.go#L310-378`](pkg/panel/config.go#L310-378)

The panel system uses an **origin + offset** model:

1. **Origin** defines the anchor point (e.g., `top-right`)
2. **Position** defines pixel offset from that anchor
3. **calculateMargins()** converts this to Kitty's margin system

```go
func (c *Config) calculateMargins() (top, left, bottom, right int, err error) {
    // Query Hyprland for monitor resolution
    monWidth, monHeight, err := getMonitorResolution(c.OutputName)

    // Convert panel dimensions to pixels
    panelWidth := c.Width.Value
    if !c.Width.IsPixels {
        panelWidth = c.Width.Value * 10 // Estimate: 10px per column
    }

    // Calculate margins based on origin
    switch c.Origin {
    case OriginTopLeft:
        left = offsetX
        top = offsetY
    case OriginCenter:
        left = (monWidth / 2) - (panelWidth / 2) + offsetX
        top = (monHeight / 2) - (panelHeight / 2) + offsetY
    // ... other cases
    }

    return top, left, bottom, right, nil
}
```

#### Panel Manager [`pkg/panel/manager.go`](pkg/panel/manager.go#L28-41)

```go
// Manager manages panel instances via Kitty remote control
type Manager struct {
    instances     map[string]*Instance // name → running instance
    kittyInstance *kittyInstance       // Detected Kitty socket
    mu            sync.RWMutex
}

// detectKittySocket finds a Kitty instance with remote control enabled
// Tries 3 methods:
// 1. KITTY_WINDOW_ID (we're inside Kitty)
// 2. KITTY_LISTEN_ON environment variable
// 3. Scan PIDs for /tmp/@mykitty-<PID> sockets
```

#### Launching Panels [`pkg/panel/manager.go#L159-218`](pkg/panel/manager.go#L159-218)

```go
func (m *Manager) LaunchViaRemoteControl(name string, config *Config, component string) (*Instance, error) {
    // Find Kitty socket
    socketPath, err := m.detectKittySocket()

    // Set window title for tracking
    config.WindowTitle = fmt.Sprintf("shine-%s", name)

    // Build args: kitty @ launch --type=os-panel ...
    args := config.ToRemoteControlArgs(component)

    // Launch via remote control
    cmd := exec.Command("kitty", fullArgs...)
    output, _ := cmd.Output()
    windowID := strings.TrimSpace(string(output)) // Kitty returns window ID

    // Create instance tracking
    instance := &Instance{
        Name:        name,
        WindowID:    windowID,
        WindowTitle: config.WindowTitle,
        Remote:      NewRemoteControl(socketPath),
    }

    return instance, nil
}
```

---

### Prism Lifecycle (`pkg/prism/`)

The prism package wraps panel management with **prism-specific tracking**.

#### [`pkg/prism/manager.go`](pkg/prism/manager.go#L14-26)

```go
// Manager handles prism lifecycle and tracking
type Manager struct {
    panelMgr *panel.Manager
    prisms   map[string]*Instance
}

// Instance tracks a running prism
type Instance struct {
    Name       string
    BinaryPath string
    Config     *config.PrismConfig
    Panel      *panel.Instance
    StartTime  time.Time
}
```

#### Launch Flow [`pkg/prism/manager.go#L45-88`](pkg/prism/manager.go#L45-88)

```go
func (m *Manager) Launch(name string, cfg *config.PrismConfig) error {
    // Get resolved binary path (from discovery)
    binaryPath := cfg.ResolvedPath
    if binaryPath == "" {
        // Fallback to PATH lookup
        binaryPath, err = exec.LookPath("shine-" + name)
    }

    // Convert to panel config
    panelCfg := cfg.ToPanelConfig()

    // Launch via panel manager (which uses Kitty remote control)
    panelInstance, err := m.panelMgr.Launch(name, panelCfg, binaryPath)

    // Track instance
    m.prisms[name] = &Instance{
        Name:       name,
        BinaryPath: binaryPath,
        Panel:      panelInstance,
        StartTime:  time.Now(),
    }
}
```

---

## Part 2: The Three Binaries

### Binary 1: `shine` (User CLI)

The `shine` binary is the **user-facing command interface** using IPC to communicate with `shinectl`.

#### Main Entry Point [`cmd/shine/main.go`](cmd/shine/main.go#L10-84)

```go
func main() {
    command := os.Args[1]

    // Route to command handlers
    switch command {
    case "start":   err = cmdStart()
    case "stop":    err = cmdStop()
    case "reload":  err = cmdReload()
    case "status":  err = cmdStatus()
    case "logs":    err = cmdLogs(panelID)
    }
}
```

#### Start Command [`cmd/shine/commands.go#L103-140`](cmd/shine/commands.go#L103-140)

```go
func cmdStart() error {
    // Check if shinectl is already running
    _, err := findShinectlSocket()
    if err == nil {
        Success("shinectl is already running")
        return nil
    }

    // Find shinectl binary
    shinectlBin, err := exec.LookPath("shinectl")

    // Start shinectl in background
    cmd := exec.Command(shinectlBin)
    cmd.Start()

    // Wait for socket to appear (up to 5 seconds)
    for i := 0; i < 50; i++ {
        if _, err := findShinectlSocket(); err == nil {
            Success("shinectl started")
            return nil
        }
        time.Sleep(100 * time.Millisecond)
    }
}
```

#### IPC Communication [`cmd/shine/commands.go#L76-101`](cmd/shine/commands.go#L76-101)

```go
// sendIPCCommand sends JSON over Unix socket
func sendIPCCommand(socketPath string, cmd ipcCommand) (*ipcResponse, error) {
    conn, err := net.DialTimeout("unix", socketPath, 5*time.Second)
    defer conn.Close()

    // Send command as JSON
    encoder := json.NewEncoder(conn)
    encoder.Encode(cmd)

    // Read response
    var resp ipcResponse
    decoder := json.NewDecoder(conn)
    decoder.Decode(&resp)

    return &resp, nil
}
```

> **⚠️ IMPORTANT**: Always find the **most recent socket** before running IPC commands. Socket paths include the PID (e.g., `prism-panel-0.12345.sock`), so they change on restart.

---

### Binary 2: `shinectl` (Service Manager)

The `shinectl` binary is a **long-running daemon** that:

1. Loads configuration from `~/.config/shine/prism.toml`
2. Spawns `prismctl` instances in Kitty panels via remote control
3. Monitors panel health and handles restarts
4. Responds to SIGHUP for config reload

#### Main Loop [`cmd/shinectl/main.go#L36-115`](cmd/shinectl/main.go#L36-115)

```go
func main() {
    // Setup logging to file
    setupLogging()

    // Load configuration
    config := LoadConfigOrDefault(cfgPath)

    // Create panel manager (pkg/panel)
    pm, err := NewPanelManager()

    // Spawn initial panels
    spawnConfiguredPanels(pm, config)

    // Signal handlers
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, SIGHUP, SIGTERM, SIGINT)

    // Main event loop
    for {
        select {
        case sig := <-sigCh:
            switch sig {
            case SIGHUP:
                reloadConfig(pm, cfgPath) // Hot-reload
            case SIGTERM, SIGINT:
                pm.Shutdown()
                return
            }
        case <-healthTicker.C:
            pm.MonitorPanels() // Check health every 30s
        }
    }
}
```

#### Panel Spawning [`cmd/shinectl/panel_manager.go#L70-150`](cmd/shinectl/panel_manager.go#L70-150)

```go
func (pm *PanelManager) SpawnPanel(config *PrismEntry, componentName string) (*Panel, error) {
    // Build command: kitten @ launch --type=window prismctl <prism-name> <component>
    title := fmt.Sprintf("shine-%s", componentName)

    cmd := exec.Command(
        "kitten", "@", "launch",
        "--type=window",
        "--title", title,
        pm.prismctlBin, config.Name, componentName,
    )

    // Execute (returns window ID)
    output, err := cmd.CombinedOutput()
    windowID := strings.TrimSpace(string(output))

    // Wait for prismctl to create socket
    socketPattern := fmt.Sprintf("/run/user/%d/shine/prism-%s.*.sock", uid, componentName)

    var socketPath string
    for i := 0; i < 50; i++ {
        matches, _ := filepath.Glob(socketPattern)
        if len(matches) > 0 {
            socketPath = matches[0]
            break
        }
        time.Sleep(100 * time.Millisecond)
    }

    // Extract PID from socket name (prism-{component}.{pid}.sock)
    parts := strings.Split(filepath.Base(socketPath), ".")
    pid, _ := strconv.Atoi(parts[len(parts)-2])

    // Track panel
    panel := &Panel{
        Name:       config.Name,
        Component:  componentName,
        WindowID:   windowID,
        SocketPath: socketPath,
        IPCClient:  NewIPCClient(socketPath),
        PID:        pid,
    }
}
```

#### Health Monitoring & Restart Policies [`cmd/shinectl/panel_manager.go#L204-279`](cmd/shinectl/panel_manager.go#L204-279)

```go
func (pm *PanelManager) handlePanelCrash(panel *Panel) {
    // Update crash tracking
    now := time.Now()
    if now.Sub(panel.LastCrash) > time.Hour {
        panel.CrashCount = 0 // Reset if > 1 hour since last crash
    }
    panel.CrashCount++

    // Check restart policy
    policy := panel.Config.GetRestartPolicy()
    shouldRestart := false

    switch policy {
    case RestartAlways:        shouldRestart = true
    case RestartOnFailure:     shouldRestart = true // Crash = failure
    case RestartUnlessStopped: shouldRestart = true
    case RestartNo:            shouldRestart = false
    }

    // Check max_restarts limit
    if shouldRestart && panel.Config.MaxRestarts > 0 &&
       panel.CrashCount > panel.Config.MaxRestarts {
        shouldRestart = false
    }

    if shouldRestart {
        delay := panel.Config.GetRestartDelay()
        time.Sleep(delay)
        // Re-spawn panel
        newPanel, err := pm.spawnPanelUnlocked(panel.Config, panel.Component)
    }
}
```

---

### Binary 3: `prismctl` (Panel Supervisor)

The `prismctl` binary is the **process supervisor** that:

1. Manages individual prism processes with PTY allocation
2. Provides IPC for hot-swapping prisms within a panel
3. Handles terminal state management
4. Implements MRU (Most Recently Used) ordering for background prisms

#### Main Entry [`cmd/prismctl/main.go#L66-129`](cmd/prismctl/main.go#L66-129)

```go
func main() {
    prismName := os.Args[1]       // e.g., "shine-clock"
    componentName := os.Args[2]    // e.g., "panel-0"

    // Initialize terminal state management
    termState, err := newTerminalState()

    // Create supervisor
    sup := newSupervisor(termState)

    // Setup signal handling (SIGCHLD, SIGTERM, SIGWINCH)
    sigHandler := newSignalHandler(sup)

    // Start IPC server
    ipcServer, err := newIPCServer(componentName, sup)
    go ipcServer.serve()

    // Start initial prism
    sup.startPrism(prismName)

    // Run signal handler (blocks until shutdown)
    sigHandler.run()
}
```

#### Supervisor Core [`cmd/prismctl/supervisor.go#L32-63`](cmd/prismctl/supervisor.go#L32-63)

```go
// supervisor manages the lifecycle of child prism processes
type supervisor struct {
    mu          sync.Mutex
    termState   *terminalState
    prismList   []prismInstance  // MRU list: [0] = foreground
    relay       *relayState      // Real PTY ↔ foreground child PTY
    relayCtx    context.Context
    relayCancel context.CancelFunc
}

// prismInstance represents a single prism process
type prismInstance struct {
    name      string
    pid       int
    state     prismState      // foreground or background
    ptyMaster *os.File        // Child's PTY master FD
}
```

The supervisor maintains a **MRU (Most Recently Used) list**:

- `prismList[0]` → foreground (visible)
- `prismList[1+]` → background (suspended via SIGSTOP)

#### Launch & Foreground [`cmd/prismctl/supervisor.go#L101-177`](cmd/prismctl/supervisor.go#L101-177)

```go
func (s *supervisor) launchAndForeground(prismName string) error {
    // Resolve binary path
    binaryPath, err := exec.LookPath(prismName)

    // Move current foreground to background
    if len(s.prismList) > 0 {
        s.prismList[0].state = prismBackground
    }

    // Allocate PTY pair for new prism
    ptyMaster, ptySlave, err := allocatePTY()

    // Sync terminal size from real terminal to child PTY
    syncTerminalSize(int(os.Stdin.Fd()), int(ptyMaster.Fd()))

    // CRITICAL: Reset terminal state
    s.termState.resetTerminalState()

    // Stabilization delay (matches Kitty's timing)
    time.Sleep(10 * time.Millisecond)

    // Fork/exec with PTY as controlling terminal
    cmd := exec.Command(binaryPath)
    cmd.Stdin = ptySlave
    cmd.Stdout = ptySlave
    cmd.Stderr = ptySlave
    cmd.SysProcAttr = &syscall.SysProcAttr{
        Setsid:  true,  // Create new session
        Setctty: true,  // Make PTY controlling terminal
    }
    cmd.Start()
    ptySlave.Close() // Parent closes slave

    // Add to front of MRU list
    newInstance := prismInstance{
        name:      prismName,
        pid:       cmd.Process.Pid,
        state:     prismForeground,
        ptyMaster: ptyMaster,
    }
    s.prismList = append([]prismInstance{newInstance}, s.prismList...)

    // Start relay to new foreground prism
    s.startRelayToForeground()
}
```

> **⚠️ CRITICAL**: The 10ms stabilization delay (line 132) is **essential** for terminal state consistency. Modifying this timing can cause terminal corruption.

#### Resume to Foreground [`cmd/prismctl/supervisor.go#L179-225`](cmd/prismctl/supervisor.go#L179-225)

```go
func (s *supervisor) resumeToForeground(targetIdx int) error {
    target := s.prismList[targetIdx]

    // Move current foreground to background
    s.prismList[0].state = prismBackground

    // CRITICAL: Reset terminal state
    s.termState.resetTerminalState()

    // Stabilization delay
    time.Sleep(10 * time.Millisecond)

    // Sync terminal size to target PTY
    syncTerminalSize(int(os.Stdin.Fd()), int(target.ptyMaster.Fd()))

    // Reorder MRU list: move target to [0]
    s.prismList = append(s.prismList[:targetIdx], s.prismList[targetIdx+1:]...)
    target.state = prismForeground
    s.prismList = append([]prismInstance{target}, s.prismList...)

    // Hot-swap relay to new foreground prism
    s.swapRelay()

    // Send SIGWINCH to trigger redraw
    unix.Kill(target.pid, unix.SIGWINCH)
}
```

#### Relay Management [`cmd/prismctl/relay.go`](cmd/prismctl/relay.go#L24-79)

The relay system provides **bidirectional I/O** between the real terminal and the foreground prism's PTY:

```go
// startRelay launches bidirectional copy
// Real PTY (stdin/stdout) ↔ child PTY master (foreground prism)
func startRelay(ctx context.Context, realPTY *os.File, childPTY *os.File) (*relayState, error) {
    relayCtx, cancel := context.WithCancel(ctx)

    state := &relayState{
        ctx:      relayCtx,
        cancel:   cancel,
        childPTY: childPTY,
    }

    // Real PTY → child PTY (user input to prism)
    go io.Copy(childPTY, realPTY)

    // child PTY → Real PTY (prism output to terminal)
    go io.Copy(os.Stdout, childPTY)

    return state, nil
}

// stopRelay cancels relay goroutines
func stopRelay(state *relayState) {
    state.cancel()

    // Force child PTY io.Copy to return by setting deadline
    state.childPTY.SetReadDeadline(time.Unix(0, 0))

    // Don't wait - stdin reader will block until next input
    state.active = false
}
```

#### Swap Relay (Hot-Swap) [`cmd/prismctl/supervisor.go#L467-494`](cmd/prismctl/supervisor.go#L467-494)

```go
func (s *supervisor) swapRelay() error {
    // Stop current relay
    if s.relay != nil {
        stopRelay(s.relay)
        s.relay = nil
    }

    // Clear screen AFTER stopping old relay but BEFORE starting new one
    // CSI 2 J = clear screen, CSI H = cursor home, CSI 0 m = reset
    os.Stdout.WriteString("\x1b[2J\x1b[H\x1b[0m")

    // Start new relay to foreground
    s.startRelayToForeground()
}
```

> **⚠️ IMPORTANT**: The screen clear sequence **must** happen between stopping the old relay and starting the new one to prevent race conditions with buffered output.

#### Terminal State Management [`cmd/prismctl/terminal.go`](cmd/prismctl/terminal.go#L10-79)

```go
// terminalState holds the saved terminal state for restoration
type terminalState struct {
    savedTermios *unix.Termios
    fd           int
}

// resetTerminalState resets to canonical mode and clears visual state
// MUST be called after EVERY child exit to prevent terminal corruption
func (ts *terminalState) resetTerminalState() error {
    // Get current termios
    termios, err := unix.IoctlGetTermios(ts.fd, unix.TCGETS)

    // Set canonical mode flags
    termios.Lflag |= unix.ICANON | unix.ECHO | unix.ISIG
    termios.Iflag |= unix.ICRNL

    // Apply settings immediately
    unix.IoctlSetTermios(ts.fd, unix.TCSETS, termios)

    // Send visual reset sequences
    resetSeq := []byte{
        0x1b, '[', '0', 'm',                     // SGR reset
        0x1b, '[', '?', '1', '0', '4', '9', 'l', // Exit alt screen
        0x1b, '[', '?', '2', '5', 'h',           // Show cursor
        0x1b, '[', '?', '1', '0', '0', '0', 'l', // Disable mouse
    }
    unix.Write(ts.fd, resetSeq)
}
```

> **⚠️ CRITICAL**: Terminal state reset is **non-negotiable** after every child exit. Skipping this causes terminal corruption that persists across prism swaps.

#### Signal Handling [`cmd/prismctl/signals.go`](cmd/prismctl/signals.go#L11-125)

```go
type signalHandler struct {
    sigCh      chan os.Signal
    supervisor *supervisor
}

func (sh *signalHandler) run() {
    for sig := range sh.sigCh {
        switch sig {
        case unix.SIGCHLD:
            sh.handleSIGCHLD()        // Reap zombies, auto-resume next
        case unix.SIGINT:
            sh.handleSIGINT()         // Kill foreground or shutdown
        case unix.SIGTERM, unix.SIGHUP:
            sh.handleShutdown(sig)    // Graceful shutdown
        case unix.SIGWINCH:
            sh.handleSIGWINCH()       // Propagate resize to ALL prisms
        }
    }
}

// handleSIGCHLD reaps zombie processes
func (sh *signalHandler) handleSIGCHLD() {
    for {
        var status unix.WaitStatus
        pid, err := unix.Wait4(-1, &status, unix.WNOHANG, nil)
        if err != nil || pid <= 0 {
            break // No more children to reap
        }

        exitCode := 0
        if status.Exited() {
            exitCode = status.ExitStatus()
        }

        // Notify supervisor of child exit
        sh.supervisor.handleChildExit(pid, exitCode)
    }
}
```

---

## Part 3: Data Flow & Critical Paths

### Startup Flow

```
1. User runs: shine start
   ↓
2. shine CLI spawns shinectl in background
   ↓
3. shinectl loads ~/.config/shine/prism.toml
   ↓
4. For each enabled prism:
   a. shinectl calls pkg/panel.Manager.LaunchViaRemoteControl()
   b. Builds args: kitty @ launch --type=os-panel prismctl <prism> <component>
   c. Kitty spawns new window running prismctl
   ↓
5. prismctl initializes:
   a. Saves terminal state
   b. Creates IPC socket (/run/user/{uid}/shine/prism-{component}.{pid}.sock)
   c. Forks prism binary with PTY
   d. Starts relay: Real PTY ↔ child PTY
   ↓
6. prism (Bubble Tea TUI) runs in foreground
```

### Hot-Swap Flow (IPC)

```
1. Client sends: {"action":"start","prism":"shine-chat"}
   ↓
2. prismctl IPC server receives command
   ↓
3. supervisor.start("shine-chat")
   ↓
4. Case detection:
   - Not in list → launchAndForeground()
   - Already foreground → no-op
   - In background → resumeToForeground()
   ↓
5. resumeToForeground():
   a. Move current foreground to background
   b. Reset terminal state (canonical mode)
   c. Sync terminal size to target PTY
   d. Reorder MRU list (target → position [0])
   e. swapRelay() (includes screen clear)
   f. Send SIGWINCH to trigger redraw
   ↓
6. New prism visible, old prism suspended
```

### Child Exit & Auto-Resume

```
1. Prism process exits (clean or crash)
   ↓
2. Kernel sends SIGCHLD to prismctl
   ↓
3. Signal handler reaps zombie with Wait4()
   ↓
4. supervisor.handleChildExit(pid, exitCode)
   ↓
5. Cleanup:
   a. Close PTY master
   b. Stop relay if foreground
   c. Reset terminal state
   d. Remove from MRU list
   ↓
6. If was foreground AND others exist:
   a. Sync terminal size to next prism
   b. Send SIGWINCH to trigger redraw
   c. Mark next as foreground
   ↓
7. If MRU list now empty:
   → Auto-shutdown prismctl
```

---

## Critical Timing & State Management

### Timing Constants (DO NOT MODIFY)

From [`cmd/prismctl/supervisor.go`](cmd/prismctl/supervisor.go):

- **10ms stabilization delay** after terminal state reset (line 132, 197)
- **20ms shutdown grace period** for SIGTERM → SIGKILL (line 402)

> **⚠️ CRITICAL**: These timings match Kitty's internal signal delivery and terminal state propagation. Modifying them causes race conditions and terminal corruption.

### Socket Naming Convention

```
/run/user/{uid}/shine/prism-{component}.{pid}.sock
```

Example: `/run/user/1000/shine/prism-panel-0.12345.sock`

**Why PID suffix?** Prevents conflicts when restarting panels - old socket is removed, new one created with new PID.

### IPC Best Practice

```bash
# ALWAYS find current socket first
SOCK=$(ls -t /run/user/$(id -u)/shine/prism-panel-0.*.sock | head -1)

# Then use it
echo '{"action":"status"}' | socat - UNIX-CONNECT:$SOCK
```

---

## Key Integration Points

### `pkg/config` → `shinectl`

[`cmd/shinectl/main.go#L64`](cmd/shinectl/main.go#L64) loads configuration:

```go
config := LoadConfigOrDefault(cfgPath) // Uses pkg/config.Load()
```

### `pkg/panel` → `shinectl`

[`cmd/shinectl/main.go#L72-75`](cmd/shinectl/main.go#L72-75) creates panel manager:

```go
pm, err := NewPanelManager() // Wraps pkg/panel.Manager
```

### `pkg/prism` → `shinectl`

[`cmd/shinectl/panel_manager.go#L143`](cmd/shinectl/panel_manager.go#L143) tracks instances:

```go
panel := &Panel{
    Name:       config.Name,
    IPCClient:  NewIPCClient(socketPath), // For health monitoring
}
```

### `shine` → `shinectl` (IPC)

[`cmd/shine/commands.go#L76`](cmd/shine/commands.go#L76) sends JSON commands:

```go
sendIPCCommand(socketPath, ipcCommand{Action: "status"})
```

---

## Summary

The Shine architecture uses **three layers of abstraction**:

1. **`shine`** - User CLI that communicates via IPC
2. **`shinectl`** - Service manager that spawns panels and monitors health
3. **`prismctl`** - Panel supervisor with hot-swap and MRU ordering

All three binaries share core logic from `pkg/`:

- **`pkg/config`** handles discovery, merging, and validation
- **`pkg/panel`** abstracts Kitty layer shell integration
- **`pkg/prism`** tracks lifecycle and health

**Critical behaviors:**

- Terminal state reset after **every** child exit
- 10ms stabilization delays for state propagation
- MRU ordering with suspend/resume (SIGSTOP/SIGCONT)
- Hot-swap via relay swapping with screen clear
- IPC sockets include PID to prevent conflicts

This design enables **zero-overhead backgrounding** and **instant hot-swapping** while maintaining terminal integrity across crashes and restarts.

_Tokens: Input: 7 | Output: 9142 | Cache Creation: 82649 | Cache Read: 29324_
