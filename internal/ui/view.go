package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiercepeda/tapas/internal/ports"
)

const (
	statusBar = "[k] Kill   [Enter] Details   [/] Search   [s] Sort   [r] Refresh   [q] Quit"

	// Column layout: keep Port, Protocol, Process, Uptime; truncate Project first (UX narrow-terminal rule).
	colPort     = 8
	colProtocol = 5   // TCP / UDP
	colProcess  = 12
	colUptime   = 12
	colGaps     = 4   // spaces between 5 columns
	minTableW   = colPort + colProtocol + colProcess + colUptime + colGaps // 41; project gets the rest
)

var (
	titleStyle    = lipgloss.NewStyle().Bold(true)
	headerStyle   = lipgloss.NewStyle().Bold(true)
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("7")).Foreground(lipgloss.Color("0"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	modalStyle    = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)
	// v0.2 color semantics (state, not decoration)
	longRunStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))   // soft red >24h
	devPortStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))   // blue 3000-3005
	dbPortStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))   // purple 5432, 6379
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))   // system / privileged
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

	disp := m.displayPorts()
	if len(disp) == 0 && m.err == "" {
		b.WriteString(dimStyle.Render("No listening ports found. Time to cook something.") + "\n")
		if m.searchQuery != "" {
			b.WriteString(dimStyle.Render("No matches for \"" + m.searchQuery + "\".") + "\n")
		}
		b.WriteString("\n" + statusStyle.Render(statusBar))
		return b.String()
	}

	// Narrow-terminal: project column gets remaining width; truncate gracefully (never break layout).
	tableWidth := m.width
	if tableWidth <= 0 {
		tableWidth = 80
	}
	projectCol := tableWidth - minTableW
	if projectCol < 1 {
		projectCol = 1
	}

	// Table header with sort indicator
	portHdr, protoHdr, processHdr, projectHdr, uptimeHdr := "PORT", "PROTO", "PROCESS", "PROJECT", "UPTIME"
	switch m.sortKey {
	case SortByPort:
		portHdr = "PORT \u2191"
	case SortByUptime:
		uptimeHdr = "UPTIME \u2191"
	case SortByProcess:
		processHdr = "PROCESS \u2191"
	}
	header := headerStyle.Render(fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s", colPort, truncate(portHdr, colPort), colProtocol, truncate(protoHdr, colProtocol), colProcess, truncate(processHdr, colProcess), projectCol, truncate(projectHdr, projectCol), colUptime, truncate(uptimeHdr, colUptime)))
	b.WriteString(header + "\n")

	// Rows (from display list, with color semantics)
	for i, p := range disp {
		row := rowLine(&p, projectCol)
		if i == m.selected {
			row = selectedStyle.Render(row)
		} else {
			row = rowStyle(&p).Render(row)
		}
		b.WriteString(row + "\n")
	}

	// Status bar and search prompt
	b.WriteString("\n")
	if m.searchMode {
		b.WriteString(statusStyle.Render("/ ") + statusStyle.Render(m.searchQuery) + dimStyle.Render("_") + "\n")
		b.WriteString(dimStyle.Render("Esc to clear search") + "\n")
	} else {
		b.WriteString(statusStyle.Render(statusBar))
	}
	return b.String()
}

// rowStyle returns the semantic style for a row (long-run red, dev blue, DB purple, system muted).
func rowStyle(p *ports.Port) lipgloss.Style {
	const longRunThreshold = 24 * time.Hour
	if p.Uptime() >= longRunThreshold {
		return longRunStyle
	}
	switch p.PortNum {
	case 3000, 3001, 3002, 3003, 3004, 3005:
		return devPortStyle
	case 5432, 6379:
		return dbPortStyle
	}
	if p.PortNum < 1024 {
		return mutedStyle
	}
	return lipgloss.NewStyle() // default
}

// rowLine formats one table row. projectCol is the width for the project column (narrow-terminal: truncate first).
func rowLine(p *ports.Port, projectCol int) string {
	uptime := formatUptime(p.Uptime())
	project := truncate(p.Project(), projectCol)
	proto := truncate(strings.ToUpper(p.Protocol), colProtocol)
	if proto == "" {
		proto = "—"
	}
	return fmt.Sprintf("%-*d %-*s %-*s %-*s %-*s", colPort, p.PortNum, colProtocol, proto, colProcess, truncate(p.Process, colProcess), projectCol, project, colUptime, uptime)
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