package tui

import "github.com/charmbracelet/lipgloss"

const (
	colorBorder  = lipgloss.Color("240")
	colorMuted   = lipgloss.Color("245")
	colorText    = lipgloss.Color("252")
	colorPrimary = lipgloss.Color("99")
	colorAccent  = lipgloss.Color("205")
)

var (
	appStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(1, 2)

	headerTitleStyle = lipgloss.NewStyle().
				Foreground(colorText).
				Bold(true)

	headerMarkStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	bannerStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	subtleStyle = lipgloss.NewStyle().Foreground(colorMuted)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(0, 1)

	focusedPanelStyle = panelStyle.
				BorderForeground(colorAccent)

	panelTitleStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(colorAccent).
				Bold(true)

	summaryStyle = lipgloss.NewStyle().
			Foreground(colorText).
			Padding(0, 1)

	keyStyle = lipgloss.NewStyle().
			Foreground(colorPrimary).
			Bold(true)
)
