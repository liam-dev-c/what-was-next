package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestProjectsPanelFocusAndSelect(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	p, _ := m.store.CreateProject("Work")
	m.store.CreateTask(p.ID, "Work task")
	m.reloadProjects()

	// shift+tab from Tasks steps back to the Projects panel.
	mi, _ := m.updateTasks(tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift})
	m = mi.(Model)
	if m.focus != focusProjects {
		t.Fatal("want projects focused after shift+tab")
	}
	// Move to "Work" (index 1) and select it.
	mi, _ = m.updateTasks(key('j'))
	m = mi.(Model)
	mi, _ = m.updateTasks(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	if m.activeProject().Name != "Work" {
		t.Fatalf("want active 'Work', got %q", m.activeProject().Name)
	}
	if m.focus != focusTasks {
		t.Fatal("want focus back on tasks after selecting a project")
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Work task" {
		t.Fatalf("want Work's tasks, got %+v", m.tasks)
	}
}

func TestAddProjectViaPanel(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.focus = focusProjects
	mi, _ := m.updateTasks(key('a'))
	m = mi.(Model)
	if !m.editing || !m.addingProject {
		t.Fatal("want project-add input active")
	}
	for _, r := range "Side" {
		mi, _ = m.Update(key(r))
		m = mi.(Model)
	}
	mi, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	found := false
	for _, p := range m.projects {
		if p.Name == "Side" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want 'Side' project created")
	}
}
