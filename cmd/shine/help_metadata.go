package main

import (
	_ "embed"
)

//go:embed help/start.md
var startHelp string

//go:embed help/stop.md
var stopHelp string

//go:embed help/status.md
var statusHelp string

//go:embed help/reload.md
var reloadHelp string

//go:embed help/logs.md
var logsHelp string

// CommandHelp defines structured metadata for command help
type CommandHelp struct {
	Name     string // Command name
	Category string // Command category for grouping
	Synopsis string // Brief one-line description
	Usage    string // Usage syntax
	Content  string // Full help content (markdown)
}

// helpRegistry contains all command help metadata
var helpRegistry = map[string]*CommandHelp{
	"start": {
		Name:     "start",
		Category: "Service Management",
		Synopsis: "Start the shine service",
		Usage:    "shine start",
		Content:  startHelp,
	},
	"stop": {
		Name:     "stop",
		Category: "Service Management",
		Synopsis: "Stop all panels",
		Usage:    "shine stop",
		Content:  stopHelp,
	},
	"status": {
		Name:     "status",
		Category: "Monitoring",
		Synopsis: "Show panel status",
		Usage:    "shine status",
		Content:  statusHelp,
	},
	"reload": {
		Name:     "reload",
		Category: "Configuration",
		Synopsis: "Reload configuration",
		Usage:    "shine reload",
		Content:  reloadHelp,
	},
	"logs": {
		Name:     "logs",
		Category: "Monitoring",
		Synopsis: "View logs",
		Usage:    "shine logs [panel-id]",
		Content:  logsHelp,
	},
}

// commandCategories defines the order and grouping of categories
var commandCategories = []struct {
	Name        string
	Description string
}{
	{"Service Management", "Starting, stopping, and managing the shine service"},
	{"Monitoring", "Viewing status, logs, and debugging"},
	{"Configuration", "Managing configuration and settings"},
}

// getCommandsByCategory returns commands grouped by category
func getCommandsByCategory() map[string][]*CommandHelp {
	result := make(map[string][]*CommandHelp)

	for _, cmd := range helpRegistry {
		result[cmd.Category] = append(result[cmd.Category], cmd)
	}

	return result
}

// getAllCommands returns all commands sorted by name
func getAllCommands() []*CommandHelp {
	commands := make([]*CommandHelp, 0, len(helpRegistry))
	for _, cmd := range helpRegistry {
		commands = append(commands, cmd)
	}
	return commands
}

// getCommandHelp returns help metadata for a specific command
func getCommandHelp(name string) (*CommandHelp, bool) {
	cmd, ok := helpRegistry[name]
	return cmd, ok
}

// getCommandNames returns all command names (useful for completion)
func getCommandNames() []string {
	names := make([]string, 0, len(helpRegistry))
	for name := range helpRegistry {
		names = append(names, name)
	}
	return names
}
