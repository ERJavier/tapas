package ui

import (
	"fmt"

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

// Model is the root Bubble Tea model.
type Model struct {
	ports     []ports.Port
	selected  int
	lister    Lister
	err       string
	width     int
	height    int

	// Modals (MVP: details and kill confirm)
	showDetails     bool
	showKillConfirm bool
	killTarget      *ports.Port
	killResult      string // error message after failed kill
	successMsg      string // e.g. "Port 3000 terminated."
}

// NewModel returns an initial model. Caller must provide a Lister (e.g. ports.DefaultLister()).
func NewModel(lister Lister) Model {
	return Model{
		lister: lister,
		ports:  nil,
		selected: 0,
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

// SelectedPort returns the currently selected port, or nil if none.
func (m *Model) SelectedPort() *ports.Port {
	if len(m.ports) == 0 {
		return nil
	}
	if m.selected < 0 {
		m.selected = 0
	}
	if m.selected >= len(m.ports) {
		m.selected = len(m.ports) - 1
	}
	return &m.ports[m.selected]
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
		m.successMsg = "" // clear success message on any key
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r", "R":
			m.err = ""
			m.successMsg = ""
			return m, m.refreshCmd()
		case "up":
			if m.selected > 0 {
				m.selected--
			}
			return m, nil
		case "down", "j", "right":
			if m.selected < len(m.ports)-1 {
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
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case refreshDoneMsg:
		m.err = ""
		if msg.err != nil {
			m.err = msg.err.Error()
			m.successMsg = ""
			return m, nil
		}
		m.ports = msg.ports
		if m.selected >= len(m.ports) {
			m.selected = max(0, len(m.ports)-1)
		}
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
