package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiercepeda/tapas/internal/ports"
)

const (
	statusBar = "[k] Kill   [Enter] Details   [r] Refresh   [q] Quit"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	headerStyle  = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	modalStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)
)

// View renders the current state. Never executes OS commands.
func (m Model) View() string {
	if m.showKillConfirm && m.killTarget != nil {
		return m.viewKillConfirm()
	}
	if m.showDetails {
		return m.viewDetails()
	}
	return m.viewTable()
}

func (m Model) viewKillConfirm() string {
	p := m.killTarget
	body := fmt.Sprintf("Kill port %d (%s)?\n\n[y] Confirm   [n] Cancel", p.PortNum, p.Process)
	if m.killResult != "" {
		body += "\n\n" + errorStyle.Render(m.killResult)
	}
	content := modalStyle.Render(body)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) viewDetails() string {
	p := m.SelectedPort()
	if p == nil {
		return m.viewTable()
	}
	lines := []string{
		"Port:       " + fmt.Sprintf("%d", p.PortNum),
		"PID:        " + fmt.Sprintf("%d", p.PID),
		"Process:    " + p.Process,
		"Protocol:   " + p.Protocol,
		"Working dir: " + p.WorkingDir,
		"Command:   " + truncate(p.Command, 60),
	}
	if !p.StartTime.IsZero() {
		lines = append(lines, "Start time: "+p.StartTime.Format("2006-01-02 15:04:05"))
	}
	lines = append(lines, "", "[q] or [Esc] Close")
	content := modalStyle.Render(strings.Join(lines, "\n"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, content)
}

func (m Model) viewTable() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("TAPAS") + "\n\n")

	if m.err != "" {
		b.WriteString(errorStyle.Render(m.err) + "\n\n")
	}
	if m.successMsg != "" {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(m.successMsg) + "\n\n")
	}

	if len(m.ports) == 0 && m.err == "" {
		b.WriteString(dimStyle.Render("No listening ports found. Time to cook something.") + "\n")
		b.WriteString("\n" + statusStyle.Render(statusBar))
		return b.String()
	}

	// Table header
	header := headerStyle.Render(fmt.Sprintf("%-6s %-12s %-10s %-12s", "PORT", "PROCESS", "PROJECT", "UPTIME"))
	b.WriteString(header + "\n")

	// Rows
	for i, p := range m.ports {
		row := rowLine(&p)
		if i == m.selected {
			row = selectedStyle.Render(row)
		}
		b.WriteString(row + "\n")
	}

	b.WriteString("\n" + statusStyle.Render(statusBar))
	return b.String()
}

func rowLine(p *ports.Port) string {
	uptime := formatUptime(p.Uptime())
	project := truncate(p.Project(), 10)
	return fmt.Sprintf("%-6d %-12s %-10s %-12s", p.PortNum, truncate(p.Process, 12), project, uptime)
}

func formatUptime(d time.Duration) string {
	if d <= 0 {
		return "—"
	}
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 || len(s) <= maxLen {
		return s
	}
	cut := maxLen - 3
	if cut <= 0 {
		return s[:maxLen]
	}
	return s[:cut] + "..."
}