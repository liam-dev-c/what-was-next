package store

import (
	"testing"
	"time"
)

// advancingClock returns times that step forward by one hour per call,
// starting from the fixed base. Lets us assert deterministic durations.
func advancingClock() func() time.Time {
	base := time.Date(2026, 7, 9, 9, 0, 0, 0, time.UTC)
	n := 0
	return func() time.Time {
		t := base.Add(time.Duration(n) * time.Hour)
		n++
		return t
	}
}

func TestStartTimerCreatesRunningEntry(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Track me")
	entry, err := s.StartTimer(tk.ID)
	if err != nil {
		t.Fatalf("StartTimer: %v", err)
	}
	if entry.TaskID != tk.ID || entry.EndedAt != nil {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	running, err := s.RunningEntry()
	if err != nil {
		t.Fatalf("RunningEntry: %v", err)
	}
	if running == nil || running.TaskID != tk.ID {
		t.Fatalf("want running entry for task %d, got %+v", tk.ID, running)
	}
}

func TestStartTimerStopsPrevious(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	a, _ := s.CreateTask(pid, "A")
	b, _ := s.CreateTask(pid, "B")
	if _, err := s.StartTimer(a.ID); err != nil {
		t.Fatalf("StartTimer A: %v", err)
	}
	if _, err := s.StartTimer(b.ID); err != nil {
		t.Fatalf("StartTimer B: %v", err)
	}
	running, _ := s.RunningEntry()
	if running == nil || running.TaskID != b.ID {
		t.Fatalf("want B running, got %+v", running)
	}
	// Exactly one entry should be open.
	var open int
	s.db.QueryRow(`SELECT COUNT(*) FROM time_entries WHERE ended_at IS NULL`).Scan(&open)
	if open != 1 {
		t.Fatalf("want 1 open entry, got %d", open)
	}
}

func TestTaskDurationSumsClosedEntries(t *testing.T) {
	s := newTestStore(t)
	s.now = advancingClock() // 9:00, 10:00, 11:00, ...
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Work") // consumes 9:00 (created_at)
	// start -> 10:00, stop -> 11:00  => 1h
	if _, err := s.StartTimer(tk.ID); err != nil {
		t.Fatalf("StartTimer: %v", err)
	}
	if err := s.StopTimer(); err != nil {
		t.Fatalf("StopTimer: %v", err)
	}
	d, err := s.TaskDuration(tk.ID)
	if err != nil {
		t.Fatalf("TaskDuration: %v", err)
	}
	if d != time.Hour {
		t.Fatalf("want 1h, got %s", d)
	}
}

func TestStopTimerNoRunningIsNoop(t *testing.T) {
	s := newTestStore(t)
	if err := s.StopTimer(); err != nil {
		t.Fatalf("StopTimer with none running should be nil, got %v", err)
	}
}
