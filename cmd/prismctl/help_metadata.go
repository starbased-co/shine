package main

import (
	_ "embed"
)

//go:embed help/usage.md
var usageHelp string

//go:embed help/ipc.md
var ipcHelp string

//go:embed help/signals.md
var signalsHelp string

// CommandHelp defines structured metadata for help topics
type CommandHelp struct {
	Name        string   // Topic name
	Category    string   // Topic category for grouping
	Synopsis    string   // Brief one-line description
	Description string   // Longer description (optional)
	Usage       string   // Usage syntax
	Content     string   // Full help content (markdown)
	Related     []string // Related topics
	SeeAlso     []string // Additional topics/resources
}

// helpRegistry contains all help topic metadata
var helpRegistry = map[string]*CommandHelp{
	"usage": {
		Name:     "usage",
		Category: "General",
		Synopsis: "General usage and command-line interface",
		Usage:    "prismctl <prism-name> [component-name]",
		Content:  usageHelp,
		Related:  []string{"ipc", "signals"},
		SeeAlso:  []string{"Terminal state management"},
	},
	"ipc": {
		Name:     "ipc",
		Category: "Operations",
		Synopsis: "IPC protocol and command reference",
		Usage:    "echo '{\"jsonrpc\":\"2.0\",\"method\":\"prism/list\",\"params\":{},\"id\":1}' | socat - UNIX-CONNECT:<socket>",
		Content:  ipcHelp,
		Related:  []string{"signals"},
		SeeAlso:  []string{"Hot-swap capability", "MRU ordering"},
	},
	"signals": {
		Name:     "signals",
		Category: "Operations",
		Synopsis: "Signal handling and process management",
		Usage:    "kill -TERM $(pgrep prismctl)",
		Content:  signalsHelp,
		Related:  []string{"ipc"},
		SeeAlso:  []string{"SIGSTOP/SIGCONT", "Terminal state"},
	},
}
