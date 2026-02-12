package main

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/javiercepeda/tapas/internal/ports"
	"github.com/javiercepeda/tapas/internal/ui"
)

func main() {
	lister := ports.DefaultLister()
	m := ui.NewModel(lister)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
