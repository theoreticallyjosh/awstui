package main

import (
	"log"

	"github.com/theoreticallyjosh/awstui/internal/config"
	"github.com/theoreticallyjosh/awstui/internal/models"
	"github.com/theoreticallyjosh/awstui/internal/styles"

	tea "github.com/charmbracelet/bubbletea"
	tint "github.com/lrstanley/bubbletint"
)

func main() {

	conf := config.LoadConfig()
	tint.NewDefaultRegistry()
	styles.Theme, _ = tint.GetTint(conf.Theme)
	styles.LoadStyle()
	tea.ClearScreen()
	m := models.NewModel()
	// Start the Bubble Tea program
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		log.Fatalf("Alas, there's been an error: %v", err)
	}
}
