- move newprism from shined to shine
- @cmd/shined/newprism.go#L102
  for \_, ch := range name {
  if !((ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-') {
  return false
  }
  redo this
- regarding shine/output.go
  You already have lipgloss v1.1+ which includes lipgloss/table.

  What output.go provides:
  1. Message helpers (Success, Error, Warning, Info, Muted, Header) - simple icon + style wrappers, ~6 lines each. These are fine.
  2. Table struct (~60 lines) - manual column width calculation, padding, rendering
  3. StatusBox - custom status line

  Candidate for replacement:
  The Table implementation could use lipgloss/table:

  import "github.com/charmbracelet/lipgloss/table"

  // Instead of manual Table struct
  t := table.New().
  Headers("Prism", "PID", "State", "Uptime").
  Row("clock", "1234", "foreground", "5m").
  Row("bar", "1235", "background", "5m")
  fmt.Println(t)

  Worth refactoring for v1.0? It's ~60 lines of working code. Could be a post-v1.0 cleanup unless you want consistent Charm usage throughout.

- shine/commands.go
  The user is asking about the architecture of cmd/shine/commands.go - specifically questioning why the CLI has functions like discoverPrismInstances, displayPanelStatus, etc.
  when these could be:
  1. RPC methods that shined exposes
  2. State file reads
  3. Part of the daemon rather than the client

  Let me analyze what's in commands.go:
  1. connectShined() - connects to shined socket
  2. isShinedRunning() - checks if socket exists
  3. discoverPrismInstances() - scans for prism-\*.sock files
  4. findPrismctlSockets() - legacy helper wrapping discoverPrismInstances
  5. cmdStart() - starts shined
  6. cmdStop() - stops shine service
  7. cmdReload() - reloads config
  8. displayStateFromMmap() - displays state from mmap file
  9. displayStateFromRPC() - displays state from RPC response
  10. cmdStatus() - shows status
  11. displayPanelStatus() - displays a single panel's status
  12. cmdLogs() - shows logs
  13. extractInstanceName() - extracts instance name from socket path

  The user's concern is valid. Let me think about what should be where:

  Currently in CLI (shine) that could be in daemon (shined):
  - discoverPrismInstances() - The CLI is doing filesystem scanning for sockets. This is a fallback when shined isn't running, but when shined IS running, it should provide
    this info via RPC.
  - findPrismctlSockets() - Same issue

  Current flow for cmdStatus():
  1. Try shined RPC first → client.Status(ctx) returns panels list
  2. If that fails, fallback to discoverPrismInstances() (filesystem scan)

  3. For each panel, call displayPanelStatus() which:
  - Tries mmap first
  - Falls back to prismctl RPC

  I see a multi-layered approach to state retrieval. The CLI implements resilient discovery mechanisms, checking multiple sources to ensure it can retrieve panel status. When
  the daemon is running, it prioritizes RPC calls, but maintains filesystem-based fallback methods for scenarios where the daemon might be unavailable. This design ensures
  robust information retrieval across different system states.

  The strategy involves progressive fallback: first attempting RPC communication, then memory-mapped files, and finally filesystem scanning. Each method provides a different
  level of access to system state, creating a comprehensive information retrieval strategy that can adapt to varying system conditions.

  The key consideration is maintaining system visibility even under degraded operational scenarios, allowing partial status reporting when full daemon communication isn't
  possible. This approach prioritizes information accessibility over strict communication protocols.

● Good architectural question. Let me break down what's in commands.go:

Discovery/Query functions (CLI-side):

- discoverPrismInstances() - scans filesystem for prism-\*.sock
- findPrismctlSockets() - legacy wrapper
- displayPanelStatus() - tries mmap, falls back to RPC
- displayStateFromMmap() / displayStateFromRPC() - rendering

Current flow for shine status:

1. Try shined RPC → client.Status() returns panel list
2. If shined down → discoverPrismInstances() (filesystem scan)
3. For each panel:
   a. Try mmap state file (instant, no connection)
   b. Fallback to prismctl RPC

What shined already provides via RPC:

- service/status → panels list, version, uptime
- panel/list → panel instances

The redundancy:

| Function                 | CLI does                         | Should be                                     |
| ------------------------ | -------------------------------- | --------------------------------------------- |
| discoverPrismInstances() | Filesystem scan                  | shined RPC (it already tracks spawned panels) |
| Panel state query        | CLI reads mmap OR calls prismctl | shined could aggregate                        |

Why it exists (resilience):
The fallbacks let CLI show partial status even when shined crashed but prismctl instances are still running.

Cleaner architecture options:

1. shined aggregates everything - CLI only talks to shined, shined talks to prismctl instances
2. State files as source of truth - CLI reads mmap directly, no RPC needed for reads
3. Current hybrid - Works but duplicates discovery logic

Which direction interests you? Option 2 (state files) aligns with the existing mmap infrastructure.
