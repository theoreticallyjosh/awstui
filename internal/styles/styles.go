package styles

import (
	"github.com/charmbracelet/lipgloss"

	tint "github.com/lrstanley/bubbletint"
)

var Theme tint.Tint

var (
	AppStyle,
	HeaderBarStyle,
	HeaderStyle,
	SubHeaderStyle,
	ErrorStyle,
	ConfirmStyle,
	DetailStyle,
	TitleStyle,
	DescriptionStyle,
	SelectedItemStyle,
	UnselectedItemStyle,
	ExtrasStyle,
	StatusStyle,
	HelpStyle,
	KeysStyle,
	MenuItemStyle,
	SelectedMenuItemStyle,
	ActivePager,
	InactivePager lipgloss.Style
)

func LoadStyle() {

	AppStyle = lipgloss.NewStyle().Padding(1, 2)

	HeaderBarStyle = lipgloss.NewStyle().Foreground(Theme.Fg()).Background(Theme.Bg())

	HeaderStyle = lipgloss.NewStyle().
		Foreground(Theme.Purple()).
		Background(Theme.BrightBlack()).
		Bold(true)

	SubHeaderStyle = lipgloss.NewStyle().
		Foreground(Theme.Green()).
		Background(Theme.Bg()).
		Bold(true)

	ErrorStyle = lipgloss.NewStyle().
		Foreground(Theme.Red()).
		Bold(true).
		PaddingTop(1)

	ConfirmStyle = lipgloss.NewStyle().
		Foreground(Theme.Green()).
		Bold(true).
		PaddingTop(1)

	DetailStyle = lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true).
		BorderForeground(Theme.Blue()).
		Padding(1, 2)

	TitleStyle = lipgloss.NewStyle().Bold(true)
	DescriptionStyle = lipgloss.NewStyle().Foreground(Theme.Fg())
	SelectedItemStyle = lipgloss.NewStyle().Foreground(Theme.BrightBlue()).Background(Theme.SelectionBg()).Border(lipgloss.NormalBorder(), false, false, false, true).
		BorderForeground(Theme.Blue()).BorderBackground(Theme.SelectionBg())
	UnselectedItemStyle = lipgloss.NewStyle().PaddingLeft(1)

	ExtrasStyle = lipgloss.NewStyle().Foreground(Theme.Fg())

	StatusStyle = lipgloss.NewStyle().Foreground(Theme.Fg()).Background(Theme.Bg()).PaddingLeft(1)

	HelpStyle = lipgloss.NewStyle().Foreground(Theme.BrightBlack())

	KeysStyle = lipgloss.NewStyle().Foreground(Theme.Bg())
	MenuItemStyle = lipgloss.NewStyle().PaddingLeft(1)
	SelectedMenuItemStyle = lipgloss.NewStyle().Foreground(Theme.Blue())

	ActivePager = lipgloss.NewStyle().Foreground(Theme.Fg())

	InactivePager = lipgloss.NewStyle().Foreground(Theme.BrightBlack())
}
