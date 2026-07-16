package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func tab(m Model) Model {
	mi, _ := m.updateTasks(tea.KeyPressMsg{Code: tea.KeyTab})
	return mi.(Model)
}

func shiftTab(m Model) Model {
	mi, _ := m.updateTasks(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	return mi.(Model)
}

func TestCycleFocusForwardAndBack(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks // starts focused on Tasks

	m = tab(m)
	if m.focus != focusDetails {
		t.Fatalf("tab from Tasks: want Details, got %v", m.focus)
	}
	m = tab(m)
	if m.focus != focusProjects {
		t.Fatalf("tab from Details: want Projects, got %v", m.focus)
	}
	m = tab(m)
	if m.focus != focusTasks {
		t.Fatalf("tab from Projects: want Tasks, got %v", m.focus)
	}

	// shift+tab walks the cycle backwards.
	m = shiftTab(m)
	if m.focus != focusProjects {
		t.Fatalf("shift+tab from Tasks: want Projects, got %v", m.focus)
	}
}

func TestDetailsFocusEntersEditModes(t *testing.T) {
	base := newModel(t)
	tk, _ := base.store.CreateTask(base.activeProject().ID, "Task")
	base.reloadTasks()
	base.focus = focusDetails

	// 'e' edits the title.
	mi, _ := base.updateTasks(key('e'))
	m := mi.(Model)
	if !m.editing || m.taggingTask || m.addingProject || m.editID != tk.ID {
		t.Fatalf("'e' should edit the task title, got editing=%v tagging=%v id=%d",
			m.editing, m.taggingTask, m.editID)
	}

	// 'g' edits the tags.
	mi, _ = base.updateTasks(key('g'))
	m = mi.(Model)
	if !m.editing || !m.taggingTask || m.editID != tk.ID {
		t.Fatalf("'g' should edit tags, got editing=%v tagging=%v id=%d",
			m.editing, m.taggingTask, m.editID)
	}
}

func TestTasksListDoesNotEditContent(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks() // focus defaults to Tasks
	// 'e'/'n'/'g' are Details-only; on the tasks list they are inert.
	for _, r := range []rune{'e', 'n', 'g'} {
		mi, _ := m.updateTasks(key(r))
		got := mi.(Model)
		if got.editing || got.notesEditing {
			t.Fatalf("%q must not start editing from the tasks list", r)
		}
	}
}
