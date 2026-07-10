package store

import (
	"fmt"
	"sort"
	"time"
)

type TaskDuration struct {
	Task     Task
	Duration time.Duration
}

type DailySummary struct {
	Day       time.Time
	Completed []Task
	Times     []TaskDuration
	Total     time.Duration
}

// DayTotal is the time tracked on a single calendar day of a week.
type DayTotal struct {
	Day      time.Time
	Duration time.Duration
}

// WeekSummary aggregates a calendar week. Days holds one bucket per day from
// Start through End inclusive, in order, so callers can render a breakdown.
type WeekSummary struct {
	Start     time.Time // first day of the week (00:00 local)
	End       time.Time // last day of the week (00:00 local), i.e. Start + 6 days
	Completed []Task
	Times     []TaskDuration
	Total     time.Duration
	Days      []DayTotal
}

func (s *Store) DailySummary(day time.Time) (DailySummary, error) {
	// The day window spans the calendar day of the given time in its own
	// location, so "today" follows the user's timezone. loadSummary passes
	// time.Now(), which carries the machine's local zone.
	loc := day.Location()
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	end := start.Add(24 * time.Hour)

	sum := DailySummary{Day: start}
	completed, err := s.completedInRange(start.UTC(), end.UTC())
	if err != nil {
		return sum, err
	}
	sum.Completed = completed

	entries, err := s.closedEntriesInRange(start.UTC(), end.UTC())
	if err != nil {
		return sum, err
	}
	sum.Times, sum.Total = aggregateByTask(entries)
	return sum, nil
}

// WeeklySummary aggregates the calendar week containing day, where the week
// begins on weekStart (e.g. time.Monday). The window follows day's timezone.
func (s *Store) WeeklySummary(day time.Time, weekStart time.Weekday) (WeekSummary, error) {
	loc := day.Location()
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, loc)
	offset := (int(dayStart.Weekday()) - int(weekStart) + 7) % 7
	start := dayStart.AddDate(0, 0, -offset)
	end := start.AddDate(0, 0, 7)

	ws := WeekSummary{Start: start, End: start.AddDate(0, 0, 6)}

	completed, err := s.completedInRange(start.UTC(), end.UTC())
	if err != nil {
		return ws, err
	}
	ws.Completed = completed

	entries, err := s.closedEntriesInRange(start.UTC(), end.UTC())
	if err != nil {
		return ws, err
	}
	ws.Times, ws.Total = aggregateByTask(entries)

	// Per-day buckets. Each day's window is [Day, Day+1) in local time, which
	// is robust across DST transitions (AddDate keeps calendar-day boundaries).
	ws.Days = make([]DayTotal, 7)
	for i := range ws.Days {
		ws.Days[i] = DayTotal{Day: start.AddDate(0, 0, i)}
	}
	for _, e := range entries {
		local := e.started.In(loc)
		for i := range ws.Days {
			lo := ws.Days[i].Day
			hi := lo.AddDate(0, 0, 1)
			if !local.Before(lo) && local.Before(hi) {
				ws.Days[i].Duration += e.ended.Sub(e.started)
				break
			}
		}
	}
	return ws, nil
}

// completedInRange returns tasks whose done_at falls within [startUTC, endUTC).
func (s *Store) completedInRange(startUTC, endUTC time.Time) ([]Task, error) {
	rows, err := s.db.Query(
		`SELECT id, project_id, title, notes, done, sort_order, created_at, done_at
		 FROM tasks
		 WHERE done = 1 AND done_at >= ? AND done_at < ?
		 ORDER BY done_at`, startUTC, endUTC,
	)
	if err != nil {
		return nil, fmt.Errorf("summary completed: %w", err)
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task
		var doneAt time.Time
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt,
		); err != nil {
			return nil, fmt.Errorf("scan completed: %w", err)
		}
		t.DoneAt = &doneAt
		out = append(out, t)
	}
	return out, rows.Err()
}

type closedEntry struct {
	task    Task
	started time.Time
	ended   time.Time
}

// closedEntriesInRange returns closed time entries started within
// [startUTC, endUTC), each joined to its task.
//
// This deliberately avoids SQLite date functions: modernc.org/sqlite stores
// time.Time as RFC3339Nano, whose 9-digit fractional seconds exceed SQLite's
// millisecond precision, so julianday()/date() return NULL on these values.
// Durations are therefore summed in Go from scanned time.Time bounds.
func (s *Store) closedEntriesInRange(startUTC, endUTC time.Time) ([]closedEntry, error) {
	rows, err := s.db.Query(
		`SELECT t.id, t.project_id, t.title, t.notes, t.done, t.sort_order,
		        t.created_at, t.done_at, e.started_at, e.ended_at
		 FROM time_entries e
		 JOIN tasks t ON t.id = e.task_id
		 WHERE e.ended_at IS NOT NULL AND e.started_at >= ? AND e.started_at < ?
		 ORDER BY t.id`, startUTC, endUTC,
	)
	if err != nil {
		return nil, fmt.Errorf("summary times: %w", err)
	}
	defer rows.Close()
	var out []closedEntry
	for rows.Next() {
		var t Task
		var doneAt *time.Time
		var started, ended time.Time
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt, &started, &ended,
		); err != nil {
			return nil, fmt.Errorf("scan times: %w", err)
		}
		t.DoneAt = doneAt
		out = append(out, closedEntry{task: t, started: started, ended: ended})
	}
	return out, rows.Err()
}

// aggregateByTask sums entry durations per task, returning them sorted by
// duration descending along with the grand total.
func aggregateByTask(entries []closedEntry) ([]TaskDuration, time.Duration) {
	order := []int64{}
	byTask := map[int64]*TaskDuration{}
	for _, e := range entries {
		td, ok := byTask[e.task.ID]
		if !ok {
			td = &TaskDuration{Task: e.task}
			byTask[e.task.ID] = td
			order = append(order, e.task.ID)
		}
		td.Duration += e.ended.Sub(e.started)
	}
	var times []TaskDuration
	var total time.Duration
	for _, id := range order {
		times = append(times, *byTask[id])
		total += byTask[id].Duration
	}
	sort.SliceStable(times, func(i, j int) bool {
		return times[i].Duration > times[j].Duration
	})
	return times, total
}
