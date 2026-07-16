package tui

import (
	"testing"
	"time"
)

// ptr is a small helper for building *time.Time literals in tests.
func ptr(t time.Time) *time.Time { return &t }

func titles(tasks []task) []string {
	out := make([]string, len(tasks))
	for i, t := range tasks {
		out[i] = t.Title
	}
	return out
}

func TestPartitionTasksHidesOldCompleted(t *testing.T) {
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.Local)
	todayDone := time.Date(2026, 7, 16, 9, 0, 0, 0, time.Local)
	yesterdayDone := time.Date(2026, 7, 15, 23, 0, 0, 0, time.Local)

	tasks := []task{
		{Title: "open-a", SortOrder: 1},
		{Title: "done-today", Done: true, DoneAt: ptr(todayDone)},
		{Title: "done-yesterday", Done: true, DoneAt: ptr(yesterdayDone)},
		{Title: "open-b", SortOrder: 2},
	}

	vis, doneStart := partitionTasks(tasks, now, false)
	if got := titles(vis); !equal(got, []string{"open-a", "open-b", "done-today"}) {
		t.Fatalf("default view: got %v", got)
	}
	if doneStart != 2 {
		t.Fatalf("want doneStart 2 (two open tasks), got %d", doneStart)
	}

	vis, doneStart = partitionTasks(tasks, now, true)
	if got := titles(vis); !equal(got, []string{"open-a", "open-b", "done-today", "done-yesterday"}) {
		t.Fatalf("show-all view: got %v", got)
	}
	if doneStart != 2 {
		t.Fatalf("show-all: want doneStart 2, got %d", doneStart)
	}
}

func TestPartitionTasksSortsCompletedNewestFirst(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.Local)
	early := time.Date(2026, 7, 16, 8, 0, 0, 0, time.Local)
	late := time.Date(2026, 7, 16, 11, 0, 0, 0, time.Local)

	tasks := []task{
		{Title: "done-early", Done: true, DoneAt: ptr(early)},
		{Title: "done-late", Done: true, DoneAt: ptr(late)},
	}
	vis, doneStart := partitionTasks(tasks, now, false)
	if doneStart != 0 {
		t.Fatalf("want doneStart 0 (no open tasks), got %d", doneStart)
	}
	if got := titles(vis); !equal(got, []string{"done-late", "done-early"}) {
		t.Fatalf("want newest-first, got %v", got)
	}
}

func TestPartitionTasksNoCompleted(t *testing.T) {
	now := time.Date(2026, 7, 16, 12, 0, 0, 0, time.Local)
	tasks := []task{{Title: "a", SortOrder: 1}, {Title: "b", SortOrder: 2}}
	vis, doneStart := partitionTasks(tasks, now, false)
	if doneStart != len(vis) {
		t.Fatalf("want doneStart == len(vis) when no completed, got %d/%d", doneStart, len(vis))
	}
}

func TestCompletedToggleKey(t *testing.T) {
	m := newModel(t)
	m.tasks = []task{
		{ID: 1, Title: "open", SortOrder: 1},
		{ID: 2, Title: "old-done", Done: true, DoneAt: ptr(time.Now().AddDate(0, 0, -2))},
	}
	if m.visibleCount() != 1 {
		t.Fatalf("want old completed hidden by default, visible=%d", m.visibleCount())
	}
	mi, _ := m.updateTasks(key('c'))
	m = mi.(Model)
	if !m.showAllCompleted {
		t.Fatal("want showAllCompleted after 'c'")
	}
	if m.visibleCount() != 2 {
		t.Fatalf("want old completed shown after toggle, visible=%d", m.visibleCount())
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
