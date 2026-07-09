package store

import (
	"testing"
	"time"
)

func TestDailySummary(t *testing.T) {
	s := newTestStore(t)
	s.now = advancingClock() // 9:00, 10:00, 11:00, 12:00 on 2026-07-09
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Report") // created_at 9:00
	s.StartTimer(tk.ID)                  // started 10:00
	s.StopTimer()                        // ended 11:00 => 1h
	s.SetTaskDone(tk.ID, true)           // done_at 12:00

	sum, err := s.DailySummary(time.Date(2026, 7, 9, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("DailySummary: %v", err)
	}
	if len(sum.Completed) != 1 || sum.Completed[0].ID != tk.ID {
		t.Fatalf("want 1 completed task, got %+v", sum.Completed)
	}
	if sum.Total != time.Hour {
		t.Fatalf("want total 1h, got %s", sum.Total)
	}
	if len(sum.Times) != 1 || sum.Times[0].Duration != time.Hour {
		t.Fatalf("want 1 timed task of 1h, got %+v", sum.Times)
	}
}

func TestDailySummaryExcludesOtherDays(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Yesterday")
	// Force done_at into the prior day.
	s.db.Exec(`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`,
		time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC), tk.ID)

	sum, err := s.DailySummary(time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("DailySummary: %v", err)
	}
	if len(sum.Completed) != 0 {
		t.Fatalf("want 0 completed today, got %d", len(sum.Completed))
	}
}
