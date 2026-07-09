package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) loadSummary() {
	sum, err := m.store.DailySummary(time.Now())
	if err != nil {
		m.setStatus(err)
		return
	}
	m.summary = sum
}

func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	switch msg.String() {
	case "esc", "s", "q":
		m.screen = screenTasks
	}
	return m, nil
}

func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Today — " + m.summary.Day.Format("Mon 2 Jan 2006")))
	b.WriteString("\n")

	b.WriteString(selectedStyle.Render("Completed"))
	b.WriteString("\n")
	if len(m.summary.Completed) == 0 {
		b.WriteString(helpStyle.Render("Nothing completed yet today.\n"))
	}
	for _, t := range m.summary.Completed {
		b.WriteString("  [x] " + t.Title + "\n")
	}

	b.WriteString("\n" + selectedStyle.Render("Time tracked") + "\n")
	if len(m.summary.Times) == 0 {
		b.WriteString(helpStyle.Render("No time tracked today.\n"))
	}
	for _, td := range m.summary.Times {
		b.WriteString(fmt.Sprintf("  %-30s %s\n", td.Task.Title, fmtDuration(td.Duration)))
	}
	b.WriteString(fmt.Sprintf("\n  %-30s %s\n", "TOTAL", fmtDuration(m.summary.Total)))

	b.WriteString(helpStyle.Render("\nesc back · q quit"))
	return b.String()
}
