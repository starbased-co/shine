# Quick Test Guide - Phase 4 PTY-per-Process

## 1. Build Everything

```bash
cd /home/starbased/dev/projects/shine

# Build prismctl
go build -o bin/prismctl ./cmd/prismctl

# Build test prism
go build -o test/fixtures/test-prism ./test/fixtures/test_prism.go

# Build example prisms (optional)
go build -o bin/shine-clock ./cmd/shine-clock
```

## 2. Quick Test - Single Prism

**Open a terminal and run:**

```bash
./bin/prismctl panel-test ./test/fixtures/test-prism
```

**Expected output:**
- `=== test-prism started (PID XXXXX) ===`
- `TTY: /dev/pts/X` (shows it has a real PTY)
- `TERM: xterm-kitty` (preserves Kitty terminal type)
- `[test-prism] tick 1 at HH:MM:SS` (prints every second)

**Press Ctrl+C to stop**

## 3. Multi-Prism Test with IPC

**Terminal 1** - Start prismctl:

```bash
./bin/prismctl panel-test ./test/fixtures/test-prism initial-prism
```

**Terminal 2** - Send IPC commands:

```bash
# Find the IPC socket
SOCK=$(ls /run/user/$(id -u)/shine/prism-panel-*.sock 2>/dev/null | head -1)
echo "Using socket: $SOCK"

# Launch prism A (goes to background, initial-prism stays foreground)
echo '{"action":"start","prism":"./test/fixtures/test-prism prism-A"}' | nc -U $SOCK

# Launch prism B (also background)
echo '{"action":"start","prism":"./test/fixtures/test-prism prism-B"}' | nc -U $SOCK

# Check status
echo '{"action":"status"}' | nc -U $SOCK | jq

# Swap to prism-A (watch Terminal 1 - should instantly switch!)
echo '{"action":"start","prism":"./test/fixtures/test-prism prism-A"}' | nc -U $SOCK

# Swap to prism-B
echo '{"action":"start","prism":"./test/fixtures/test-prism prism-B"}' | nc -U $SOCK

# Notice: Background prisms keep ticking! (check tick numbers increase)

# Kill a specific prism
echo '{"action":"kill","prism":"./test/fixtures/test-prism prism-A"}' | nc -U $SOCK

# Stop everything
echo '{"action":"stop"}' | nc -U $SOCK
```

## 4. Verify Background Processing

1. Launch two prisms (A and B)
2. Watch prism-A in foreground, note the tick count
3. Swap to prism-B, wait 5 seconds
4. Swap back to prism-A
5. **Verify**: Tick count increased by ~5 (proves it kept running in background!)

## 5. Test Terminal Resize

With prismctl running:
1. Resize your terminal window
2. Prism should automatically redraw at new size
3. All background prisms also get the new size (SIGWINCH to all PTYs)

## 6. Run Unit Tests

```bash
# Run prismctl unit tests
go test ./cmd/prismctl -v

# Run all tests
go test ./...

# Run specific integration test (requires PTY)
cd test/integration
go test -v -run TestBasicCompilation
```

## 7. Test with Real Prism (shine-clock)

```bash
./bin/prismctl panel-test ./bin/shine-clock
```

This runs the actual clock prism instead of the test fixture.

## What to Look For

✅ **Success indicators:**
- Prisms launch with `TTY: /dev/pts/X` (real PTY allocated)
- `TERM: xterm-kitty` (not tmux-256color)
- Background prisms continue ticking
- Hot-swap between prisms is instant (<50ms)
- Resize works without errors
- No FD leaks (check with `lsof -p <prismctl-pid>`)

❌ **Failure indicators:**
- `ERROR: stdin is not a TTY`
- Prisms freeze when backgrounded (SIGSTOP still in code)
- Swap takes >1 second
- PTY FD leak (hundreds of open FDs)
- Crashes on SIGWINCH

## Troubleshooting

**If prismctl won't start:**
```bash
# Check for stale sockets
rm -f /run/user/$(id -u)/shine/prism-*.sock

# Check TERM is set
echo $TERM  # Should show xterm-kitty or similar
```

**If IPC doesn't work:**
```bash
# Verify socket exists
ls -la /run/user/$(id -u)/shine/

# Try with absolute path
nc -U /run/user/$(id -u)/shine/prism-panel-0.12345.sock
```

**To see debug logs:**
Check prismctl stderr output for relay and PTY messages.

## Architecture Verification

After testing, verify the architecture matches the design:

1. **Each prism has dedicated PTY**: `lsof -p <pid>` shows unique /dev/pts/X per child
2. **No SIGSTOP**: `ps aux | grep test-prism` - all prisms show "S" (sleeping), not "T" (stopped)
3. **Hot-swap works**: Swap latency logged in prismctl output
4. **Background processing**: Tick counters increase even when not visible

## Next Steps

Once basic functionality works:
1. Test with real Kitty panel: `kitten panel ./bin/prismctl ...`
2. Run full integration tests: `./test/e2e_test.sh`
3. Stress test: Launch 10+ prisms, swap rapidly
4. Memory test: Monitor RSS with `ps` while swapping
