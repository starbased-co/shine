package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	colorSuccess = lipgloss.Color("2")   // Green
	colorError   = lipgloss.Color("1")   // Red
	colorWarning = lipgloss.Color("3")   // Yellow
	colorInfo    = lipgloss.Color("4")   // Blue
	colorMuted   = lipgloss.Color("240") // Gray
)

var (
	styleSuccess = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	styleError   = lipgloss.NewStyle().Foreground(colorError).Bold(true)
	styleWarning = lipgloss.NewStyle().Foreground(colorWarning).Bold(true)
	styleInfo    = lipgloss.NewStyle().Foreground(colorInfo).Bold(true)
	styleMuted   = lipgloss.NewStyle().Foreground(colorMuted)
	styleBold    = lipgloss.NewStyle().Bold(true)
)

func Success(msg string) {
	fmt.Println(styleSuccess.Render("✓") + " " + msg)
}

func Error(msg string) {
	fmt.Println(styleError.Render("✗") + " " + msg)
}

func Warning(msg string) {
	fmt.Println(styleWarning.Render("⚠") + " " + msg)
}

func Info(msg string) {
	fmt.Println(styleInfo.Render("ℹ") + " " + msg)
}

func Muted(msg string) {
	fmt.Println(styleMuted.Render(msg))
}

func Header(title string) {
	fmt.Println()
	fmt.Println(styleBold.Render(title))
	fmt.Println(strings.Repeat("─", len(title)))
}

// Table represents a simple table for display
type Table struct {
	Headers []string
	Rows    [][]string
}

func NewTable(headers ...string) *Table {
	return &Table{
		Headers: headers,
		Rows:    [][]string{},
	}
}

func (t *Table) AddRow(values ...string) {
	t.Rows = append(t.Rows, values)
}

func (t *Table) Render() string {
	if len(t.Rows) == 0 {
		return styleMuted.Render("(no data)")
	}

	colWidths := make([]int, len(t.Headers))
	for i, header := range t.Headers {
		colWidths[i] = len(header)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if len(cell) > colWidths[i] {
				colWidths[i] = len(cell)
			}
		}
	}

	var sb strings.Builder

	for i, header := range t.Headers {
		sb.WriteString(styleBold.Render(padRight(header, colWidths[i])))
		if i < len(t.Headers)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	for i, width := range colWidths {
		sb.WriteString(strings.Repeat("─", width))
		if i < len(colWidths)-1 {
			sb.WriteString("  ")
		}
	}
	sb.WriteString("\n")

	for _, row := range t.Rows {
		for i, cell := range row {
			sb.WriteString(padRight(cell, colWidths[i]))
			if i < len(row)-1 {
				sb.WriteString("  ")
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

func (t *Table) Print() {
	fmt.Print(t.Render())
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + strings.Repeat(" ", length-len(s))
}

func StatusBox(foreground string, backgroundCount int, totalCount int) string {
	var parts []string

	if foreground != "" {
		parts = append(parts, styleSuccess.Render(fmt.Sprintf("Foreground: %s", foreground)))
	} else {
		parts = append(parts, styleMuted.Render("Foreground: (none)"))
	}

	if backgroundCount > 0 {
		parts = append(parts, styleInfo.Render(fmt.Sprintf("Background: %d", backgroundCount)))
	} else {
		parts = append(parts, styleMuted.Render("Background: 0"))
	}

	parts = append(parts, fmt.Sprintf("Total: %d", totalCount))

	return strings.Join(parts, " │ ")
}
