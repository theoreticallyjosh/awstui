package main

import (
	"awstui/internal/models"
	"fmt"
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	fmt.Print("\033[H\033[2J")

	m := models.NewModel()
	// Start the Bubble Tea program
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
