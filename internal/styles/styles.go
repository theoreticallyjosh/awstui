package styles

import (
	"github.com/charmbracelet/lipgloss"

	tint "github.com/lrstanley/bubbletint"
)

var theme = tint.TintTokyoNightStorm

var (
	AppStyle = lipgloss.NewStyle().Padding(1, 2)

	HeaderBarStyle = lipgloss.NewStyle().Foreground(theme.Fg()).Background(theme.Bg())

	HeaderStyle = lipgloss.NewStyle().
			Foreground(theme.Purple()).
			Background(theme.BrightBlack()).
			Bold(true)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(theme.Green()).
			Background(theme.Bg()).
			Bold(true)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(theme.Red()).
			Bold(true).
			PaddingTop(1)

	ConfirmStyle = lipgloss.NewStyle().
			Foreground(theme.Green()).
			Bold(true).
			PaddingTop(1)

	DetailStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(theme.Blue()).
			Padding(1, 2)

	TitleStyle        = lipgloss.NewStyle().Bold(true)
	DescriptionStyle  = lipgloss.NewStyle().Foreground(theme.Fg())
	SelectedItemStyle = lipgloss.NewStyle().Foreground(theme.BrightBlue()).Background(theme.SelectionBg()).Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(theme.Blue()).BorderBackground(theme.SelectionBg())
	UnselectedItemStyle = lipgloss.NewStyle().PaddingLeft(1)

	ExtrasStyle = lipgloss.NewStyle().Foreground(theme.Fg())

	StatusStyle = lipgloss.NewStyle().Foreground(theme.Fg()).Background(theme.Bg()).PaddingLeft(1)

	HelpStyle = lipgloss.NewStyle().Foreground(theme.BrightBlack())

	KeysStyle             = lipgloss.NewStyle().Foreground(theme.Bg())
	MenuItemStyle         = lipgloss.NewStyle().PaddingLeft(1)
	SelectedMenuItemStyle = lipgloss.NewStyle().Foreground(theme.Blue()).String()

	ActivePager = lipgloss.NewStyle().Foreground(theme.Fg())

	InactivePager = lipgloss.NewStyle().Foreground(theme.BrightBlack())
)
