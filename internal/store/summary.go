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

func (s *Store) DailySummary(day time.Time) (DailySummary, error) {
	day = day.UTC()
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	sum := DailySummary{Day: start}

	// Completed tasks with done_at within the day.
	rows, err := s.db.Query(
		`SELECT id, project_id, title, notes, done, sort_order, created_at, done_at
		 FROM tasks
		 WHERE done = 1 AND done_at >= ? AND done_at < ?
		 ORDER BY done_at`, start, end,
	)
	if err != nil {
		return sum, fmt.Errorf("summary completed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var t Task
		var doneAt time.Time
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt,
		); err != nil {
			return sum, fmt.Errorf("scan completed: %w", err)
		}
		t.DoneAt = &doneAt
		sum.Completed = append(sum.Completed, t)
	}
	if err := rows.Err(); err != nil {
		return sum, err
	}

	// Per-task time from closed entries started within the day. Durations are
	// summed in Go by scanning started_at/ended_at as time.Time (matching
	// TaskDuration in timelog.go). This deliberately avoids SQLite date
	// functions: the driver stores time.Time in Go's t.String() format
	// ("2006-01-02 15:04:05 -0700 MST"), which julianday() cannot parse.
	trows, err := s.db.Query(
		`SELECT t.id, t.project_id, t.title, t.notes, t.done, t.sort_order,
		        t.created_at, t.done_at, e.started_at, e.ended_at
		 FROM time_entries e
		 JOIN tasks t ON t.id = e.task_id
		 WHERE e.ended_at IS NOT NULL AND e.started_at >= ? AND e.started_at < ?
		 ORDER BY t.id`, start, end,
	)
	if err != nil {
		return sum, fmt.Errorf("summary times: %w", err)
	}
	defer trows.Close()

	type taskAgg struct {
		task     Task
		duration time.Duration
	}
	order := []int64{}
	byTask := map[int64]*taskAgg{}
	for trows.Next() {
		var t Task
		var doneAt *time.Time
		var started, ended time.Time
		if err := trows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt, &started, &ended,
		); err != nil {
			return sum, fmt.Errorf("scan times: %w", err)
		}
		t.DoneAt = doneAt
		a, ok := byTask[t.ID]
		if !ok {
			a = &taskAgg{task: t}
			byTask[t.ID] = a
			order = append(order, t.ID)
		}
		a.duration += ended.Sub(started)
	}
	if err := trows.Err(); err != nil {
		return sum, err
	}
	for _, id := range order {
		a := byTask[id]
		sum.Times = append(sum.Times, TaskDuration{Task: a.task, Duration: a.duration})
		sum.Total += a.duration
	}
	// Order by duration descending (as the prior SQL did with ORDER BY secs DESC).
	sort.SliceStable(sum.Times, func(i, j int) bool {
		return sum.Times[i].Duration > sum.Times[j].Duration
	})
	return sum, nil
}
