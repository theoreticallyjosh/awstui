package main

import "github.com/charmbracelet/lipgloss"

// Tokyo Night Color Palette
var (
	tokyoNightBlue    = lipgloss.Color("#7AA2F7")
	tokyoNightGreen   = lipgloss.Color("#9EEB49")
	tokyoNightYellow  = lipgloss.Color("#E0AF68")
	tokyoNightRed     = lipgloss.Color("#F7768E")
	tokyoNightPurple  = lipgloss.Color("#BB9AF7")
	tokyoNightCyan    = lipgloss.Color("#7DCFFF")
	tokyoNightGray    = lipgloss.Color("#A9B1D6")
	tokyoNightDarkBg  = lipgloss.Color("#1A1B26")
	tokyoNightLightFg = lipgloss.Color("#C0CAF5")
)

// Define styles using lipgloss
var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	headerStyle = lipgloss.NewStyle().
			Foreground(tokyoNightPurple).
			Bold(true).
			PaddingBottom(1)

	statusStyle = lipgloss.NewStyle().
			Foreground(tokyoNightGray).
			PaddingTop(1)

	errorStyle = lipgloss.NewStyle().
			Foreground(tokyoNightRed).
			Bold(true).
			PaddingTop(1)

	confirmStyle = lipgloss.NewStyle().
			Foreground(tokyoNightGreen).
			Bold(true).
			PaddingTop(1)

	detailStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder(), true).
			BorderForeground(tokyoNightBlue).
			Padding(1, 2)

	titleStyle          = lipgloss.NewStyle().Bold(true)
	descriptionStyle    = lipgloss.NewStyle().Foreground(tokyoNightGray)
	selectedItemStyle   = lipgloss.NewStyle().Foreground(tokyoNightBlue)
	unselectedItemStyle = lipgloss.NewStyle()

	menuItemStyle         = lipgloss.NewStyle()
	selectedMenuItemStyle = lipgloss.NewStyle().Foreground(tokyoNightBlue).String()
)
