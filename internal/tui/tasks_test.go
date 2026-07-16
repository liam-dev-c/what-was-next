package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func key(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func TestAddTaskFlow(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks // tasks is the landing screen; this flow lives on tasks
	// Press 'a' to start adding.
	mi, _ := m.updateTasks(key('a'))
	m = mi.(Model)
	if !m.editing {
		t.Fatal("want editing mode after 'a'")
	}
	// Type "Hello".
	for _, r := range "Hello" {
		mi, _ = m.Update(key(r))
		m = mi.(Model)
	}
	// Enter to commit.
	mi, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	if m.editing {
		t.Fatal("want editing off after Enter")
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Hello" {
		t.Fatalf("want 1 task 'Hello', got %+v", m.tasks)
	}
}

func TestEnterOpensDetailsThenToggleDone(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	// Enter on the tasks list now opens Details rather than toggling done.
	mi, _ := m.updateTasks(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	if m.focus != focusDetails {
		t.Fatal("want Details focused after Enter on the tasks list")
	}
	// Enter again, now in Details, toggles the task done.
	mi, _ = m.updateTasks(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	if !m.tasks[0].Done {
		t.Fatal("want task done after Enter in Details")
	}
}

func TestDeleteTask(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	m.focus = focusDetails // delete lives in the Details panel now
	mi, _ := m.updateTasks(key('d'))
	m = mi.(Model)
	if len(m.tasks) != 0 {
		t.Fatalf("want 0 tasks after delete, got %d", len(m.tasks))
	}
	if m.focus != focusTasks {
		t.Fatal("want focus back on Tasks after delete")
	}
}

func TestShiftTabFocusesProjectsAndHistory(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	// shift+tab from Tasks steps back to Projects in the focus cycle.
	mi, _ := m.updateTasks(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	if mi.(Model).focus != focusProjects {
		t.Fatal("want projects focused after shift+tab")
	}
	mi, _ = m.updateTasks(key('h'))
	if mi.(Model).screen != screenHistory {
		t.Fatal("want screenHistory after 'h'")
	}
}

func TestHistoryKeyOpensHistory(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.updateTasks(key('h'))
	if mi.(Model).screen != screenHistory {
		t.Fatal("want screenHistory after 'h'")
	}
}

func TestStatusClearsOnNextCommandKey(t *testing.T) {
	m := newModel(t)
	m.status = "stale error"
	mi, _ := m.updateTasks(key('j')) // any command key
	if mi.(Model).status != "" {
		t.Fatalf("want status cleared on next command key, got %q", mi.(Model).status)
	}
}
