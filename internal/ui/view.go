package ui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiercepeda/tapas/internal/ports"
)

const (
	statusBar = "[k] Kill   [Enter] Details   [/] Search   [s] Sort   [r] Refresh   [w] Watch   [q] Quit"

	// Column layout: symbol + Port, Protocol, Process, App, Bind, Conn (connection count), Uptime; truncate Project first.
	colSymbol   = 1
	colPort     = 8
	colProtocol = 5
	colProcess  = 45
	colApp      = 10
	colBind     = 6
	colConn     = 5  // connection count or —
	colUptime   = 12
	colGaps     = 4
	minTableW   = colSymbol + colPort + colProtocol + colProcess + colApp + colBind + colConn + colUptime + colGaps
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	headerStyle  = lipgloss.NewStyle().Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	modalStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)
	// Selected row: dark background + bright text (ANSI 16-color so all terminals show highlight).
	selectedStyle = lipgloss.NewStyle().Background(lipgloss.Color("8")).Foreground(lipgloss.Color("15"))
	// v0.2 color semantics: ANSI 16-color codes so blue, purple, muted show in 16-color terminals.
	longRunStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))  // red >24h
	devPortStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))  // blue 3000-3005
	dbPortStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("5"))  // magenta/purple 5432, 6379, etc.
	dockerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan Docker containers
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))  // system / privileged (muted)
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
	body := fmt.Sprintf("Kill port %d (%s)?\n\n[y] Confirm   [n] Cancel", p.PortNum, processLabel(p))
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
	if p.Framework != "" {
		lines = append(lines, "Framework:  "+p.Framework)
	}
	if p.DockerContainerName != "" {
		line := "Container:  Docker → " + p.DockerContainerName
		if p.DockerImage != "" {
			line += " (" + p.DockerImage + ")"
		}
		lines = append(lines, line)
	} else if p.InDocker {
		lines = append(lines, "Container:  Docker")
	}
	if p.BindAddress != "" {
		lines = append(lines, "Bind:      "+bindLabel(p.BindAddress))
	}
	if p.ConnectionCount >= 0 {
		lines = append(lines, fmt.Sprintf("Connections: %d", p.ConnectionCount))
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
	title := "TAPAS"
	if m.WatchEnabled {
		title += "  (watch " + m.WatchInterval.String() + ")"
	}
	b.WriteString(titleStyle.Render(title) + "\n\n")

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
	portHdr, protoHdr, processHdr, appHdr, bindHdr, connHdr, projectHdr, uptimeHdr := "PORT", "PROTO", "PROCESS", "APP", "BIND", "CONN", "PROJECT", "UPTIME"
	switch m.sortKey {
	case SortByPort:
		portHdr = "PORT \u2191"
	case SortByUptime:
		uptimeHdr = "UPTIME \u2191"
	case SortByProcess:
		processHdr = "PROCESS \u2191"
	}
	header := headerStyle.Render(fmt.Sprintf("%-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s", colSymbol, " ", colPort, truncate(portHdr, colPort), colProtocol, truncate(protoHdr, colProtocol), colProcess, truncate(processHdr, colProcess), colApp, truncate(appHdr, colApp), colBind, truncate(bindHdr, colBind), colConn, truncate(connHdr, colConn), projectCol, truncate(projectHdr, projectCol), colUptime, truncate(uptimeHdr, colUptime)))
	b.WriteString(header + "\n")

	// Rows (from display list, with color semantics + symbol cues)
	for i, p := range disp {
		kind := rowKindFor(&p)
		row := rowLine(&p, projectCol, rowSymbol(kind))
		if i == m.selected {
			row = selectedStyle.Render(row)
		} else {
			row = rowStyleForKind(kind).Render(row)
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

// rowKind is the semantic category of a port row (for style + symbol; never color alone).
type rowKind int

const (
	kindDefault rowKind = iota
	kindLongRun
	kindDev
	kindDB
	kindDocker
	kindSystem
)

func rowKindFor(p *ports.Port) rowKind {
	const longRunThreshold = 24 * time.Hour
	if p.DockerContainerName != "" || p.InDocker {
		return kindDocker
	}
	if p.Uptime() >= longRunThreshold {
		return kindLongRun
	}
	switch p.PortNum {
	case 3000, 3001, 3002, 3003, 3004, 3005:
		return kindDev
	case 5432, 6379, 27017, 3306, 1433, 5984, 9200:
		return kindDB
	}
	if p.PortNum < 1024 {
		return kindSystem
	}
	return kindDefault
}

// rowSymbol returns a single-character cue for the row kind (accessibility: not color alone).
func rowSymbol(k rowKind) string {
	switch k {
	case kindLongRun:
		return "!"
	case kindDev:
		return "D"
	case kindDB:
		return "B"
	case kindDocker:
		return "C" // container
	case kindSystem:
		return "\u00b7" // middle dot
	default:
		return "-"
	}
}

func rowStyleForKind(k rowKind) lipgloss.Style {
	switch k {
	case kindLongRun:
		return longRunStyle
	case kindDev:
		return devPortStyle
	case kindDB:
		return dbPortStyle
	case kindDocker:
		return dockerStyle
	case kindSystem:
		return mutedStyle
	default:
		return lipgloss.NewStyle()
	}
}

// projectLabel returns the PROJECT column text: "dili (Next.js)" or for Docker "pulso-api (Docker)".
func projectLabel(p *ports.Port) string {
	base := ""
	if p.DockerContainerName != "" {
		// Docker: use container name as project context (host cwd is meaningless)
		base = p.DockerContainerName
	} else {
		base = p.ProjectDisplayName
		if base == "" {
			base = p.Project()
		}
	}
	if base == "" || base == "/" {
		base = "—"
	}
	if base == "—" {
		return base
	}
	if p.DockerContainerName != "" {
		return base + " (Docker)"
	}
	if p.Framework != "" {
		return base + " (" + p.Framework + ")"
	}
	return base
}

// connLabel returns the connection count for the CONN column ("0", "4", etc.).
func connLabel(count int) string {
	if count <= 0 {
		return "0"
	}
	return fmt.Sprintf("%d", count)
}

// bindLabel returns LOCAL, PUBLIC, or the bind address for security awareness.
func bindLabel(addr string) string {
	switch addr {
	case "127.0.0.1", "::1":
		return "LOCAL"
	case "0.0.0.0", "*", "":
		return "PUBLIC"
	default:
		return addr
	}
}

// processLabel returns the process column text: Docker, "PostgreSQL (local)", "Framework (process)", or process name.
func processLabel(p *ports.Port) string {
	if p.DockerContainerName != "" {
		s := "Docker → " + p.DockerContainerName
		if p.DockerImage != "" {
			s += " (" + p.DockerImage + ")"
		}
		return s
	}
	if db := ports.DatabaseProductName(p.PortNum); db != "" {
		return db + " (" + bindLabel(p.BindAddress) + ")"
	}
	if p.Framework != "" {
		if p.Process != "" && p.Process != "—" {
			return p.Framework + " (" + p.Process + ")"
		}
		return p.Framework
	}
	if p.Process == "" {
		return "—"
	}
	return p.Process
}

// appBadge returns the framework badge + Docker indicator for the APP column.
func appBadge(p *ports.Port) string {
	var badge string
	if p.DockerContainerName != "" {
		badge = "Docker"
	} else if p.Framework != "" {
		badge = p.Framework
		if p.InDocker {
			badge += " D"
		}
	} else {
		badge = "—"
		if p.InDocker {
			badge += " D"
		}
	}
	return truncate(badge, colApp)
}

// rowLine formats one table row. symbol is the row-kind cue (1 char). projectCol is the width for the project column.
func rowLine(p *ports.Port, projectCol int, symbol string) string {
	uptime := formatUptime(p.Uptime())
	project := truncate(projectLabel(p), projectCol)
	proto := truncate(strings.ToUpper(p.Protocol), colProtocol)
	if proto == "" {
		proto = "—"
	}
	// First rune only (symbol is 1 char; avoid cutting multi-byte runes).
	sym := symbol
	if utf8.RuneCountInString(sym) > 1 {
		sym = string([]rune(symbol)[0])
	}
	return fmt.Sprintf("%-*s %-*d %-*s %-*s %-*s %-*s %-*s %-*s %-*s", colSymbol, sym, colPort, p.PortNum, colProtocol, proto, colProcess, truncate(processLabel(p), colProcess), colApp, appBadge(p), colBind, truncate(bindLabel(p.BindAddress), colBind), colConn, connLabel(p.ConnectionCount), projectCol, project, colUptime, uptime)
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