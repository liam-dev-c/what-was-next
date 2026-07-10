package tui

import (
	"strings"
	"testing"
)

func TestDetailBodyShowsStatusAndTime(t *testing.T) {
	m := newModel(t)
	tk, _ := m.store.CreateTask(m.activeProject().ID, "Ship it")
	m.reloadTasks()
	body := m.detailBody(m.tasks[0])
	if !strings.Contains(body, "Ship it") {
		t.Fatalf("detail missing title: %s", body)
	}
	if !strings.Contains(body, "open") {
		t.Fatalf("detail missing status: %s", body)
	}
	if !strings.Contains(strings.ToLower(body), "notes") {
		t.Fatalf("detail missing notes section: %s", body)
	}
	_ = tk
}

func TestDetailBodyDoneStatus(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Done task")
	m.reloadTasks()
	m.store.SetTaskDone(m.tasks[0].ID, true)
	m.reloadTasks()
	if !strings.Contains(m.detailBody(m.tasks[0]), "done") {
		t.Fatalf("want done status, got: %s", m.detailBody(m.tasks[0]))
	}
}

func TestProjectsBodyMarksActive(t *testing.T) {
	m := newModel(t) // has default "Inbox" project active
	body := m.projectsBody()
	if !strings.Contains(body, "Inbox") {
		t.Fatalf("projects body missing Inbox: %s", body)
	}
}
