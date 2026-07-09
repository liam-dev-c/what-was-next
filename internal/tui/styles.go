package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Strikethrough(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1)
)
