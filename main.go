package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbletea"
	"github.com/javiercepeda/tapas/internal/ports"
	"github.com/javiercepeda/tapas/internal/ui"
)

func main() {
	ascii := flag.Bool("ascii", false, "Use ASCII indicators only (! public, - Docker)")
	flag.Parse()

	lister := ports.DefaultLister()
	m := ui.NewModel(lister, *ascii)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
