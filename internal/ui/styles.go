package ui

import "charm.land/lipgloss/v2"

var (
	purple   = lipgloss.Color("#7D56F4")
	green    = lipgloss.Color("#04B575")
	red      = lipgloss.Color("#FF4672")
	yellow   = lipgloss.Color("#F1FA8C")
	cyan     = lipgloss.Color("#8BE9FD")
	orange   = lipgloss.Color("#FFB86C")
	white    = lipgloss.Color("#FAFAFA")
	dimWhite = lipgloss.Color("#888888")
	midGray  = lipgloss.Color("#555555")
	darkGray = lipgloss.Color("#333333")

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white).
			Background(purple).
			Padding(0, 2)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(midGray).
			Padding(1, 2)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(purple).
			MarginBottom(1)

	bigNumStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(green)

	labelStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	valueStyle = lipgloss.NewStyle().
			Foreground(white)

	boldValueStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(white)

	warnValStyle = lipgloss.NewStyle().
			Foreground(yellow)

	errValStyle = lipgloss.NewStyle().
			Foreground(red)

	cyanStyle = lipgloss.NewStyle().
			Foreground(cyan)

	orangeStyle = lipgloss.NewStyle().
			Foreground(orange)

	sparkStyle = lipgloss.NewStyle().
			Foreground(purple)

	tableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(purple)

	statusStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	helpStyle = lipgloss.NewStyle().
			Foreground(dimWhite)

	slugStyle = lipgloss.NewStyle().
			Foreground(cyan).
			Bold(true)
)
