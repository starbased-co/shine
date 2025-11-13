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
		Synopsis: "General usage and command-line options",
		Usage:    "shinectl [options]",
		Content:  usageHelp,
		Related:  []string{"signals", "config"},
		SeeAlso:  []string{"shine start", "shine stop"},
	},
	"signals": {
		Name:     "signals",
		Category: "Operations",
		Synopsis: "Signal handling for reload and shutdown",
		Usage:    "pkill -HUP shinectl",
		Content:  signalsHelp,
		Related:  []string{"config"},
		SeeAlso:  []string{"shine reload", "shine stop"},
	},
	"config": {
		Name:     "config",
		Category: "Configuration",
		Synopsis: "Configuration file format and loading",
		Usage:    "shinectl -config PATH",
		Content:  configHelp,
		Related:  []string{"signals"},
		SeeAlso:  []string{"prism.toml reference"},
	},
}

// topicCategories defines the order and grouping of categories
var topicCategories = []struct {
	Name        string
	Description string
}{
	{"General", "Basic usage and command-line interface"},
	{"Operations", "Runtime operations and signal handling"},
	{"Configuration", "Configuration file management"},
}

// getTopicsByCategory returns topics grouped by category
func getTopicsByCategory() map[string][]*CommandHelp {
	result := make(map[string][]*CommandHelp)

	for _, topic := range helpRegistry {
		result[topic.Category] = append(result[topic.Category], topic)
	}

	return result
}

// getAllTopics returns all topics sorted by name
func getAllTopics() []*CommandHelp {
	topics := make([]*CommandHelp, 0, len(helpRegistry))
	for _, topic := range helpRegistry {
		topics = append(topics, topic)
	}
	return topics
}

// getTopicHelp returns help metadata for a specific topic
func getTopicHelp(name string) (*CommandHelp, bool) {
	topic, ok := helpRegistry[name]
	return topic, ok
}

// getTopicNames returns all topic names (useful for completion)
func getTopicNames() []string {
	names := make([]string, 0, len(helpRegistry))
	for name := range helpRegistry {
		names = append(names, name)
	}
	return names
}
