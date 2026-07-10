package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSummaryLoadsAndRenders(t *testing.T) {
	m := newModel(t)
	tk, _ := m.store.CreateTask(m.activeProject().ID, "Wrote report")
	m.store.SetTaskDone(tk.ID, true)
	m.reloadTasks()

	m.loadSummary()
	out := m.viewSummary()
	if !strings.Contains(out, "Wrote report") {
		t.Fatalf("summary should list completed task, got:\n%s", out)
	}
}

func TestSummaryEscReturns(t *testing.T) {
	m := newModel(t)
	m.screen = screenSummary
	mi, _ := m.updateSummary(tea.KeyPressMsg{Code: tea.KeyEscape})
	if mi.(Model).screen != screenTasks {
		t.Fatal("want return to tasks on esc")
	}
}

func TestSummaryIsDefaultScreen(t *testing.T) {
	m := newModel(t)
	if m.screen != screenSummary {
		t.Fatalf("want summary as landing screen, got %v", m.screen)
	}
	if !strings.Contains(m.viewSummary(), "Today") {
		t.Fatalf("landing view should be the daily summary:\n%s", m.viewSummary())
	}
}

func TestSummaryDayWeekToggle(t *testing.T) {
	m := newModel(t)
	m.screen = screenSummary

	mi, _ := m.updateSummary(tea.KeyPressMsg{Code: 'w', Text: "w"})
	m = mi.(Model)
	if m.summaryPeriod != periodWeek {
		t.Fatal("want week period after pressing 'w'")
	}
	if !strings.Contains(m.viewSummary(), "This week") {
		t.Fatalf("week view should show 'This week':\n%s", m.viewSummary())
	}
	if !strings.Contains(m.viewSummary(), "By day") {
		t.Fatalf("week view should show the per-day breakdown:\n%s", m.viewSummary())
	}

	mi, _ = m.updateSummary(tea.KeyPressMsg{Code: 'd', Text: "d"})
	m = mi.(Model)
	if m.summaryPeriod != periodDay {
		t.Fatal("want day period after pressing 'd'")
	}
	if !strings.Contains(m.viewSummary(), "Today") {
		t.Fatalf("day view should show 'Today':\n%s", m.viewSummary())
	}
}
