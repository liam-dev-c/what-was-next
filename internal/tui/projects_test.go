package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSelectProjectReloadsTasks(t *testing.T) {
	m := newModel(t)
	p, _ := m.store.CreateProject("Work")
	m.store.CreateTask(p.ID, "Work task")
	m.reloadProjects()
	m.screen = screenProjects
	m.projCursor = 0

	// Move down to "Work" (index 1) and select it.
	mi, _ := m.updateProjects(key('j'))
	m = mi.(Model)
	mi, _ = m.updateProjects(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)

	if m.screen != screenTasks {
		t.Fatal("want return to tasks after selecting project")
	}
	if m.activeProject().Name != "Work" {
		t.Fatalf("want active 'Work', got %q", m.activeProject().Name)
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Work task" {
		t.Fatalf("want Work's tasks loaded, got %+v", m.tasks)
	}
}

func TestAddProjectFlow(t *testing.T) {
	m := newModel(t)
	m.screen = screenProjects
	mi, _ := m.updateProjects(key('a'))
	m = mi.(Model)
	for _, r := range "Side" {
		mi, _ = m.Update(key(r))
		m = mi.(Model)
	}
	mi, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	names := make([]string, len(m.projects))
	for i, p := range m.projects {
		names[i] = p.Name
	}
	found := false
	for _, n := range names {
		if n == "Side" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a 'Side' project, got %v", names)
	}
}
