package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestAddTaskFlow(t *testing.T) {
	m := newModel(t)
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
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(Model)
	if m.editing {
		t.Fatal("want editing off after Enter")
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Hello" {
		t.Fatalf("want 1 task 'Hello', got %+v", m.tasks)
	}
}

func TestToggleDone(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	mi, _ := m.updateTasks(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(Model)
	if !m.tasks[0].Done {
		t.Fatal("want task done after Enter toggle")
	}
}

func TestDeleteTask(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	mi, _ := m.updateTasks(key('d'))
	m = mi.(Model)
	if len(m.tasks) != 0 {
		t.Fatalf("want 0 tasks after delete, got %d", len(m.tasks))
	}
}

func TestSwitchToProjectsAndSummary(t *testing.T) {
	m := newModel(t)
	mi, _ := m.updateTasks(key('p'))
	if mi.(Model).screen != screenProjects {
		t.Fatal("want screenProjects after 'p'")
	}
	mi, _ = m.updateTasks(key('s'))
	if mi.(Model).screen != screenSummary {
		t.Fatal("want screenSummary after 's'")
	}
}
