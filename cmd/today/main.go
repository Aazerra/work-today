package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alireza/work-today/internal/tui"
)

func main() {
	m, err := tui.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "work-today: failed to load tasks: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "work-today: %v\n", err)
		os.Exit(1)
	}
}
