package styles

import "github.com/charmbracelet/lipgloss"

// Tokyo Night Color Palette
var (
	TokyoNightBlue    = lipgloss.Color("#7AA2F7")
	TokyoNightGreen   = lipgloss.Color("#9EEB49")
	TokyoNightYellow  = lipgloss.Color("#E0AF68")
	TokyoNightRed     = lipgloss.Color("#F7768E")
	TokyoNightPurple  = lipgloss.Color("#BB9AF7")
	TokyoNightCyan    = lipgloss.Color("#7DCFFF")
	TokyoNightGray    = lipgloss.Color("#A9B1D6")
	TokyoNightDarkBg  = lipgloss.Color("#1A1B26")
	TokyoNightLightFg = lipgloss.Color("#C0CAF5")
)

// Define styles using lipgloss
var (
	AppStyle = lipgloss.NewStyle().Padding(1, 2)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(TokyoNightPurple).
			Bold(true).
			PaddingBottom(1)

	StatusStyle = lipgloss.NewStyle().
			Foreground(TokyoNightGray).
			PaddingTop(1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(TokyoNightRed).
			Bold(true).
			PaddingTop(1)

	ConfirmStyle = lipgloss.NewStyle().
			Foreground(TokyoNightGreen).
			Bold(true).
			PaddingTop(1)

	DetailStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(TokyoNightBlue).
			Padding(1, 2)

	TitleStyle          = lipgloss.NewStyle().Bold(true)
	DescriptionStyle    = lipgloss.NewStyle().Foreground(TokyoNightGray)
	SelectedItemStyle   = lipgloss.NewStyle().Foreground(TokyoNightBlue)
	UnselectedItemStyle = lipgloss.NewStyle()

	MenuItemStyle         = lipgloss.NewStyle()
	SelectedMenuItemStyle = lipgloss.NewStyle().Foreground(TokyoNightBlue).String()
)
