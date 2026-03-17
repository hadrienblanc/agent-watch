package main

import (
	"flag"
	"fmt"
	"os"

	"claude_monitor/internal/server"
	"claude_monitor/internal/ui"

	tea "charm.land/bubbletea/v2"
)

func main() {
	port := flag.Int("port", 9999, "HTTP server port for peer connections")
	flag.Parse()

	// Create dashboard
	d := ui.NewDashboard()
	d.SetPort(*port)

	// Start HTTP server in background
	statsProvider := func() interface{} {
		return d.GetLocalStats()
	}
	srv := server.New(*port, statsProvider)
	go func() {
		if err := srv.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "HTTP server error: %v\n", err)
		}
	}()

	// Run TUI
	p := tea.NewProgram(d)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Erreur: %v\n", err)
		os.Exit(1)
	}
}
