package main

import (
	"fmt"
	"os"

	"claude_monitor/internal/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	p := tea.NewProgram(ui.NewDashboard())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}
}
