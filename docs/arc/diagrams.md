# Architectural Building Blocks for Shine

Here are the fundamental components you can use to diagram and communicate your ideas:

## Core Building Blocks

### 1. **Process Blocks**

```
┌─────────────────┐
│   Process       │
│   (PID: 1234)   │
│                 │
│   Binary: name  │
└─────────────────┘

┌─────────────────┐     ┌─────────────────┐
│  Parent Process │────>│  Child Process  │
│   (spawner)     │fork │   (spawned)     │
└─────────────────┘     └─────────────────┘
```

### 2. **PTY Components**

```
┌─────────────────┐         ┌─────────────────┐
│  PTY Master     │<───────>│   PTY Slave     │
│  FD: 15         │  kernel │   /dev/pts/5    │
│  (no path)      │  driver │   (crw-------)  │
└─────────────────┘         └─────────────────┘

┌─────────────────┐
│  PTY Pair       │
│  Master: FD 15  │
│  Slave: pts/5   │
└─────────────────┘
```

### 3. **File Descriptors**

```
┌─────────────┐
│   FD: 0     │  stdin
│   FD: 1     │  stdout
│   FD: 2     │  stderr
│   FD: 15    │  custom (PTY master)
└─────────────┘

FD 0 ──→ /dev/pts/5  (points to)
FD 1 ──→ /dev/pts/5
FD 2 ──→ /dev/pts/5
```

### 4. **Bubble Tea Program**

```
┌───────────────────────┐
│  tea.Program          │
│                       │
│  Model:  state        │
│  Update: msg → model  │
│  View:   model → str  │
│                       │
│  Input:  io.Reader    │
│  Output: io.Writer    │
└───────────────────────┘
```

### 5. **I/O Streams**

```
Reader ──→  [Component]  ──→  Writer
   ↑                            ↓
  Input                      Output

┌──────────┐      ┌──────────┐      ┌──────────┐
│ io.Reader│─────>│ Process  │─────>│io.Writer │
└──────────┘      └──────────┘      └──────────┘

Examples:
- os.Stdin (io.Reader)
- os.Stdout (io.Writer)
- bytes.Buffer (io.Reader & io.Writer)
- net.Conn (io.Reader & io.Writer)
- PipeReader/PipeWriter
```

### 6. **Kitty Components**

```
┌───────────────────────┐
│  Kitty Instance       │
│                       │
│  Remote Control:      │
│  - Unix socket        │
│  - JSON protocol      │
│                       │
│  Commands:            │
│  - kitten @ launch    │
│  - kitten @ send-text │
│  - kitten @ ls        │
└───────────────────────┘

┌───────────────────────┐
│  Kitty Window/Panel   │
│                       │
│  ID: 1234             │
│  Title: "Clock"       │
│  PTY Master: FD 15    │
│  Child PID: 5678      │
└───────────────────────┘
```

### 7. **Communication Channels**

```
Direct (same process):
┌───────┐             ┌───────┐
│   A   │────chan────>│   B   │
└───────┘             └───────┘

Pipe:
┌───────┐   io.Pipe   ┌───────┐
│   A   │────────────>│   B   │
└───────┘   r ←─→ w   └───────┘

Remote Control:
┌───────┐  kitten @   ┌───────┐
│   A   │───────────> │Kitty  │
└───────┘  send-text  └───────┘

Unix Socket:
┌───────┐             ┌───────┐
│   A   │──/tmp/sock─>│   B   │
└───────┘             └───────┘
```

### 8. **Shine-Specific Components**

```
┌───────────────────────┐
│  Prism                │
│                       │
│  Executable: path     │
│  Config: toml         │
│  Protocol: JSON       │
│  Type: widget type    │
└───────────────────────┘

┌───────────────────────┐
│  Panel Manager        │
│                       │
│  Spawns panels        │
│  Tracks PIDs          │
│  Lifecycle control    │
└───────────────────────┘

┌───────────────────────┐
│  Prism Protocol       │
│                       │
│  stdin:  config       │
│  stdout: updates      │
│  Format: JSON         │
└───────────────────────┘
```

## Communication Patterns

### Pattern 1: Direct Function Call

```
┌─────────┐
│  Func A │──calls──> Func B
└─────────┘

Same process, shared memory
```

### Pattern 2: Channel (Go)

```
┌─────────┐            ┌─────────┐
│ Goroutine│──chan Msg─>│Goroutine│
│    A     │            │    B    │
└─────────┘            └─────────┘

Same process, async communication
```

### Pattern 3: Pipe

```
┌─────────┐  r,w := io.Pipe()  ┌─────────┐
│ Writer  │──────────────────> │ Reader  │
└─────────┘  w.Write() r.Read()└─────────┘

Same or different process
```

### Pattern 4: File Descriptor Passing

```
┌─────────┐  FD 15  ┌─────────┐
│ Parent  │────────>│  Child  │
│ Process │  fork   │ Process │
└─────────┘  exec   └─────────┘

Child inherits FDs from parent
```

### Pattern 5: Remote Control

```
┌─────────┐   kitten @   ┌─────────┐
│ Client  │─────────────>│  Kitty  │
│ Process │  send-text   │ Process │
└─────────┘              └─────────┘

Via socket, external process
```

## Data Flow Notation

```
──→   Synchronous data flow
··>   Asynchronous data flow
<──>  Bidirectional
├──>  Fork/Split
──┤   Join/Merge

[A]──(transform)──>[B]  Pipeline with transformation
[A]══{buffer}══>[B]     Buffered communication
```

## Abstraction Layers

```
┌──────────────────────────────┐  Application Layer
│  Shine Library API           │  (Your Go package)
└──────────────────────────────┘
            ↓
┌──────────────────────────────┐  Rendering Layer
│  Bubble Tea Programs         │  (TUI framework)
└──────────────────────────────┘
            ↓
┌──────────────────────────────┐  I/O Abstraction
│  io.Reader / io.Writer       │  (Go interfaces)
└──────────────────────────────┘
            ↓
┌──────────────────────────────┐  Transport Layer
│  PTY / Pipe / Socket         │  (Communication mechanism)
└──────────────────────────────┘
            ↓
┌──────────────────────────────┐  Platform Layer
│  Kernel / OS / Kitty         │  (System services)
└──────────────────────────────┘
```

## Complete PTY Architecture Visualization

### Full stdin/stdout/stderr Flow

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          Terminal Emulator Process                      │
│                              (Kitty - PID 1234)                         │
│                                                                         │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │                    Application Layer                           │    │
│  │  - Keyboard input handler                                      │    │
│  │  - Screen renderer (OpenGL)                                    │    │
│  │  - Bubble Tea Program (shine-clock)                            │    │
│  └──────────────────┬──────────────────────────┬──────────────────┘    │
│                     │ write()                  │ read()                 │
│                     ↓                          ↑                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    PTY Master Side                              │   │
│  │                                                                 │   │
│  │  File Descriptor: 15 (example)                                 │   │
│  │  Type: Character device (no filesystem path)                   │   │
│  │  Kernel object: /dev/ptmx → master side                        │   │
│  │                                                                 │   │
│  │  Operations:                                                    │   │
│  │  • write(15, "Hello\n", 6)   → send to slave                   │   │
│  │  • read(15, buf, 1024)       → receive from slave              │   │
│  │  • ioctl(15, TIOCSWINSZ, &ws) → set terminal size              │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                     │                          ↑                        │
└─────────────────────┼──────────────────────────┼─────────────────────────┘
                      │                          │
                      │    Kernel PTY Driver     │
                      │    (pseudo-terminal)     │
                      │                          │
                      ↓                          ↑
                 write/read                 write/read
                      │                          │
                      │                          │
┌─────────────────────┼──────────────────────────┼─────────────────────────┐
│                     ↓                          │                         │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                    PTY Slave Side                               │   │
│  │                                                                 │   │
│  │  File Path: /dev/pts/5 (example)                               │   │
│  │  Permissions: crw------- (600) owned by child process          │   │
│  │  Type: Character device (has filesystem path)                  │   │
│  │                                                                 │   │
│  │  File Descriptors in child process:                            │   │
│  │  • FD 0 (stdin)  → /dev/pts/5                                  │   │
│  │  • FD 1 (stdout) → /dev/pts/5                                  │   │
│  │  • FD 2 (stderr) → /dev/pts/5                                  │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                     │                          ↑                        │
│                     │ read(0)                  │ write(1/2)             │
│                     ↓                          │                        │
│  ┌─────────────────────────────────────────────────────────────────┐   │
│  │                  Child Process (bash - PID 5678)                │   │
│  │                                                                 │   │
│  │  stdin  (FD 0) ──→ read from user input                        │   │
│  │  stdout (FD 1) ──→ write program output                        │   │
│  │  stderr (FD 2) ──→ write error messages                        │   │
│  │                                                                 │   │
│  │  Environment:                                                   │   │
│  │  • TERM=xterm-kitty                                            │   │
│  │  • Controlling TTY: /dev/pts/5                                 │   │
│  │  • Process session leader                                      │   │
│  └─────────────────────────────────────────────────────────────────┘   │
│                                                                         │
│                          Child Process                                  │
└─────────────────────────────────────────────────────────────────────────┘
```

### Data Flow Example: User Types "ls" and Presses Enter

```
┌──────────────┐
│     User     │
│   Keyboard   │
└──────┬───────┘
       │ Key press: 'l', 's', '\n'
       ↓
┌──────────────────────────────────────┐
│  Kitty Input Handler                 │
│  - Captures keyboard events          │
│  - Converts to bytes: "ls\n"         │
└──────┬───────────────────────────────┘
       │ write(master_fd, "ls\n", 3)
       ↓
┌──────────────────────────────────────┐
│  PTY Master FD 15                    │
│  (in Kitty process)                  │
└──────┬───────────────────────────────┘
       │
       │ Kernel routes through PTY driver
       ↓
┌──────────────────────────────────────┐
│  PTY Slave /dev/pts/5                │
│  (in child process)                  │
└──────┬───────────────────────────────┘
       │ read(0, buf, 1024)  [stdin]
       ↓
┌──────────────────────────────────────┐
│  bash process                        │
│  - Reads "ls\n" from stdin           │
│  - Parses command                    │
│  - Executes /bin/ls                  │
└──────┬───────────────────────────────┘
       │
       ↓
┌──────────────────────────────────────┐
│  ls program                          │
│  - Lists directory                   │
│  - Writes to stdout                  │
│  - write(1, "file1\nfile2\n", 12)   │
└──────┬───────────────────────────────┘
       │ stdout → /dev/pts/5
       ↓
┌──────────────────────────────────────┐
│  PTY Slave /dev/pts/5                │
└──────┬───────────────────────────────┘
       │
       │ Kernel routes through PTY driver
       ↓
┌──────────────────────────────────────┐
│  PTY Master FD 15                    │
└──────┬───────────────────────────────┘
       │ read(master_fd, buf, 1024)
       ↓
┌──────────────────────────────────────┐
│  Kitty Renderer                      │
│  - Receives "file1\nfile2\n"         │
│  - Parses terminal sequences         │
│  - Renders to screen via OpenGL      │
└──────┬───────────────────────────────┘
       ↓
┌──────────────┐
│    Screen    │
│   Display    │
└──────────────┘
```

### Bubble Tea App Running in Kitty Panel

```
┌─────────────────────────────────────────────────────────────────────┐
│                   Kitty Process (shine manager)                      │
│                                                                      │
│  Creates panel with: kitty @ launch --type=window shine-clock       │
└──────────────────────────────────┬───────────────────────────────────┘
                                   │ fork() + exec()
                                   ↓
┌─────────────────────────────────────────────────────────────────────┐
│                 New Kitty Window/Panel Process                       │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  PTY Master FD 18 (for this panel)                       │      │
│  └────────┬────────────────────────────────────────┬────────┘      │
│           │ write()                                │ read()         │
│           ↓                                        ↑                │
│      [Terminal Emulator Layer]                                      │
│           │                                        │                │
└───────────┼────────────────────────────────────────┼────────────────┘
            │                                        │
            │        Kernel PTY Driver               │
            │        /dev/ptmx ↔ /dev/pts/8          │
            ↓                                        ↑
┌───────────┴────────────────────────────────────────┴────────────────┐
│  ┌──────────────────────────────────────────────────────────┐      │
│  │  PTY Slave FDs in child                                  │      │
│  │  FD 0 (stdin)  → /dev/pts/8                              │      │
│  │  FD 1 (stdout) → /dev/pts/8                              │      │
│  │  FD 2 (stderr) → /dev/pts/8                              │      │
│  └────────┬────────────────────────────────────────┬────────┘      │
│           │ read(0)                                │ write(1)       │
│           ↓                                        ↑                │
│  ┌────────────────────────────────────────────────────────────┐   │
│  │              shine-clock Process                           │   │
│  │                                                            │   │
│  │  main() {                                                  │   │
│  │    p := tea.NewProgram(clockModel())                      │   │
│  │    p.Run()  // Bubble Tea takes over terminal            │   │
│  │  }                                                         │   │
│  │                                                            │   │
│  │  Bubble Tea internals:                                    │   │
│  │  - Reads from stdin (FD 0) for keyboard input             │   │
│  │  - Writes to stdout (FD 1) for rendering                  │   │
│  │  - Sends ANSI escape sequences                            │   │
│  │                                                            │   │
│  │  Update cycle:                                            │   │
│  │    model.Update(msg) → model.View() → string             │   │
│  │                                           ↓                │   │
│  │    write(1, "\x1b[2J\x1b[H23:45:12\n", ...)  [stdout]    │   │
│  └────────────────────────────────────────────────────────────┘   │
│                           Child Process                           │
│                        (shine-clock - PID 9012)                   │
└───────────────────────────────────────────────────────────────────┘
```

## Example Architectures Using Building Blocks

### Current Shine (Multi-Process)

```
┌─────────────────┐
│  shine binary   │
│  (coordinator)  │
└────────┬────────┘
         │
         ├──> kitty @ launch shine-clock
         │         ↓
         │    ┌─────────────┐
         │    │Kitty Panel 1│
         │    │PTY: pts/5   │
         │    └──────┬──────┘
         │           │
         │    ┌──────────────┐
         │    │ shine-clock  │
         │    │ tea.Program  │
         │    │ FD 0,1,2 →   │
         │    │   /dev/pts/5 │
         │    └──────────────┘
         │
         ├──> kitty @ launch shine-workspace
         │         ↓
         │    [Panel 2 with pts/6]
         │
         └──> kitty @ launch shine-spotify
                   ↓
              [Panel 3 with pts/7]
```

### Multi-Process Shine Architecture (Current Detail)

```
┌──────────────────────────────────────────────────────────────────────┐
│                    Main Kitty Instance                               │
└──────────────────────────────────────────────────────────────────────┘
                                 │
                                 │ User runs: shine
                                 ↓
┌──────────────────────────────────────────────────────────────────────┐
│                    shine (coordinator process)                        │
│                                                                       │
│  Launches 3 panels:                                                  │
│  1. kitty @ launch --type=window shine-clock                        │
│  2. kitty @ launch --type=window shine-workspace                    │
│  3. kitty @ launch --type=window shine-spotify                      │
└────┬───────────────────────┬───────────────────────┬─────────────────┘
     │                       │                       │
     │ spawn                 │ spawn                 │ spawn
     ↓                       ↓                       ↓
┌─────────────┐        ┌─────────────┐        ┌─────────────┐
│ Panel 1     │        │ Panel 2     │        │ Panel 3     │
│ (Kitty win) │        │ (Kitty win) │        │ (Kitty win) │
└──────┬──────┘        └──────┬──────┘        └──────┬──────┘
       │                      │                      │
       │ PTY Master           │ PTY Master           │ PTY Master
       │ FD 15                │ FD 18                │ FD 21
       ↓                      ↓                      ↓
    /dev/pts/5            /dev/pts/6            /dev/pts/7
       ↓                      ↓                      ↓
┌──────────────────┐   ┌──────────────────┐   ┌──────────────────┐
│ shine-clock      │   │ shine-workspace  │   │ shine-spotify    │
│ (PID 1001)       │   │ (PID 1002)       │   │ (PID 1003)       │
│                  │   │                  │   │                  │
│ FD 0: /dev/pts/5 │   │ FD 0: /dev/pts/6 │   │ FD 0: /dev/pts/7 │
│ FD 1: /dev/pts/5 │   │ FD 1: /dev/pts/6 │   │ FD 1: /dev/pts/7 │
│ FD 2: /dev/pts/5 │   │ FD 2: /dev/pts/6 │   │ FD 2: /dev/pts/7 │
│                  │   │                  │   │                  │
│ tea.NewProgram() │   │ tea.NewProgram() │   │ tea.NewProgram() │
│   ↓              │   │   ↓              │   │   ↓              │
│ Renders clock    │   │ Renders WS info  │   │ Renders Spotify  │
└──────────────────┘   └──────────────────┘   └──────────────────┘

Each process:
- Has its own PTY slave
- Writes to its own stdout
- Completely independent
- No cross-talk or coordination needed
```

### Potential Library Architecture (Single Process)

```
┌─────────────────────────────────────┐
│  User's Go Application              │
│                                     │
│  import "github.com/shine/lib"      │
│                                     │
│  shine.NewManager()                 │
│    .AddPanel(clockWidget)           │
│    .AddPanel(workspaceWidget)       │
│    .Run()                           │
└─────────────────────────────────────┘
         │
         │ Uses Shine as library
         ↓
┌─────────────────────────────────────┐
│  Shine Library (pkg/shine)          │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ PanelManager                │   │
│  │  - LaunchPanel()            │   │
│  │  - GetPanelWriter()         │   │
│  └─────────────────────────────┘   │
│                                     │
│  ┌─────────────────────────────┐   │
│  │ PanelWriter (io.Writer)     │   │
│  │  - Wraps kitten @ send-text │   │
│  │  - Implements Write()       │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

### Why Direct PTY Access is Hard

```
Goal: Single process writing to multiple panels
Problem: Permission model prevents this

┌─────────────────────────────────────────────────────────────────┐
│              shine (single process - PID 2000)                   │
│                                                                  │
│  Wants to write to multiple PTYs:                               │
│                                                                  │
│  tea.NewProgram(clock, tea.WithOutput(??))                      │
│                         What goes here? ↑                        │
└─────────────────────────────────────────────────────────────────┘
                               │
                               │ Needs writers for:
                               ↓
        ┌──────────────────────┬──────────────────────┐
        │                      │                      │
        ↓                      ↓                      ↓
   /dev/pts/5             /dev/pts/6             /dev/pts/7
   (owned by PID 1001)    (owned by PID 1002)    (owned by PID 1003)
        │                      │                      │
        ↓                      ↓                      ↓
   ┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
   │ Permission: 600 │    │ Permission: 600 │    │ Permission: 600 │
   │ Owner: user     │    │ Owner: user     │    │ Owner: user     │
   └─────────────────┘    └─────────────────┘    └─────────────────┘
        ↓                      ↓                      ↓
   ❌ open("/dev/pts/5", O_WRONLY)  → Permission denied!

   Kernel enforces: Only the process holding the MASTER FD can write
                   PID 2000 doesn't hold any of these master FDs
                   Kitty holds them, won't share them

Solution: Use Kitty's remote control API
          kitten @ send-text --match=id:X --stdin
          Writes via Kitty (which holds the master FD)
```

## Interface Building Blocks

### Core Interfaces for Library Design

```go
// Panel represents a Kitty panel
type Panel interface {
    ID() int
    Writer() io.Writer
    Close() error
}

// Manager coordinates multiple panels
type Manager interface {
    LaunchPanel(title string) (Panel, error)
    GetPanel(id int) (Panel, error)
    Shutdown() error
}

// Widget is anything that can render to a panel
type Widget interface {
    Run(output io.Writer) error
}

// BubbleTeaWidget wraps a tea.Program
type BubbleTeaWidget struct {
    Model tea.Model
}

func (w *BubbleTeaWidget) Run(output io.Writer) error {
    p := tea.NewProgram(w.Model, tea.WithOutput(output))
    _, err := p.Run()
    return err
}
```

## Template for Your Diagrams

Use this template to sketch your architecture:

```
┌─────────────────────────────────────────────────────────────┐
│                         Component Name                       │
│                                                              │
│  Purpose: [what it does]                                    │
│  Inputs:  [what it receives]                                │
│  Outputs: [what it produces]                                │
│  State:   [what it stores]                                  │
│                                                              │
│  Key methods/operations:                                    │
│  - Method1()                                                │
│  - Method2()                                                │
└─────────────────────────────────────────────────────────────┘
         ↓ data flow direction
┌─────────────────────────────────────────────────────────────┐
│                      Next Component                          │
└─────────────────────────────────────────────────────────────┘
```

## Questions to Guide Your Design

When creating your architecture diagrams, consider:

1. **Process Boundary**: Where does single process end and multi-process begin?
2. **I/O Abstraction**: What io.Writer/Reader implementations do you need?
3. **Lifecycle**: Who owns creation/destruction of panels?
4. **Communication**: How do widgets talk to each other (if needed)?
5. **Configuration**: How does user configure panels?
6. **Error Handling**: What happens if a panel crashes?
7. **Library API**: What's the public interface for users?

## Usage Example for Your Library Idea

```go
package main

import (
    "github.com/yourusername/shine/pkg/shine"
    tea "github.com/charmbracelet/bubbletea"
)

func main() {
    // Create manager (single process)
    mgr := shine.NewManager()

    // Launch 3 panels via Kitty remote control
    panel1, _ := mgr.LaunchPanel("Clock")
    panel2, _ := mgr.LaunchPanel("Workspace")
    panel3, _ := mgr.LaunchPanel("Spotify")

    // Run 3 Bubble Tea programs (same process)
    go tea.NewProgram(clockModel(), tea.WithOutput(panel1.Writer())).Run()
    go tea.NewProgram(workspaceModel(), tea.WithOutput(panel2.Writer())).Run()
    go tea.NewProgram(spotifyModel(), tea.WithOutput(panel3.Writer())).Run()

    // Keep running
    mgr.Wait()
}
```

## Key Takeaways

1. **PTY Master** (Kitty side):
   - File descriptor only (no path)
   - Held by terminal emulator
   - Writes send data to child's stdin
   - Reads receive data from child's stdout/stderr

2. **PTY Slave** (Child side):
   - Has filesystem path `/dev/pts/N`
   - Opened by child process
   - Becomes FD 0, 1, 2 (stdin, stdout, stderr)
   - Permission 600 (owner-only)

3. **Security Model**:
   - Only master FD holder can write to slave
   - Prevents external processes from injecting commands
   - This is why you can't `echo "text" > /dev/pts/5`

4. **Shine's Architecture Options**:
   - **Current (multi-process)**: Each panel = separate process with own PTY
   - **Library (single-process)**: One process, multiple tea.Programs, using Kitty remote control for I/O
   - **Hybrid**: Mix of both approaches based on use case
