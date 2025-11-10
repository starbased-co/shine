package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func main() {
	// Set window title using ANSI escape sequence
	fmt.Print("\033]0;shine-sysinfo\007")

	// Use alt screen mode to take over the full terminal
	p := tea.NewProgram(
		initialModel(),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

type tickMsg time.Time

type sysInfo struct {
	hostname   string
	uptime     string
	cpuPercent string
	memPercent string
}

type model struct {
	info   sysInfo
	width  int
	height int
}

func initialModel() model {
	return model{
		info:   getSysInfo(),
		width:  30,
		height: 10,
	}
}

func (m model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
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
		m.info = getSysInfo()
		return m, tickCmd()
	}

	return m, nil
}

func (m model) View() string {
	// Styles
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14")). // Cyan
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")) // White

	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("14")).
		Padding(1, 2).
		Width(m.width)

	// Build info display
	lines := []string{
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Host: "),
			valueStyle.Render(m.info.hostname),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Uptime: "),
			valueStyle.Render(m.info.uptime),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("CPU: "),
			valueStyle.Render(m.info.cpuPercent),
		),
		lipgloss.JoinHorizontal(lipgloss.Left,
			labelStyle.Render("Memory: "),
			valueStyle.Render(m.info.memPercent),
		),
	}

	content := strings.Join(lines, "\n")

	return containerStyle.Render(content)
}

// getSysInfo retrieves basic system information
func getSysInfo() sysInfo {
	info := sysInfo{
		hostname:   getHostname(),
		uptime:     getUptime(),
		cpuPercent: "N/A",
		memPercent: getMemoryUsage(),
	}
	return info
}

func getHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return hostname
}

func getUptime() string {
	cmd := exec.Command("uptime", "-p")
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}
	return strings.TrimSpace(string(output))
}

func getMemoryUsage() string {
	cmd := exec.Command("free", "-h")
	output, err := cmd.Output()
	if err != nil {
		return "N/A"
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return "N/A"
	}

	// Parse memory line
	fields := strings.Fields(lines[1])
	if len(fields) < 3 {
		return "N/A"
	}

	// Calculate percentage if available
	totalStr := strings.TrimSuffix(fields[1], "Gi")
	usedStr := strings.TrimSuffix(fields[2], "Gi")

	total, err1 := strconv.ParseFloat(totalStr, 64)
	used, err2 := strconv.ParseFloat(usedStr, 64)

	if err1 == nil && err2 == nil && total > 0 {
		percent := (used / total) * 100
		return fmt.Sprintf("%.1f%% (%s / %s)", percent, fields[2], fields[1])
	}

	return fmt.Sprintf("%s / %s", fields[2], fields[1])
}
