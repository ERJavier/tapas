package ui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/javiercepeda/tapas/internal/ports"
)

// Lister is satisfied by ports.Lister. UI never runs OS commands; it uses this.
type Lister interface {
	List() ([]ports.Port, error)
}

// refreshDoneMsg is sent when port list refresh completes.
type refreshDoneMsg struct {
	ports []ports.Port
	err   error
}

// killDoneMsg is sent after a kill attempt (from the same program, so no async needed; we can show result in Update).
type killDoneMsg struct {
	ok    bool
	error string
}

// tickMsg is sent when watch-mode tick fires; triggers one refresh (efficient: one tick at a time).
type tickMsg struct{}

// SortKey is the current table sort key.
type SortKey int

const (
	SortByPort   SortKey = iota
	SortByUptime
	SortByProcess
)

func (k SortKey) String() string {
	switch k {
	case SortByPort:
		return "Port"
	case SortByUptime:
		return "Uptime"
	case SortByProcess:
		return "Process"
	default:
		return "Port"
	}
}

// Model is the root Bubble Tea model.
type Model struct {
	ports     []ports.Port
	selected  int
	lister    Lister
	err       string
	width     int
	height    int

	// v0.2: sort and filter
	sortKey     SortKey
	searchMode  bool
	searchQuery string

	// Modals (MVP: details and kill confirm)
	showDetails     bool
	showKillConfirm bool
	killTarget      *ports.Port
	killResult      string // error message after failed kill
	successMsg      string // e.g. "Port 3000 terminated."

	// v1.0 Watch mode: auto-refresh every WatchInterval (no heavy polling; one tick in flight).
	WatchEnabled  bool
	WatchInterval time.Duration
}

// NewModel returns an initial model. Caller must provide a Lister (e.g. ports.DefaultLister()).
func NewModel(lister Lister) Model {
	return Model{
		lister:        lister,
		ports:         nil,
		selected:      0,
		WatchInterval: 5 * time.Second,
	}
}

// Init runs once at startup. Triggers initial refresh.
func (m Model) Init() tea.Cmd {
	return m.refreshCmd()
}

// refreshCmd runs lister.List() in a goroutine and returns a Cmd that sends refreshDoneMsg.
func (m Model) refreshCmd() tea.Cmd {
	return func() tea.Msg {
		ports, err := m.lister.List()
		return refreshDoneMsg{ports: ports, err: err}
	}
}

// scheduleTick returns a Cmd that sends tickMsg after WatchInterval (for watch mode).
func (m Model) scheduleTick() tea.Cmd {
	return tea.Tick(m.WatchInterval, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

// displayPorts returns filtered and sorted ports for display. Selection index applies to this slice.
func (m *Model) displayPorts() []ports.Port {
	return filterAndSort(m.ports, m.searchQuery, m.sortKey)
}

func filterAndSort(list []ports.Port, query string, sortKey SortKey) []ports.Port {
	var out []ports.Port
	q := strings.TrimSpace(strings.ToLower(query))
	for _, p := range list {
		if q == "" || portMatches(p, q) {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return lessPort(out[i], out[j], sortKey)
	})
	return out
}

func portMatches(p ports.Port, q string) bool {
	if strings.Contains(strings.ToLower(fmt.Sprint(p.PortNum)), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Process), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.Project()), q) {
		return true
	}
	if strings.Contains(strings.ToLower(p.WorkingDir), q) {
		return true
	}
	if p.Framework != "" && strings.Contains(strings.ToLower(p.Framework), q) {
		return true
	}
	if p.DockerContainerName != "" && strings.Contains(strings.ToLower(p.DockerContainerName), q) {
		return true
	}
	if p.DockerImage != "" && strings.Contains(strings.ToLower(p.DockerImage), q) {
		return true
	}
	return false
}

func lessPort(a, b ports.Port, sortKey SortKey) bool {
	switch sortKey {
	case SortByPort:
		return a.PortNum < b.PortNum
	case SortByUptime:
		ua, ub := a.Uptime(), b.Uptime()
		return ua < ub
	case SortByProcess:
		return strings.ToLower(a.Process) < strings.ToLower(b.Process)
	default:
		return a.PortNum < b.PortNum
	}
}

// SelectedPort returns the currently selected port, or nil if none (from display list).
func (m *Model) SelectedPort() *ports.Port {
	disp := m.displayPorts()
	if len(disp) == 0 {
		return nil
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(disp) {
		m.selected = len(disp) - 1
	}
	p := disp[m.selected]
	return &p
}

// clampSelected ensures selected is within display list length.
func (m *Model) clampSelected() {
	disp := m.displayPorts()
	if m.selected >= len(disp) {
		m.selected = max(0, len(disp)-1)
	}
	if m.selected < 0 {
		m.selected = 0
	}
}

// Update handles messages. UI does not execute OS commands; kill is done via ports.Kill in response to confirm.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.showKillConfirm {
			switch msg.String() {
			case "y", "Y":
				if m.killTarget != nil {
					p := m.killTarget
					r := ports.KillPort(p.PortNum, p.PID)
					m.showKillConfirm = false
					m.killTarget = nil
					if r.OK {
						m.killResult = ""
						m.successMsg = fmt.Sprintf("Port %d terminated.", p.PortNum)
						return m, m.refreshCmd()
					}
					m.killResult = r.Error
					return m, nil
				}
			case "n", "N", "q", "esc":
				m.showKillConfirm = false
				m.killTarget = nil
				return m, nil
			}
			return m, nil
		}
		if m.showDetails {
			switch msg.String() {
			case "q", "esc":
				m.showDetails = false
				return m, nil
			}
			return m, nil
		}
		// Search mode: only Esc and backspace and runes
		if m.searchMode {
			switch msg.String() {
			case "esc":
				m.searchMode = false
				m.searchQuery = ""
				return m, nil
			case "backspace":
				if len(m.searchQuery) > 0 {
					m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
				}
				m.clampSelected()
				return m, nil
			}
			if len(msg.String()) == 1 {
				r := msg.String()[0]
				if r >= 32 && r < 127 {
					m.searchQuery += msg.String()
					m.clampSelected()
					return m, nil
				}
			}
			return m, nil
		}
		m.successMsg = "" // clear success message on any key
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r", "R":
			m.err = ""
			m.successMsg = ""
			return m, m.refreshCmd()
		case "up":
			disp := m.displayPorts()
			if m.selected > 0 && len(disp) > 0 {
				m.selected--
			}
			return m, nil
		case "down", "j", "right":
			disp := m.displayPorts()
			if m.selected < len(disp)-1 {
				m.selected++
			}
			return m, nil
		case "enter":
			if m.SelectedPort() != nil {
				m.showDetails = true
			}
			return m, nil
		case "k":
			if p := m.SelectedPort(); p != nil {
				m.showKillConfirm = true
				dup := *p
				m.killTarget = &dup
			}
			return m, nil
		case "s", "S":
			m.sortKey = SortKey((int(m.sortKey) + 1) % 3)
			m.clampSelected()
			return m, nil
		case "w", "W":
			m.WatchEnabled = !m.WatchEnabled
			if m.WatchEnabled {
				return m, m.scheduleTick()
			}
			return m, nil
		case "/":
			m.searchMode = true
			return m, nil
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tickMsg:
		if m.WatchEnabled {
			return m, tea.Batch(m.refreshCmd(), m.scheduleTick())
		}
		return m, nil
	case refreshDoneMsg:
		m.err = ""
		if msg.err != nil {
			m.err = msg.err.Error()
			m.successMsg = ""
			return m, nil
		}
		m.ports = msg.ports
		m.clampSelected()
		return m, nil
	}
	return m, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
