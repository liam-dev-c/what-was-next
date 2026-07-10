package tui

import "charm.land/lipgloss/v2"

// Soft truecolor palette tuned for a dark background (dark-mode Ghostty).
var (
	accentColor  = lipgloss.Color("#C8A2FF") // lavender: titles, selection, focus
	borderColor  = lipgloss.Color("#4A4E5A") // muted grey: unfocused panel frame
	errorColor   = lipgloss.Color("#F2A0A0") // rose: errors/status
	successColor = lipgloss.Color("#A7E0B8") // mint: done / running timer
	faintColor   = lipgloss.Color("#6B7280") // slate: hints/labels
)

const (
	projectsPanelWidth = 24 // total cells incl. borders
	minWorkspaceWidth  = 80 // below this, fall back to single column
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	doneStyle = lipgloss.NewStyle().
			Foreground(faintColor).
			Strikethrough(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(faintColor).
			MarginTop(1)

	faintStyle = lipgloss.NewStyle().
			Foreground(faintColor)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

	borderFocusStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(accentColor)
)

// panel renders body inside a rounded border sized to width×height cells,
// with title as a styled header line. Border is accent-coloured when focused.
func panel(title, body string, focused bool, width, height int) string {
	style := borderStyle
	if focused {
		style = borderFocusStyle
	}
	head := panelTitleStyle.Render(title)
	content := head + "\n" + body
	// Lipgloss v2 Width/Height are border-inclusive, so pass the full cell
	// dimensions here (not width-2/height-2). This makes every panel exactly
	// width×height whether its content fills the box or is padded — which keeps
	// the left sidebar the same height as the stacked right column.
	return style.Width(width).Height(height).Render(content)
}
