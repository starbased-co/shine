package main

import (
	"fmt"
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Set window title using ANSI escape sequence
	fmt.Print("\033]0;shine-clock\007")

	// Use alt screen mode to take over the full terminal
	// This prevents prismctl logs from interfering with the display
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type tickMsg time.Time

type model struct {
	currentTime time.Time
	width       int
	height      int
}

func initialModel() model {
	return model{
		currentTime: time.Now(),
		width:       20,
		height:      5,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, tea.Quit
		}

	case tickMsg:
		m.currentTime = time.Time(msg)
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	// Large clock display
	timeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("13")). // Bright magenta
		Background(lipgloss.Color("0")).  // Black background
		Bold(true).
		Align(lipgloss.Center, lipgloss.Center).
		Width(m.width).
		Height(m.height)

	// Format time with larger text feel
	timeStr := m.currentTime.Format("15:04:05")

	return timeStyle.Render(timeStr)
}
