package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/javiercepeda/tapas/internal/ports"
)

const (
	statusBar = "[k] Kill   [Enter] Details   [/] Search   [s] Sort   [r] Refresh   [w] Watch   [q] Quit"
	// Legend under footer: what keys do and what table indicators mean.
	// Column layout: symbol + Port, Protocol, Process, App, Bind, Conn, Env, Uptime; truncate Project first.
	colSymbol   = 2 // two cells so ●/○ render reliably and don't get clipped
	colPort     = 8
	colProtocol = 5
	colProcess  = 45
	colApp      = 10
	colBind     = 6
	colConn     = 5
	colEnv      = 7  // npm, yarn, pnpm, poetry, pipenv, cargo, go
	colUptime   = 12
	colGaps     = 4
	minTableW   = colSymbol + colPort + colProtocol + colProcess + colApp + colBind + colConn + colEnv + colUptime + colGaps
)

// TAPAS color system: calm, professional, ~90% neutral. See docs.
var (
	colorAccent  = lipgloss.Color("#4C8DFF") // selection, focus, active sort
	colorMuted   = lipgloss.Color("#6C757D") // footer, secondary, system
	colorWarning = lipgloss.Color("#E57373") // public exposure, errors (minimal)
	colorSuccess = lipgloss.Color("#4CAF50") // temporary confirmation only
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	headerStyle  = lipgloss.NewStyle().Bold(true)
	errorStyle   = lipgloss.NewStyle().Foreground(colorWarning)
	dimStyle     = lipgloss.NewStyle().Foreground(colorMuted)
	statusStyle  = lipgloss.NewStyle().Foreground(colorMuted)
	modalStyle   = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(1, 2)
	// Selected: subtle blue tint, slight bold; do not override text color (inherit terminal foreground).
	selectedStyle = lipgloss.NewStyle().Background(colorAccent).Bold(true)
	// Success: momentary feedback only (kill success, etc.).
	successStyle = lipgloss.NewStyle().Foreground(colorSuccess)
	// Public port: soft red dot only (never entire row). Indicator system: ● or ! (ASCII).
	publicDotStyle = lipgloss.NewStyle().Foreground(colorWarning)
	// Docker: muted circle; secondary to public. Indicator system: ○ or - (ASCII).
	dockerDotStyle = lipgloss.NewStyle().Foreground(colorMuted)
	// System: muted dot (port < 1024). Indicator system: ● or · (ASCII).
	systemDotStyle = lipgloss.NewStyle().Foreground(colorMuted)
	// Row semantics: long-run and system use default text + symbol; only system gets muted.
	longRunStyle = lipgloss.NewStyle() // >24h: symbol "!" only, no color
	mutedStyle   = lipgloss.NewStyle().Foreground(colorMuted) // system ports <1024
	// Active mode (search, sort indicator): accent blue.
	accentStyle = lipgloss.NewStyle().Foreground(colorAccent)
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
	if p.Environment != "" {
		lines = append(lines, "Environment: "+p.Environment)
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
		b.WriteString(successStyle.Render(m.successMsg) + "\n\n")
	}

	disp := m.displayPorts()
	if len(disp) == 0 && m.err == "" {
		b.WriteString(dimStyle.Render("No listening ports found. Time to cook something.") + "\n")
		if m.searchQuery != "" {
			b.WriteString(dimStyle.Render("No matches for \"" + m.searchQuery + "\".") + "\n")
		}
		b.WriteString("\n" + statusStyle.Render(statusBar) + "\n" + m.statusLegend())
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

	// Table header: bold, active sort column in accent blue
	portHdr, protoHdr, processHdr, appHdr, bindHdr, connHdr, envHdr, projectHdr, uptimeHdr := "PORT", "PROTO", "PROCESS", "APP", "BIND", "CONN", "ENV", "PROJECT", "UPTIME"
	switch m.sortKey {
	case SortByPort:
		portHdr = "PORT \u2191"
	case SortByUptime:
		uptimeHdr = "UPTIME \u2193" // descending: longest first
	case SortByProcess:
		processHdr = "PROCESS \u2191"
	}
	headerParts := []string{
		headerStyle.Render(fmt.Sprintf("%-*s", colSymbol, "")),
		headerStyle.Render(fmt.Sprintf("%-*s", colPort, truncate(portHdr, colPort))),
		headerStyle.Render(fmt.Sprintf("%-*s", colProtocol, truncate(protoHdr, colProtocol))),
		headerStyle.Render(fmt.Sprintf("%-*s", colProcess, truncate(processHdr, colProcess))),
		headerStyle.Render(fmt.Sprintf("%-*s", colApp, truncate(appHdr, colApp))),
		headerStyle.Render(fmt.Sprintf("%-*s", colBind, truncate(bindHdr, colBind))),
		headerStyle.Render(fmt.Sprintf("%-*s", colConn, truncate(connHdr, colConn))),
		headerStyle.Render(fmt.Sprintf("%-*s", colEnv, truncate(envHdr, colEnv))),
		headerStyle.Render(fmt.Sprintf("%-*s", projectCol, truncate(projectHdr, projectCol))),
		headerStyle.Render(fmt.Sprintf("%-*s", colUptime, truncate(uptimeHdr, colUptime))),
	}
	// Apply accent to active sort column only
	switch m.sortKey {
	case SortByPort:
		headerParts[1] = accentStyle.Bold(true).Render(fmt.Sprintf("%-*s", colPort, truncate(portHdr, colPort)))
	case SortByUptime:
		headerParts[9] = accentStyle.Bold(true).Render(fmt.Sprintf("%-*s", colUptime, truncate(uptimeHdr, colUptime)))
	case SortByProcess:
		headerParts[3] = accentStyle.Bold(true).Render(fmt.Sprintf("%-*s", colProcess, truncate(processHdr, colProcess)))
	}
	header := strings.Join(headerParts, " ") + "\n"
	b.WriteString(header)

	// Rows: indicator system — first column Docker ○/- or System ●/· (muted), right column Public ●/! (warning only).
	// Apply row style (selection/kind) only to the middle so indicator colors are not overridden.
	for i, p := range disp {
		kind := rowKindFor(&p)
		firstCol, _, isSystem := firstColumnIndicator(&p, m.AsciiMode)
		var firstPart string
		if firstCol == " " {
			firstPart = "  " // fixed width so column aligns
		} else {
			style := dockerDotStyle
			if isSystem {
				style = systemDotStyle
			}
			// Pad to colSymbol width so the indicator column is stable and ●/○ don't get clipped
			firstPart = style.Render(firstCol) + " "
		}
		middlePart := rowLineMiddle(&p, projectCol)
		var publicPart string
		if pub := publicIndicator(&p, m.AsciiMode); pub != "" {
			publicPart = " " + publicDotStyle.Render(pub)
		} else {
			publicPart = " "
		}
		rowStyle := rowStyleForKind(kind)
		if i == m.selected {
			rowStyle = selectedStyle
		}
		b.WriteString(firstPart + rowStyle.Render(middlePart) + publicPart + "\n")
	}

	// Footer: muted gray; only active mode (e.g. search) in accent blue
	b.WriteString("\n")
	if m.searchMode {
		b.WriteString(accentStyle.Render("/ ") + statusStyle.Render(m.searchQuery) + dimStyle.Render("_") + "\n")
		b.WriteString(dimStyle.Render("Esc to clear search") + "\n")
	} else {
		b.WriteString(statusStyle.Render(statusBar) + "\n" + m.statusLegend())
	}
	return b.String()
}

// statusLegend returns the footer legend (what table indicators mean). Muted style; symbols use indicator colors.
func (m Model) statusLegend() string {
	pubSym, dockSym, sysSym := indicatorPublicUnicode, indicatorDockerUnicode, indicatorSystemUnicode
	if m.AsciiMode {
		pubSym, dockSym, sysSym = indicatorPublicASCII, indicatorDockerASCII, indicatorSystemASCII
	}
	return publicDotStyle.Render(pubSym) + dimStyle.Render(" public port   ") +
		dockerDotStyle.Render(dockSym) + dimStyle.Render(" Docker   ") +
		systemDotStyle.Render(sysSym) + dimStyle.Render(" system")
}

// Indicator system: Public ●/!, Docker ○/-, System ●/· (muted), Local empty. Shape-first, color-second.

const (
	indicatorPublicUnicode = "\u25cf" // ●
	indicatorPublicASCII   = "!"
	indicatorDockerUnicode = "\u25cb" // ○
	indicatorDockerASCII   = "-"
	indicatorSystemUnicode = "\u25cf" // ● (muted, same shape as public)
	indicatorSystemASCII   = "\u00b7" // · middle dot
)

// firstColumnIndicator returns the first-column char and which style to use (Docker ○, System ●, or space).
// System (port < 1024) takes precedence so system ports are always visible; Docker is shown when in container and port >= 1024.
func firstColumnIndicator(p *ports.Port, ascii bool) (char string, isDocker, isSystem bool) {
	if p.PortNum < 1024 {
		if ascii {
			return indicatorSystemASCII, false, true
		}
		return indicatorSystemUnicode, false, true
	}
	if p.DockerContainerName != "" || p.InDocker {
		if ascii {
			return indicatorDockerASCII, true, false
		}
		return indicatorDockerUnicode, true, false
	}
	return " ", false, false
}

// publicIndicator returns the right-column indicator for public bind (● or ! in ASCII); empty for local.
func publicIndicator(p *ports.Port, ascii bool) string {
	if !isPublicBind(p.BindAddress) {
		return ""
	}
	if ascii {
		return indicatorPublicASCII
	}
	return indicatorPublicUnicode
}

// rowKind is the semantic category of a port row (for style only; indicators are separate).
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

func rowStyleForKind(k rowKind) lipgloss.Style {
	switch k {
	case kindLongRun:
		return longRunStyle
	case kindSystem:
		return mutedStyle
	default:
		return lipgloss.NewStyle() // dev, db, docker, default: terminal default color
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

// envLabel returns the ENV column text (npm, yarn, pnpm, etc.) or "—".
func envLabel(env string) string {
	if env == "" {
		return "—"
	}
	return env
}

// isPublicBind reports whether the bind address exposes the port publicly (0.0.0.0, *, or empty).
func isPublicBind(addr string) bool {
	switch addr {
	case "0.0.0.0", "*", "":
		return true
	default:
		return false
	}
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

// rowLineMiddle returns the row content without the first column (port through uptime). Used so we can apply row style only to this part and keep indicator colors intact.
func rowLineMiddle(p *ports.Port, projectCol int) string {
	uptime := formatUptime(p.Uptime())
	project := truncate(projectLabel(p), projectCol)
	proto := truncate(strings.ToUpper(p.Protocol), colProtocol)
	if proto == "" {
		proto = "—"
	}
	// Leading space aligns with the gap between symbol column and port in the header.
	return " " + fmt.Sprintf("%-*d %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s %-*s", colPort, p.PortNum, colProtocol, proto, colProcess, truncate(processLabel(p), colProcess), colApp, appBadge(p), colBind, truncate(bindLabel(p.BindAddress), colBind), colConn, connLabel(p.ConnectionCount), colEnv, truncate(envLabel(p.Environment), colEnv), projectCol, project, colUptime, uptime)
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