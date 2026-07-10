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
	m.screen = screenHistory
	mi, _ := m.updateSummary(tea.KeyPressMsg{Code: tea.KeyEscape})
	if mi.(Model).screen != screenTasks {
		t.Fatal("want return to tasks on esc")
	}
}

// TestHistoryShowsDailySnapshotByDefault replaces the former
// TestSummaryIsDefaultScreen: tasks is now the landing screen (see
// TestNewSelectsDefaultProject in app_test.go), so this test instead checks
// that opening History via 'h' lands on the daily snapshot.
func TestHistoryShowsDailySnapshotByDefault(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.updateTasks(key('h'))
	m = mi.(Model)
	if m.screen != screenHistory {
		t.Fatalf("want screenHistory after 'h', got %v", m.screen)
	}
	if !strings.Contains(m.viewSummary(), "Today") {
		t.Fatalf("history should default to the daily snapshot:\n%s", m.viewSummary())
	}
}

func TestSummaryDayWeekToggle(t *testing.T) {
	m := newModel(t)
	m.screen = screenHistory

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
