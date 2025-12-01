package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/starbased-co/shine/pkg/help"
)

//go:embed help/*.md
var helpFiles embed.FS

var registry = help.NewRegistry()

func init() {
	usageContent, err := helpFiles.ReadFile("help/usage.md")
	if err == nil {
		registry.Register(&help.Topic{
			Name:     "",
			Category: "General",
			Synopsis: "General usage information",
			Content:  string(usageContent),
		})
	}

	for _, meta := range helpRegistry {
		registry.Register(&help.Topic{
			Name:     meta.Name,
			Category: meta.Category,
			Synopsis: meta.Synopsis,
			Usage:    meta.Usage,
			Related:  meta.Related,
			Content:  meta.Content,
		})
	}
}

func showHelp(topic string) {
	output, err := registry.Render(topic, help.RenderOptions{Width: 100})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	fmt.Print(output)
}
