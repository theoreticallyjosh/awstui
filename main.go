package main

import (
	"awstui/internal/models"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	tea.ClearScreen()

	m := models.NewModel()
	// Start the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
