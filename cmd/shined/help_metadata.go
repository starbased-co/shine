package main

import (
	_ "embed"
)

//go:embed help/usage.md
var usageHelp string

//go:embed help/signals.md
var signalsHelp string

//go:embed help/config.md
var configHelp string

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

var helpRegistry = map[string]*CommandHelp{
	"usage": {
		Name:     "usage",
		Category: "General",
		Synopsis: "General usage and command-line options",
		Usage:    "shined [options]",
		Content:  usageHelp,
		Related:  []string{"signals", "config"},
		SeeAlso:  []string{"shine start", "shine stop"},
	},
	"signals": {
		Name:     "signals",
		Category: "Operations",
		Synopsis: "Signal handling for reload and shutdown",
		Usage:    "pkill -HUP shined",
		Content:  signalsHelp,
		Related:  []string{"config"},
		SeeAlso:  []string{"shine reload", "shine stop"},
	},
	"config": {
		Name:     "config",
		Category: "Configuration",
		Synopsis: "Configuration file format and loading",
		Usage:    "shined -config PATH",
		Content:  configHelp,
		Related:  []string{"signals"},
		SeeAlso:  []string{"shine.toml reference"},
	},
}
