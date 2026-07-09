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
