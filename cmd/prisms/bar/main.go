package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Set window title using ANSI escape sequence
	fmt.Print("\033]0;shine-bar\007")

	// Note: Don't use tea.WithAltScreen() for thin status bars
	// Alt screen mode is for full-screen TUIs, causes rendering issues in panels
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

// Hyprland workspace types
type workspace struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

type activeWorkspace struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type tickMsg time.Time

type model struct {
	workspaces      []workspace
	activeWorkspaceID int
	currentTime     time.Time
	width           int
	err             error
}

func initialModel() model {
	workspaces, activeID := getWorkspaces()
	return model{
		workspaces:      workspaces,
		activeWorkspaceID: activeID,
		currentTime:     time.Now(),
		width:           80,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tickCmd(),
		refreshWorkspacesCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

type workspacesMsg struct {
	workspaces []workspace
	activeID   int
}

func refreshWorkspacesCmd() tea.Cmd {
	return func() tea.Msg {
		workspaces, activeID := getWorkspaces()
		return workspacesMsg{
			workspaces: workspaces,
			activeID:   activeID,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc:
			return m, tea.Quit
		}

	case tickMsg:
		m.currentTime = time.Time(msg)
		return m, tea.Batch(
			tickCmd(),
			refreshWorkspacesCmd(),
		)

	case workspacesMsg:
		m.workspaces = msg.workspaces
		m.activeWorkspaceID = msg.activeID
		return m, nil
	}

	return m, nil
}

func (m model) View() string {
	// Styles with high contrast for visibility in thin panels
	// Use bright colors on dark/transparent background
	activeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")).  // Bright cyan
		Background(lipgloss.Color("0")).    // Black background for contrast
		Bold(true).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).  // White
		Background(lipgloss.Color("0")).    // Black background
		Padding(0, 1)

	clockStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("13")).  // Bright magenta
		Background(lipgloss.Color("0")).    // Black background
		Bold(true).
		Padding(0, 1)

	// Render workspaces
	var workspaceStrs []string
	for _, ws := range m.workspaces {
		wsLabel := fmt.Sprintf("%d", ws.ID)
		if ws.ID == m.activeWorkspaceID {
			workspaceStrs = append(workspaceStrs, activeStyle.Render(wsLabel))
		} else {
			workspaceStrs = append(workspaceStrs, inactiveStyle.Render(wsLabel))
		}
	}

	workspacesView := strings.Join(workspaceStrs, "")

	// Render clock
	clockView := clockStyle.Render(m.currentTime.Format("15:04:05"))

	// Calculate spacer width
	contentWidth := lipgloss.Width(workspacesView) + lipgloss.Width(clockView)
	spacerWidth := m.width - contentWidth
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := strings.Repeat(" ", spacerWidth)

	// Combine all parts
	return lipgloss.JoinHorizontal(
		lipgloss.Top,
		workspacesView,
		spacer,
		clockView,
	)
}

// getWorkspaces queries Hyprland for workspace information
func getWorkspaces() ([]workspace, int) {
	// Get all workspaces
	cmd := exec.Command("hyprctl", "workspaces", "-j")
	output, err := cmd.Output()
	if err != nil {
		return []workspace{{ID: 1, Name: "1"}}, 1
	}

	var workspaces []workspace
	if err := json.Unmarshal(output, &workspaces); err != nil {
		return []workspace{{ID: 1, Name: "1"}}, 1
	}

	// Get active workspace
	cmd = exec.Command("hyprctl", "activeworkspace", "-j")
	output, err = cmd.Output()
	if err != nil {
		if len(workspaces) > 0 {
			return workspaces, workspaces[0].ID
		}
		return []workspace{{ID: 1, Name: "1"}}, 1
	}

	var active activeWorkspace
	if err := json.Unmarshal(output, &active); err != nil {
		if len(workspaces) > 0 {
			return workspaces, workspaces[0].ID
		}
		return []workspace{{ID: 1, Name: "1"}}, 1
	}

	return workspaces, active.ID
}
