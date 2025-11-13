package help

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

// RenderOptions configures help rendering
type RenderOptions struct {
	Width int
	Style string
}

// Render renders a help topic using Glamour
func (r *Registry) Render(topicName string, opts RenderOptions) (string, error) {
	// If no topic specified, show topic list
	if topicName == "" {
		return r.renderTopicList(opts), nil
	}

	// Special topics
	switch topicName {
	case "list":
		return r.renderTopicList(opts), nil
	case "categories":
		return r.renderCategories(opts), nil
	}

	// Get specific topic
	topic, ok := r.Get(topicName)
	if !ok {
		return "", fmt.Errorf("topic %q not found", topicName)
	}

	// Render with Glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(opts.Width),
	)
	if err != nil {
		return "", fmt.Errorf("failed to create renderer: %w", err)
	}

	rendered, err := renderer.Render(topic.Content)
	if err != nil {
		return "", fmt.Errorf("failed to render content: %w", err)
	}

	return rendered, nil
}

// renderTopicList renders a list of all topics
func (r *Registry) renderTopicList(opts RenderOptions) string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	categoryStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))

	b.WriteString(titleStyle.Render("Available Help Topics"))
	b.WriteString("\n\n")

	byCategory := r.ListByCategory()
	categories := r.Categories()

	for _, cat := range categories {
		b.WriteString(categoryStyle.Render(cat))
		b.WriteString("\n")

		for _, topic := range byCategory[cat] {
			b.WriteString(fmt.Sprintf("  %-15s  %s\n", topic.Name, topic.Synopsis))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderCategories renders category list
func (r *Registry) renderCategories(opts RenderOptions) string {
	var b strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))

	b.WriteString(titleStyle.Render("Help Categories"))
	b.WriteString("\n\n")

	byCategory := r.ListByCategory()
	categories := r.Categories()

	for _, cat := range categories {
		topics := byCategory[cat]
		b.WriteString(fmt.Sprintf("  %-20s  (%d topics)\n", cat, len(topics)))
	}

	return b.String()
}

// JSON renders topic metadata as JSON
func (r *Registry) JSON(topicName string) ([]byte, error) {
	if topicName == "" || topicName == "list" {
		// List all topics
		return json.MarshalIndent(r.List(), "", "  ")
	}

	if topicName == "categories" {
		// List categories
		return json.MarshalIndent(r.Categories(), "", "  ")
	}

	if topicName == "names" {
		// List topic names
		return json.MarshalIndent(r.Names(), "", "  ")
	}

	// Get specific topic
	topic, ok := r.Get(topicName)
	if !ok {
		return nil, fmt.Errorf("topic %q not found", topicName)
	}

	return json.MarshalIndent(topic, "", "  ")
}
