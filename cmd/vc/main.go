package main

import (
	"fmt"
	"os"

	"github.com/avitaltamir/vibecommander/internal/app"
	tea "github.com/charmbracelet/bubbletea"
)

var version = "dev"

func main() {
	// Set the app version for display in the UI
	app.Version = version

	if len(os.Args) > 1 && (os.Args[1] == "-v" || os.Args[1] == "--version") {
		fmt.Println("vc", version)
		return
	}

	p := tea.NewProgram(
		app.New(),
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
