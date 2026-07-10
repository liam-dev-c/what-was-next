package store

import (
	"testing"
	"time"
)

func TestDailySummaryUsesDayArgumentTimezone(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Late")
	// Completed at 2026-07-09T23:30:00Z. In a +10:00 zone that instant is
	// 2026-07-10 09:30 local, so it belongs to the Jul-10 *local* day.
	doneUTC := time.Date(2026, 7, 9, 23, 30, 0, 0, time.UTC)
	s.db.Exec(`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`, doneUTC, tk.ID)

	east := time.FixedZone("UTC+10", 10*3600)
	jul10 := time.Date(2026, 7, 10, 12, 0, 0, 0, east)
	sum, err := s.DailySummary(jul10)
	if err != nil {
		t.Fatalf("DailySummary jul10: %v", err)
	}
	if len(sum.Completed) != 1 {
		t.Fatalf("want task in Jul-10 local day, got %d completed", len(sum.Completed))
	}

	// The Jul-9 local day (in +10) must NOT contain it.
	jul9 := time.Date(2026, 7, 9, 12, 0, 0, 0, east)
	sum9, err := s.DailySummary(jul9)
	if err != nil {
		t.Fatalf("DailySummary jul9: %v", err)
	}
	if len(sum9.Completed) != 0 {
		t.Fatalf("want task excluded from Jul-9 local day, got %d", len(sum9.Completed))
	}
}

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

func TestWeeklySummary(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)

	// Completed within the week: Wed Jul 8.
	done, _ := s.CreateTask(pid, "Shipped")
	s.db.Exec(`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`,
		time.Date(2026, 7, 8, 9, 0, 0, 0, time.UTC), done.ID)

	// Time tracked: 1h Wed Jul 8, 30m Fri Jul 10.
	timed, _ := s.CreateTask(pid, "Coding")
	s.db.Exec(`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, ?)`,
		timed.ID, time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC), time.Date(2026, 7, 8, 11, 0, 0, 0, time.UTC))
	s.db.Exec(`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, ?)`,
		timed.ID, time.Date(2026, 7, 10, 14, 0, 0, 0, time.UTC), time.Date(2026, 7, 10, 14, 30, 0, 0, time.UTC))

	// Outside the week (Sun Jul 5) — must be excluded.
	old, _ := s.CreateTask(pid, "LastWeek")
	s.db.Exec(`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`,
		time.Date(2026, 7, 5, 9, 0, 0, 0, time.UTC), old.ID)
	s.db.Exec(`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, ?)`,
		old.ID, time.Date(2026, 7, 5, 10, 0, 0, 0, time.UTC), time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC))

	// Thursday Jul 9, Monday-start week => Mon Jul 6 .. Sun Jul 12.
	ws, err := s.WeeklySummary(time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC), time.Monday)
	if err != nil {
		t.Fatalf("WeeklySummary: %v", err)
	}
	if !ws.Start.Equal(time.Date(2026, 7, 6, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("want week start Mon Jul 6, got %s", ws.Start)
	}
	if !ws.End.Equal(time.Date(2026, 7, 12, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("want week end Sun Jul 12, got %s", ws.End)
	}
	if len(ws.Completed) != 1 || ws.Completed[0].ID != done.ID {
		t.Fatalf("want only the in-week completed task, got %+v", ws.Completed)
	}
	if ws.Total != 90*time.Minute {
		t.Fatalf("want total 1h30m, got %s", ws.Total)
	}
	if len(ws.Days) != 7 {
		t.Fatalf("want 7 day buckets, got %d", len(ws.Days))
	}
	// index 2 = Wed Jul 8, index 4 = Fri Jul 10.
	if ws.Days[2].Duration != time.Hour {
		t.Fatalf("want Wed 1h, got %s", ws.Days[2].Duration)
	}
	if ws.Days[4].Duration != 30*time.Minute {
		t.Fatalf("want Fri 30m, got %s", ws.Days[4].Duration)
	}
	if ws.Days[0].Day.Weekday() != time.Monday {
		t.Fatalf("want first bucket Monday, got %s", ws.Days[0].Day.Weekday())
	}
}

func TestWeeklySummarySundayStart(t *testing.T) {
	s := newTestStore(t)
	// Thursday Jul 9 with a Sunday-start week => Sun Jul 5 .. Sat Jul 11.
	ws, err := s.WeeklySummary(time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC), time.Sunday)
	if err != nil {
		t.Fatalf("WeeklySummary: %v", err)
	}
	if !ws.Start.Equal(time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("want week start Sun Jul 5, got %s", ws.Start)
	}
	if ws.Days[0].Day.Weekday() != time.Sunday {
		t.Fatalf("want first bucket Sunday, got %s", ws.Days[0].Day.Weekday())
	}
}

func TestDailySummaryBucketsSubSecondBoundary(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Edge")
	// Completed just after midnight UTC, with a sub-second fraction.
	justAfterMidnight := time.Date(2026, 7, 9, 0, 0, 0, 500_000_000, time.UTC)
	s.db.Exec(`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`, justAfterMidnight, tk.ID)
	// A closed entry started 00:00:00.5, ended 00:00:01.5 => 1s.
	s.db.Exec(`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, ?)`,
		tk.ID, justAfterMidnight, justAfterMidnight.Add(time.Second))

	sum, err := s.DailySummary(time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("DailySummary: %v", err)
	}
	if len(sum.Completed) != 1 {
		t.Fatalf("want the sub-second task in its own day, got %d completed", len(sum.Completed))
	}
	if sum.Total != time.Second {
		t.Fatalf("want 1s tracked, got %s", sum.Total)
	}
	// Prior day must exclude it.
	prev, err := s.DailySummary(time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("DailySummary prev: %v", err)
	}
	if len(prev.Completed) != 0 || prev.Total != 0 {
		t.Fatalf("prior day should exclude the 00:00:00.5 task, got %d completed / %s", len(prev.Completed), prev.Total)
	}
}
