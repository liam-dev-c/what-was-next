package tui

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m Model) updateSettings(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	switch msg.String() {
	case "esc", ",":
		m.screen = screenSummary
	case "enter", "space", "left", "right", "h", "l":
		// The only setting today: flip the first day of the week.
		if m.weekStart == time.Monday {
			m.weekStart = time.Sunday
		} else {
			m.weekStart = time.Monday
		}
		if err := m.store.SetWeekStart(m.weekStart); err != nil {
			m.setStatus(err)
			return m, nil
		}
		// Keep the weekly summary consistent with the new boundary.
		m.loadWeek()
	}
	return m, nil
}

func (m Model) viewSettings() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Settings"))
	b.WriteString("\n")

	b.WriteString(selectedStyle.Render("> Week starts on: ") + m.weekStart.String() + "\n")

	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	b.WriteString(helpStyle.Render("\nenter/←→ change · esc back · q quit"))
	return b.String()
}
