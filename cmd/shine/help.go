package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/charmbracelet/glamour"
)

//go:embed help/usage.md
var usageHelp string

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

var renderer *glamour.TermRenderer

func init() {
	var err error
	renderer, err = glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(100),
	)
	if err != nil {
		// Fallback: renderer stays nil, will use plain text
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize help renderer: %v\n", err)
	}
}

func renderHelp(markdown string) string {
	if renderer == nil {
		return markdown
	}

	out, err := renderer.Render(markdown)
	if err != nil {
		return markdown
	}
	return out
}

// showHelp displays help for a specific topic or general usage
func showHelp(topic string) {
	var content string

	// Check for special topics
	switch topic {
	case "topics", "commands":
		content = helpTopics()
	case "list":
		content = helpList()
	case "categories":
		content = helpCategories()
	case "":
		content = usageHelp
	default:
		// Look up command in registry
		if cmd, ok := helpRegistry[topic]; ok {
			content = cmd.Content
		} else {
			// Unknown topic, show main help
			content = usageHelp
		}
	}

	fmt.Print(renderHelp(content))
}

// helpTopics returns a list of available help topics
func helpTopics() string {
	return `# shine - Available Help Topics

Use **shine help <topic>** to view detailed help for a specific command.

## Available Commands

- **start** - Start the shine service and enabled panels
- **stop** - Stop all running panels gracefully
- **status** - Display current state of panels and prisms
- **reload** - Reload configuration without restarting
- **logs** - View service and panel log files

## Examples

View help for the start command:
` + "```bash" + `
shine help start
` + "```" + `

View help for status command:
` + "```bash" + `
shine help status
` + "```" + `

List all help topics:
` + "```bash" + `
shine help topics
` + "```" + `

## Getting Started

For general usage information:
` + "```bash" + `
shine --help
` + "```" + `

## Configuration

Configuration file: ` + "`~/.config/shine/prism.toml`" + `
Log directory: ` + "`~/.local/share/shine/logs/`" + `

## Documentation

For complete documentation, see: https://github.com/starbased-co/shine
`
}

// helpList generates a compact list of all commands with brief descriptions
func helpList() string {
	content := "# shine - Command Reference\n\n"
	content += "All available commands with brief descriptions.\n\n"
	content += "## Commands\n\n"

	// Get commands by category
	byCategory := getCommandsByCategory()

	// Display by category in defined order
	for _, cat := range commandCategories {
		commands := byCategory[cat.Name]
		if len(commands) == 0 {
			continue
		}

		content += fmt.Sprintf("### %s\n\n", cat.Name)
		content += fmt.Sprintf("*%s*\n\n", cat.Description)

		for _, cmd := range commands {
			content += fmt.Sprintf("**`%s`** - %s\n", cmd.Usage, cmd.Synopsis)
			content += "\n"
		}
	}

	content += "## Getting More Help\n\n"
	content += "Use `shine help <command>` to view detailed help for a specific command.\n"

	return content
}

// helpCategories shows commands organized by category with descriptions
func helpCategories() string {
	content := "# shine - Command Categories\n\n"
	content += "Commands organized by functional category.\n\n"

	// Get commands by category
	byCategory := getCommandsByCategory()

	// Display by category in defined order
	for _, cat := range commandCategories {
		commands := byCategory[cat.Name]
		if len(commands) == 0 {
			continue
		}

		content += fmt.Sprintf("## %s\n\n", cat.Name)
		content += fmt.Sprintf("%s\n\n", cat.Description)

		for _, cmd := range commands {
			content += fmt.Sprintf("- **%s** - %s\n", cmd.Name, cmd.Synopsis)
		}
		content += "\n"
	}

	content += "## Usage\n\n"
	content += "```bash\n"
	content += "# View all commands\n"
	content += "shine help list\n\n"
	content += "# View detailed help for a command\n"
	content += "shine help <command>\n"
	content += "```\n"

	return content
}

// helpJSON outputs help metadata in JSON format (for machine consumption)
func helpJSON(topic string) error {
	var output interface{}

	if topic == "" {
		// Output all commands metadata
		output = getAllCommands()
	} else if topic == "categories" {
		// Output categories structure
		output = commandCategories
	} else if topic == "names" {
		// Output just command names (for completion)
		output = getCommandNames()
	} else {
		// Output specific command metadata
		if cmd, ok := getCommandHelp(topic); ok {
			// Omit the full markdown content for JSON output
			type CommandMeta struct {
				Name     string `json:"name"`
				Category string `json:"category"`
				Synopsis string `json:"synopsis"`
				Usage    string `json:"usage"`
			}
			output = CommandMeta{
				Name:     cmd.Name,
				Category: cmd.Category,
				Synopsis: cmd.Synopsis,
				Usage:    cmd.Usage,
			}
		} else {
			return fmt.Errorf("unknown command: %s", topic)
		}
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
