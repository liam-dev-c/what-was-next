package store

import (
	"fmt"
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

	// Per-task time from closed entries started within the day.
	// Note: substr(..., 1, 19) extracts the timestamp portion from RFC3339 format
	// (which Go's sqlite driver uses), allowing julianday() to parse it.
	// Round to nearest second to avoid floating point precision errors.
	trows, err := s.db.Query(
		`SELECT t.id, t.project_id, t.title, t.notes, t.done, t.sort_order,
		        t.created_at, t.done_at,
		        ROUND(SUM((julianday(substr(e.ended_at, 1, 19)) - julianday(substr(e.started_at, 1, 19))) * 86400.0)) AS secs
		 FROM time_entries e
		 JOIN tasks t ON t.id = e.task_id
		 WHERE e.ended_at IS NOT NULL AND e.started_at >= ? AND e.started_at < ?
		 GROUP BY t.id
		 ORDER BY secs DESC`, start, end,
	)
	if err != nil {
		return sum, fmt.Errorf("summary times: %w", err)
	}
	defer trows.Close()
	for trows.Next() {
		var t Task
		var doneAt *time.Time
		var secs *float64
		if err := trows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt, &secs,
		); err != nil {
			return sum, fmt.Errorf("scan times: %w", err)
		}
		t.DoneAt = doneAt
		if secs != nil {
			d := time.Duration(int64(*secs) * int64(time.Second))
			sum.Times = append(sum.Times, TaskDuration{Task: t, Duration: d})
			sum.Total += d
		}
	}
	return sum, trows.Err()
}
