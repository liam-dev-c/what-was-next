package tui

import (
	"path/filepath"
	"testing"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

// newFileModel builds a Model backed by a real on-disk DB, plus a second Store
// on the same file standing in for the MCP server writing concurrently.
func newFileModel(t *testing.T) (Model, *store.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "wwn.db")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open self: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	other, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open other: %v", err)
	}
	t.Cleanup(func() { other.Close() })
	m, err := New(s)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m, other
}

func TestTickPicksUpExternalTask(t *testing.T) {
	m, other := newFileModel(t)
	before := len(m.tasks)

	if _, err := other.CreateTask(m.activeProject().ID, "from mcp"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	upd, cmd := m.Update(tickMsg{})
	m = upd.(Model)
	if cmd == nil {
		t.Fatal("tick should reschedule itself")
	}
	if len(m.tasks) != before+1 {
		t.Fatalf("want %d tasks after external add, got %d", before+1, len(m.tasks))
	}
}

func TestTickSkipsRefreshWhileEditing(t *testing.T) {
	m, other := newFileModel(t)
	before := len(m.tasks)
	m.editing = true

	if _, err := other.CreateTask(m.activeProject().ID, "from mcp"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}

	upd, _ := m.Update(tickMsg{})
	m = upd.(Model)
	if len(m.tasks) != before {
		t.Fatalf("should not reload mid-edit: want %d tasks, got %d", before, len(m.tasks))
	}

	// Once editing ends, the next tick catches up.
	m.editing = false
	upd, _ = m.Update(tickMsg{})
	m = upd.(Model)
	if len(m.tasks) != before+1 {
		t.Fatalf("should reload after edit ends: want %d tasks, got %d", before+1, len(m.tasks))
	}
}

func TestTickPreservesSelectedTask(t *testing.T) {
	m, other := newFileModel(t)
	proj := m.activeProject().ID
	if _, err := other.CreateTask(proj, "one"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if _, err := other.CreateTask(proj, "two"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	// Pull those in and select the second one.
	upd, _ := m.Update(tickMsg{})
	m = upd.(Model)
	if len(m.tasks) < 2 {
		t.Fatalf("setup: want >=2 tasks, got %d", len(m.tasks))
	}
	m.cursor = 1
	selID := m.tasks[1].ID

	// An external insert must not move the cursor off the selected task.
	if _, err := other.CreateTask(proj, "three"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	upd, _ = m.Update(tickMsg{})
	m = upd.(Model)
	if got := m.tasks[m.cursor].ID; got != selID {
		t.Fatalf("cursor moved off selected task: want id %d, got %d", selID, got)
	}
}
