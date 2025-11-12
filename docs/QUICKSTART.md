# Shine Quick Start Guide

Get up and running with Shine's prismtty architecture in 5 minutes.

---

## Prerequisites

- Kitty terminal with remote control enabled
- Go 1.21+ installed
- Prism binaries in PATH (e.g., shine-clock, shine-sysinfo)

---

## Installation

### 1. Build Binaries

```bash
cd /path/to/shine
go build -o bin/prismctl ./cmd/prismctl
go build -o bin/shinectl ./cmd/shinectl
go build -o bin/shine ./cmd/shine
```

### 2. Install to PATH

```bash
# Option A: Add bin/ to PATH
export PATH="$PATH:$(pwd)/bin"

# Option B: Copy to ~/.local/bin
cp bin/* ~/.local/bin/
```

### 3. Verify Installation

```bash
prismctl --help
shinectl --help
shine --help
```

---

## Configuration

### Create Config Directory

```bash
mkdir -p ~/.config/shine
```

### Create prism.toml

```bash
cat > ~/.config/shine/prism.toml <<'EOF'
# Example configuration with 3 prisms

[[prism]]
name = "shine-clock"
restart = "on-failure"
restart_delay = "3s"
max_restarts = 5

[[prism]]
name = "shine-sysinfo"
restart = "unless-stopped"
restart_delay = "5s"
max_restarts = 10

[[prism]]
name = "shine-chat"
restart = "no"
EOF
```

### Verify Config

```bash
cat ~/.config/shine/prism.toml
```

---

## First Run

### Start the Service

```bash
shine start
```

Expected output:
```
ℹ Starting shinectl service...
✓ shinectl started (PID: 12345)
```

### Check Status

```bash
shine status
```

Expected output:
```
Shine Status (3 panel(s))

Panel: panel-0
Socket: /run/user/1000/shine/prism-panel-0.12346.sock
Foreground: shine-clock │ Background: 0 │ Total: 1

Prism       PID    State
──────────  ─────  ──────────
shine-clock 12347  foreground
```

### View Logs

```bash
shine logs
```

Expected output:
```
Log File         Size
───────────────  ──────────
shinectl.log     1234 bytes

ℹ View a log with: shine logs <filename>
```

### View Specific Log

```bash
shine logs shinectl
```

---

## Basic Operations

### Hot-Reload Configuration

1. Edit `~/.config/shine/prism.toml`
2. Reload: `pkill -HUP shinectl`
3. Verify: `shine status`

### Stop All Panels

```bash
shine stop
```

### Restart Service

```bash
shine stop
shine start
```

---

## Working with Panels

### Switch Prisms in a Panel

Use prismctl IPC (from another terminal):

```bash
# Find socket
ls /run/user/$(id -u)/shine/prism-*.sock

# Send start command
echo '{"action":"start","prism":"shine-sysinfo"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

### Check Panel Status

```bash
echo '{"action":"status"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

### Kill a Prism in Panel

```bash
echo '{"action":"kill","prism":"shine-clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

---

## Testing Crash Recovery

### Simulate a Crash

1. Find prism PID: `shine status`
2. Kill it: `kill -9 <pid>`
3. Watch logs: `tail -f ~/.local/share/shine/logs/shinectl.log`
4. Verify restart based on policy

### Expected Behavior

With `restart = "on-failure"`:
- Prism killed → Waits restart_delay → Restarts
- Logs show: "Panel panel-X crashed", "Restarting panel..."

With `restart = "no"`:
- Prism killed → Stays stopped
- No automatic restart

---

## Troubleshooting

### Problem: `shine start` fails with "socket not created"

**Solution**: Check if shinectl is actually running
```bash
ps aux | grep shinectl
```

### Problem: No panels appear

**Solution**: Check shinectl logs
```bash
tail -f ~/.local/share/shine/logs/shinectl.log
```

Common issues:
- Prism binary not in PATH
- Kitty remote control not enabled
- Config file syntax error

### Problem: Panel shows "socket not found"

**Solution**: Verify prismctl socket exists
```bash
ls -la /run/user/$(id -u)/shine/
```

If missing, prismctl may have crashed. Check logs.

### Problem: Prisms not restarting after crash

**Solution**: Check restart policy in prism.toml
```bash
grep -A 3 "name = \"your-prism\"" ~/.config/shine/prism.toml
```

Ensure `restart` is set correctly.

---

## Advanced Configuration

### Restart Policies Explained

```toml
# Never restart (default)
restart = "no"

# Restart only on crash/error
restart = "on-failure"

# Always restart unless explicitly stopped
restart = "unless-stopped"

# Always restart, even on clean exit
restart = "always"
```

### Restart Delay Format

```toml
restart_delay = "500ms"  # Milliseconds
restart_delay = "5s"     # Seconds
restart_delay = "1m"     # Minutes
```

### Max Restarts

```toml
max_restarts = 0   # Unlimited (default)
max_restarts = 5   # Max 5 restarts per hour
max_restarts = 10  # Max 10 restarts per hour
```

After exceeding max_restarts, prism stays stopped until manually restarted or 1 hour passes.

---

## Common Workflows

### Development Workflow

```bash
# 1. Start service
shine start

# 2. Edit prism code
vim cmd/shine-clock/main.go

# 3. Rebuild prism
go build -o ~/.local/bin/shine-clock ./cmd/shine-clock

# 4. Hot-swap in panel (prismctl IPC)
echo '{"action":"start","prism":"shine-clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock

# 5. Iterate
```

### Production Workflow

```bash
# 1. Configure prisms
vim ~/.config/shine/prism.toml

# 2. Start service
shine start

# 3. Monitor status
watch -n 5 shine status

# 4. Hot-reload config changes
pkill -HUP shinectl

# 5. Check logs periodically
shine logs shinectl
```

---

## Next Steps

1. **Read Full Documentation**
   - Architecture: `docs/tour.md` - Complete system overview
   - Configuration: `docs/configuration.md` - Complete config reference

2. **Build Your Own Prisms**
   - See example: `cmd/shine-clock/`
   - Template: `cmd/shinectl/templates/`

3. **Customize Configuration**
   - See: `docs/prism.toml.example`

4. **Set Up Autostart** (optional)
   - Systemd user service
   - Hyprland exec-once
   - Shell profile (.bashrc/.zshrc)

---

## Getting Help

### Check Logs

```bash
# Service log
tail -f ~/.local/share/shine/logs/shinectl.log

# All logs
ls -lh ~/.local/share/shine/logs/
```

### Verify Sockets

```bash
# All shine sockets
ls -la /run/user/$(id -u)/shine/

# Prismctl sockets
ls -la /run/user/$(id -u)/shine/prism-*.sock
```

### Test IPC Manually

```bash
# Status command
echo '{"action":"status"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock

# Start command
echo '{"action":"start","prism":"shine-clock"}' | nc -U /run/user/$(id -u)/shine/prism-panel-0.*.sock
```

---

## Quick Reference

### shine Commands

```bash
shine start           # Start service
shine stop            # Stop all panels
shine reload          # Reload config (via SIGHUP)
shine status          # Show panel status
shine logs            # List log files
shine logs <file>     # View specific log
shine --help          # Show help
shine --version       # Show version
```

### shinectl Signals

```bash
pkill -HUP shinectl   # Reload config
pkill -TERM shinectl  # Graceful shutdown
pkill -INT shinectl   # Interrupt (same as TERM)
```

### prismctl IPC Actions

```json
{"action":"start","prism":"shine-clock"}    // Launch/resume prism
{"action":"kill","prism":"shine-clock"}     // Kill prism
{"action":"status"}                         // Query status
{"action":"stop"}                           // Stop prismctl
```

---

**Happy Shining! ✨**
