package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/alireza/work-today/internal/cli"
	"github.com/alireza/work-today/internal/tui"
)

func main() {
	if len(os.Args) > 1 {
		if err := cli.Run(os.Args[1:]); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		return
	}

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
