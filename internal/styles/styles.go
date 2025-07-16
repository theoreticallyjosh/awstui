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
	TokyoNightDarkFG  = lipgloss.Color("#333336")
	TokyoNightLightBg = lipgloss.Color("#1F2335")
	TokyoNightLightFg = lipgloss.Color("#3B4261")
)

// Define styles using lipgloss
var (
	AppStyle = lipgloss.NewStyle().Padding(1, 2)

	HeaderBarStyle = lipgloss.NewStyle().Foreground(TokyoNightGray).Background(TokyoNightDarkBg)

	HeaderStyle = lipgloss.NewStyle().
			Foreground(TokyoNightPurple).
			Background(TokyoNightDarkBg).
			Bold(true)

	SubHeaderStyle = lipgloss.NewStyle().
			Foreground(TokyoNightGreen).
			Background(TokyoNightDarkBg).
			Bold(true)

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

	TitleStyle        = lipgloss.NewStyle().Bold(true)
	DescriptionStyle  = lipgloss.NewStyle().Foreground(TokyoNightLightFg)
	SelectedItemStyle = lipgloss.NewStyle().Foreground(TokyoNightBlue).Background(TokyoNightLightBg).Border(lipgloss.NormalBorder(), false, false, false, true).
				BorderForeground(TokyoNightBlue).BorderBackground(TokyoNightLightBg)
	UnselectedItemStyle = lipgloss.NewStyle().PaddingLeft(1)

	ExtrasStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Dark:  string(TokyoNightLightFg),
		Light: string(TokyoNightDarkFG),
	})

	StatusStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Dark:  string(TokyoNightLightFg),
		Light: string(TokyoNightDarkFG),
	}).Background(TokyoNightDarkBg).PaddingLeft(1)

	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Dark:  string(TokyoNightLightFg),
		Light: string(TokyoNightDarkFG),
	})

	KeysStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: string(TokyoNightLightFg),
		Dark:  string(TokyoNightDarkFG),
	})
	MenuItemStyle         = lipgloss.NewStyle().PaddingLeft(1)
	SelectedMenuItemStyle = lipgloss.NewStyle().Foreground(TokyoNightBlue).String()

	ActivePager = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Dark:  string(TokyoNightGray),
		Light: string(TokyoNightDarkFG),
	})

	InactivePager = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{
		Light: string(TokyoNightLightFg),
		Dark:  string(TokyoNightDarkFG),
	})
)
