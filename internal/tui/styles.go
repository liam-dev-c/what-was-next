package tui

import "charm.land/lipgloss/v2"

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

	// faintStyle is helpStyle without the top margin, for dim text used inline
	// (e.g. the day/week tab selector) rather than as a trailing block.
	faintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240"))
)
