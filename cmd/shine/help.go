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
}

func showHelp(topic string) {
	output, err := registry.Render(topic, help.RenderOptions{Width: 100})
	if err != nil {
		Error(err.Error())
		os.Exit(1)
	}
	fmt.Print(output)
}
