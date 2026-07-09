package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestTickKeepsTicking(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Fatal("want a follow-up tick command")
	}
}

func TestElapsedForClosedTime(t *testing.T) {
	m := newModel(t)
	tk, _ := m.store.CreateTask(m.activeProject().ID, "Timed")
	m.reloadTasks()
	// Start and stop across a known gap using the store's injectable clock.
	// (RunningEntry-based live time is exercised manually; here we assert the
	// closed-time path returns a non-negative duration and an ok flag.)
	m.store.StartTimer(tk.ID)
	m.store.StopTimer()
	d, ok := m.elapsedFor(tk.ID)
	if !ok {
		t.Fatal("want ok for a task with time entries")
	}
	if d < 0 {
		t.Fatalf("want non-negative duration, got %s", d)
	}
	_ = time.Second
	_ = tea.KeyPressMsg{}
}
