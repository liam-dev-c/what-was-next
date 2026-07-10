package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m *Model) loadSummary() {
	sum, err := m.store.DailySummary(time.Now())
	if err != nil {
		m.setStatus(err)
		return
	}
	m.summary = sum
}

func (m *Model) loadWeek() {
	ws, err := m.store.WeeklySummary(time.Now(), m.weekStart)
	if err != nil {
		m.setStatus(err)
		return
	}
	m.week = ws
}

func (m Model) updateSummary(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	m.status = ""
	switch msg.String() {
	case "d":
		m.summaryPeriod = periodDay
		m.loadSummary()
	case "w":
		m.summaryPeriod = periodWeek
		m.loadWeek()
	case "esc":
		m.screen = screenTasks
	case ",":
		m.screen = screenSettings
	}
	return m, nil
}

func (m Model) viewSummary() string {
	if m.summaryPeriod == periodWeek {
		return m.viewWeek()
	}
	return m.viewDay()
}

func (m Model) viewDay() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Today — " + m.summary.Day.Format("Mon 2 Jan 2006")))
	b.WriteString("\n")
	b.WriteString(periodTabs(periodDay) + "\n\n")

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
		fmt.Fprintf(&b, "  %-30s %s\n", td.Task.Title, fmtDuration(td.Duration))
	}
	fmt.Fprintf(&b, "\n  %-30s %s\n", "TOTAL", fmtDuration(m.summary.Total))

	b.WriteString(helpStyle.Render(summaryHelp))
	return b.String()
}

func (m Model) viewWeek() string {
	var b strings.Builder
	label := m.week.Start.Format("Mon 2 Jan") + " – " + m.week.End.Format("Mon 2 Jan 2006")
	b.WriteString(titleStyle.Render("This week — " + label))
	b.WriteString("\n")
	b.WriteString(periodTabs(periodWeek) + "\n\n")

	fmt.Fprintf(&b, "%s (%d)\n", selectedStyle.Render("Completed"), len(m.week.Completed))
	if len(m.week.Completed) == 0 {
		b.WriteString(helpStyle.Render("Nothing completed this week.\n"))
	}
	for _, t := range m.week.Completed {
		b.WriteString("  [x] " + t.Title + "\n")
	}

	b.WriteString("\n" + selectedStyle.Render("Time tracked") + "\n")
	if len(m.week.Times) == 0 {
		b.WriteString(helpStyle.Render("No time tracked this week.\n"))
	}
	for _, td := range m.week.Times {
		fmt.Fprintf(&b, "  %-30s %s\n", td.Task.Title, fmtDuration(td.Duration))
	}
	fmt.Fprintf(&b, "\n  %-30s %s\n", "TOTAL", fmtDuration(m.week.Total))

	b.WriteString("\n" + selectedStyle.Render("By day") + "\n ")
	for _, d := range m.week.Days {
		dur := "–"
		if d.Duration > 0 {
			dur = fmtDuration(d.Duration)
		}
		fmt.Fprintf(&b, " %s %s ", d.Day.Format("Mon"), dur)
	}
	b.WriteString("\n")

	b.WriteString(helpStyle.Render(summaryHelp))
	return b.String()
}

const summaryHelp = "\nd day · w week · esc back · , settings · q quit"

// periodTabs renders the day/week selector, highlighting the active period.
func periodTabs(active period) string {
	day, week := faintStyle.Render("d day"), faintStyle.Render("w week")
	if active == periodDay {
		day = selectedStyle.Render("[d]ay")
	} else {
		week = selectedStyle.Render("[w]eek")
	}
	return day + faintStyle.Render("  ·  ") + week
}
