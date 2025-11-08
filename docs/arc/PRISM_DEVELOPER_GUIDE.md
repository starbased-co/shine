# Prism Developer Guide

Complete guide for creating custom Shine prisms.

## Table of Contents

- [Getting Started](#getting-started)
- [What is a Prism?](#what-is-a-prism)
- [Prism Interface Requirements](#prism-interface-requirements)
- [Development Workflow](#development-workflow)
- [Best Practices](#best-practices)
- [Advanced Topics](#advanced-topics)
- [API Reference](#api-reference)
- [Example Walkthroughs](#example-walkthroughs)
- [Troubleshooting](#troubleshooting)

## Getting Started

### Prerequisites

- Go 1.21 or later
- Shine installed and configured
- Basic knowledge of Bubble Tea framework
- Kitty terminal with panel support

### Quick Start

Create a new prism in seconds:

```bash
# Generate prism template
shinectl new-prism my-widget

# Navigate to prism directory
cd ~/.config/shine/prisms/my-widget

# Build the prism
make build

# Install to PATH
make install

# Configure in shine.toml
# Add configuration, then launch shine
```

See the generated README.md for detailed instructions.

## What is a Prism?

A **prism** is a self-contained Bubble Tea application that displays information in a Shine panel. Prisms "refract" system information (CPU, weather, music, etc.) into beautiful TUI displays.

### Key Characteristics

- **Standard Bubble Tea App**: Uses familiar Model-Update-View pattern
- **No Alt Screen**: Renders in normal terminal mode (NOT full-screen)
- **Discoverable**: Follows naming convention (`shine-<name>`)
- **Configurable**: Accepts standard panel configuration from `shine.toml`
- **Composable**: Multiple prisms can run simultaneously

### Prism vs Full-Screen TUI

| Feature | Prism | Full-Screen TUI |
|---------|-------|-----------------|
| Screen Mode | Normal | Alt Screen |
| Location | Fixed panel position | Entire terminal |
| Purpose | Status/widget display | Interactive application |
| Quit Behavior | Managed by Shine | User quits manually |
| Examples | Weather, CPU monitor | Text editor, file manager |

## Prism Interface Requirements

All prisms MUST follow these conventions to work with Shine:

### 1. Window Title (REQUIRED)

Set the window title to `shine-<name>` for Shine to track and manage the window:

```go
fmt.Print("\033]0;shine-myprism\007")
```

This MUST be the first thing printed, before creating the Bubble Tea program.

### 2. No Alt Screen (REQUIRED)

Do NOT use `tea.WithAltScreen()` when creating your program:

```go
// CORRECT
p := tea.NewProgram(initialModel())

// WRONG - breaks panel rendering
p := tea.NewProgram(initialModel(), tea.WithAltScreen())
```

Panels render in the normal terminal buffer, not an alternate screen.

### 3. Binary Naming (REQUIRED)

Binary MUST be named `shine-<name>`:

- Prism "weather" → binary `shine-weather`
- Prism "spotify" → binary `shine-spotify`
- Prism "my-widget" → binary `shine-my-widget`

### 4. Clean Exit (REQUIRED)

Handle quit signals gracefully:

```go
case tea.KeyMsg:
	switch msg.Type {
	case tea.KeyEsc, tea.KeyCtrlC:
		return m, tea.Quit
	}
```

### 5. Responsive (RECOMMENDED)

Handle terminal resize events:

```go
case tea.WindowSizeMsg:
	m.width = msg.Width
	m.height = msg.Height
	return m, nil
```

## Development Workflow

### Step 1: Create Prism

```bash
# Generate template
shinectl new-prism weather

# Or manually create directory structure
mkdir -p ~/.config/shine/prisms/weather
cd ~/.config/shine/prisms/weather
```

### Step 2: Implement Functionality

Edit `main.go` to customize:

```go
package main

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Set window title
	fmt.Print("\033]0;shine-weather\007")

	// Run program
	p := tea.NewProgram(initialModel())
	p.Run()
}

type model struct {
	// Your state here
}

func (m model) Init() tea.Cmd {
	// Initial command
	return nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle messages
	return m, nil
}

func (m model) View() string {
	// Render UI with lipgloss
	return "Hello from prism!"
}
```

### Step 3: Build and Test

```bash
# Build binary
make build

# Test standalone (opens in terminal window)
./shine-weather

# Test with kitty panel (proper panel environment)
kitten panel --edge=top --lines=50px ./shine-weather
```

### Step 4: Install

```bash
# Install to ~/.local/bin (ensure in PATH)
make install

# Verify installation
which shine-weather
```

### Step 5: Configure

Edit `~/.config/shine/shine.toml`:

```toml
[prisms.weather]
enabled = true
origin = "top-right"
width = "300px"
height = "50px"
position = "0,0"
focus_policy = "not-allowed"
output_name = "DP-2"
```

### Step 6: Launch

```bash
# Launch all configured prisms
shine
```

## Best Practices

### Panel-Friendly Design

#### Size Considerations

Panels are space-constrained. Design for efficiency:

```go
// Horizontal layout for top/bottom edges (limited height)
lipgloss.JoinHorizontal(lipgloss.Top, elements...)

// Vertical layout for left/right edges (limited width)
lipgloss.JoinVertical(lipgloss.Left, elements...)
```

#### High Contrast Colors

Use bright ANSI colors for visibility:

```go
// Good: Bright colors (8-15)
style := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // Bright green

// Avoid: Dim colors (0-7)
style := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))  // Dim green
```

#### Compact Information

Prioritize essential information:

```go
// Good: Compact
"CPU: 45% ███████░░░"

// Avoid: Verbose
"Current CPU Usage: 45 percent [|||||||   ]"
```

### Update Frequency

Choose appropriate refresh rates:

```go
// Fast: System monitors (1-2 seconds)
tea.Tick(2*time.Second, ...)

// Medium: Spotify/music (5 seconds)
tea.Tick(5*time.Second, ...)

// Slow: Weather/static info (5-15 minutes)
tea.Tick(15*time.Minute, ...)
```

### Efficient Rendering

Minimize unnecessary redraws:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Only update if data actually changed
		newData := fetchData()
		if newData != m.data {
			m.data = newData
		}
		return m, tickCmd()
	}
	return m, nil
}
```

### Resource Cleanup

Clean up resources on exit:

```go
type model struct {
	conn *Connection
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.QuitMsg:
		// Clean up before quitting
		if m.conn != nil {
			m.conn.Close()
		}
	}
	return m, nil
}
```

### Error Handling

Handle errors gracefully, don't crash:

```go
func fetchData() (data, error) {
	resp, err := http.Get(url)
	if err != nil {
		// Return cached/default data instead of panicking
		return defaultData, nil
	}
	// ...
}
```

### Logging

Log to stderr, not stdout:

```go
import "log"

func main() {
	// Configure logger to stderr
	log.SetOutput(os.Stderr)

	// Log errors
	if err := something(); err != nil {
		log.Printf("Error: %v", err)
	}
}
```

## Advanced Topics

### Custom Configuration Fields

**Recommended approach:** Use the `metadata` field in your `prism.toml`:

```toml
# In your prism's prism.toml file
name = "weather"
version = "1.0.0"
enabled = false  # Default, user can override in shine.toml

[metadata]
api_key = "default-api-key"
location = "San Francisco"
units = "imperial"
refresh_interval = 300
```

**Important:** Custom fields in shine.toml `[prisms.weather]` sections are NOT preserved. Use `metadata` in prism.toml for prism-specific configuration.

Users can override standard fields in shine.toml:

```toml
[prisms.weather]
enabled = true
origin = "top-right"
width = "300px"
height = "50px"
```

Read metadata in your prism:

```go
import (
	"github.com/BurntSushi/toml"
	"os"
	"path/filepath"
)

type PrismConfig struct {
	Name     string                 `toml:"name"`
	Version  string                 `toml:"version"`
	Metadata map[string]interface{} `toml:"metadata"`
}

func loadConfig() PrismConfig {
	// Look for prism.toml in prism directory
	prismDir := os.Getenv("PRISM_DIR")
	if prismDir == "" {
		prismDir = "."
	}
	configPath := filepath.Join(prismDir, "prism.toml")

	var cfg PrismConfig
	if _, err := toml.DecodeFile(configPath, &cfg); err != nil {
		// Use defaults
		return PrismConfig{
			Name:     "weather",
			Metadata: make(map[string]interface{}),
		}
	}

	return cfg
}

func main() {
	cfg := loadConfig()

	// Access metadata
	apiKey, _ := cfg.Metadata["api_key"].(string)
	location, _ := cfg.Metadata["location"].(string)
	units, _ := cfg.Metadata["units"].(string)

	// Use configuration...
}
```

### Inter-Prism Communication

Prisms can communicate via:

1. **Shared Files**: Write to `~/.cache/shine/` or `/tmp/shine/`
2. **Unix Sockets**: Create socket for IPC
3. **D-Bus**: Use system message bus (Linux)

Example shared state:

```go
// Writer prism
func writeState(data string) {
	os.WriteFile("/tmp/shine-state.json", []byte(data), 0644)
}

// Reader prism
func readState() string {
	data, _ := os.ReadFile("/tmp/shine-state.json")
	return string(data)
}
```

### External API Integration

Fetch data from external APIs:

```go
import (
	"encoding/json"
	"net/http"
)

func fetchWeather(apiKey, city string) (WeatherData, error) {
	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?q=%s&appid=%s", city, apiKey)

	resp, err := http.Get(url)
	if err != nil {
		return WeatherData{}, err
	}
	defer resp.Body.Close()

	var data WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return WeatherData{}, err
	}

	return data, nil
}

// Integrate with Bubble Tea commands
func fetchWeatherCmd(apiKey, city string) tea.Cmd {
	return func() tea.Msg {
		data, err := fetchWeather(apiKey, city)
		if err != nil {
			return errorMsg{err}
		}
		return weatherMsg{data}
	}
}
```

### State Persistence

Save state between sessions:

```go
import (
	"encoding/json"
	"os"
	"path/filepath"
)

type PersistentState struct {
	LastUpdate time.Time
	CachedData interface{}
}

func saveState(state PersistentState) error {
	home, _ := os.UserHomeDir()
	stateFile := filepath.Join(home, ".cache/shine/myprism-state.json")

	os.MkdirAll(filepath.Dir(stateFile), 0755)

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return os.WriteFile(stateFile, data, 0644)
}

func loadState() (PersistentState, error) {
	home, _ := os.UserHomeDir()
	stateFile := filepath.Join(home, ".cache/shine/myprism-state.json")

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return PersistentState{}, err
	}

	var state PersistentState
	err = json.Unmarshal(data, &state)
	return state, err
}
```

### Async Operations

Handle long-running operations without blocking:

```go
type fetchMsg struct {
	data interface{}
	err  error
}

func fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		// This runs in a goroutine
		data, err := fetchExpensiveData()
		return fetchMsg{data, err}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case fetchMsg:
		if msg.err != nil {
			m.error = msg.err
			return m, nil
		}
		m.data = msg.data
		return m, nil
	}
	return m, nil
}
```

## API Reference

### Panel Configuration

Available in `~/.config/shine/shine.toml`:

```toml
[prisms.myprism]
enabled = true              # Launch this prism

# Positioning & Layout
origin = "top-center"       # Anchor point on screen
position = "0,0"            # Offset from origin as "x,y" in pixels
width = 80                  # Width in columns (int) or pixels (string with "px")
height = 10                 # Height in lines (int) or pixels (string with "px")

# Alternatively, use pixel dimensions:
# width = "300px"
# height = "100px"

# Behavior
focus_policy = "not-allowed"  # not-allowed, on-demand, exclusive
hide_on_focus_loss = false    # Hide when panel loses focus
output_name = "DP-2"          # Target specific monitor

# Binary (optional)
path = "shine-myprism"    # Custom path or binary name (defaults to shine-{name})
```

**See [docs/configuration.md](configuration.md) for complete field documentation.**

### Origin (Anchor Point) Options

- `top-left` - Top-left corner
- `top-center` - Top edge, centered horizontally
- `top-right` - Top-right corner
- `left-center` - Left edge, centered vertically
- `center` - Screen center
- `right-center` - Right edge, centered vertically
- `bottom-left` - Bottom-left corner
- `bottom-center` - Bottom edge, centered horizontally
- `bottom-right` - Bottom-right corner

### Focus Policies

- `not-allowed` - Never receives keyboard focus (status displays)
- `on-demand` - Can receive focus when clicked (interactive widgets)
- `exclusive` - Always has focus when visible (rarely used)

### Kitty Remote Control

Shine uses Kitty's remote control protocol (`kitty @`) for panel management. Prisms don't need to interact with this directly, but it's available for advanced use:

```bash
# List windows
kitty @ ls

# Close panel
kitty @ close-window --match title:shine-weather

# Get window info
kitty @ ls | jq '.[] | .tabs[] | .windows[] | select(.title == "shine-weather")'
```

**Note:** Shine automatically handles panel launching via `kitty @ launch --type=os-panel`.

### Hyprland Integration

Prisms render via Kitty's Wayland layer shell integration. No direct Hyprland API calls needed, but you can query Hyprland:

```bash
# Get monitor info
hyprctl monitors -j

# Get workspace info
hyprctl workspaces -j
```

## Example Walkthroughs

### Example 1: Simple Counter

Minimal prism that increments a counter:

```go
package main

import (
	"fmt"
	"time"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	fmt.Print("\033]0;shine-counter\007")
	tea.NewProgram(model{count: 0}).Run()
}

type tickMsg time.Time

type model struct {
	count int
}

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case tickMsg:
		m.count++
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	return m, nil
}

func (m model) View() string {
	return fmt.Sprintf("Count: %d", m.count)
}
```

### Example 2: Clock

Digital clock with styled output:

```go
package main

import (
	"fmt"
	"time"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	fmt.Print("\033]0;shine-clock\007")
	tea.NewProgram(model{}).Run()
}

type tickMsg time.Time

type model struct{}

func (m model) Init() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tea.KeyMsg:
		return m, tea.Quit
	case tickMsg:
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return tickMsg(t)
		})
	}
	return m, nil
}

func (m model) View() string {
	now := time.Now()

	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).
		Bold(true).
		Padding(0, 1)

	dateStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("12")).
		Padding(0, 1)

	timeStr := timeStyle.Render(now.Format("15:04:05"))
	dateStr := dateStyle.Render(now.Format("Mon Jan 2"))

	return lipgloss.JoinHorizontal(lipgloss.Top, timeStr, dateStr)
}
```

### Example 3: Weather (Annotated)

See `examples/prisms/weather/` for a complete, annotated weather prism demonstrating:

- API integration (simulated)
- Periodic updates
- Complex layout with lipgloss
- Icons and styling
- Error handling

## Troubleshooting

### Prism Not Launching

```bash
# Check if binary exists and is executable
which shine-myprism
ls -l ~/.local/bin/shine-myprism

# Check if enabled in config
grep -A5 'prisms.myprism' ~/.config/shine/shine.toml

# Test standalone
shine-myprism

# Check shine logs
shine  # Will show launch errors
```

### Window Title Not Set

Symptom: Shine can't control the prism window.

Fix: Ensure window title is set BEFORE creating Bubble Tea program:

```go
func main() {
	// MUST be first
	fmt.Print("\033]0;shine-myprism\007")

	// Then create program
	p := tea.NewProgram(...)
}
```

### Alt Screen Mode Error

Symptom: Prism content not visible or panel appears blank.

Fix: Remove `tea.WithAltScreen()`:

```go
// Wrong
p := tea.NewProgram(model{}, tea.WithAltScreen())

// Correct
p := tea.NewProgram(model{})
```

### Binary Not Found

Symptom: `prism myprism not found (binary: shine-myprism)`

Fix:

1. Ensure binary is named correctly: `shine-<name>`
2. Install to PATH: `make install` or copy to `~/.local/bin/`
3. Verify PATH includes `~/.local/bin/`: `echo $PATH`
4. Or configure discovery paths in config:

```toml
[core]
path = [
	"~/.config/shine/prisms",
	"~/.local/bin",
	"~/.local/share/shine/bin",
]
```

### Panel Size Issues

Symptom: Content cut off or wrapped incorrectly.

Fix: Handle `tea.WindowSizeMsg` and adjust layout:

```go
case tea.WindowSizeMsg:
	m.width = msg.Width
	m.height = msg.Height
	// Adjust view based on available space
	return m, nil
```

### High CPU Usage

Symptom: Prism consumes excessive CPU.

Fix: Increase tick interval:

```go
// Too fast
tea.Tick(100*time.Millisecond, ...)

// Better
tea.Tick(2*time.Second, ...)
```

### Memory Leaks

Symptom: Memory usage grows over time.

Fix:

1. Limit cache sizes
2. Clean up old data
3. Close connections in quit handler
4. Profile with `pprof`:

```go
import _ "net/http/pprof"

func main() {
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()
	// ... rest of main
}
```

Then: `go tool pprof http://localhost:6060/heap`

## Further Reading

- [Shine Documentation](../README.md)
- [Bubble Tea Tutorial](https://github.com/charmbracelet/bubbletea/tree/master/tutorials)
- [Lip Gloss Examples](https://github.com/charmbracelet/lipgloss/tree/master/examples)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Example Prisms](../examples/prisms/)

## Community

- Share your prisms on GitHub with the `shine-prism` topic
- Join discussions in Shine issues/discussions
- Contribute examples to the repository

## License

Prisms you create are your own and can use any license you choose.
