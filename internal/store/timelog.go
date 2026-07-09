package store

import (
	"database/sql"
	"fmt"
	"time"
)

// StartTimer stops any running entry and starts a new one for taskID,
// atomically, so at most one timer is ever running.
func (s *Store) StartTimer(taskID int64) (TimeEntry, error) {
	now := s.now()
	tx, err := s.db.Begin()
	if err != nil {
		return TimeEntry{}, fmt.Errorf("start timer begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE time_entries SET ended_at = ? WHERE ended_at IS NULL`, now,
	); err != nil {
		return TimeEntry{}, fmt.Errorf("start timer stop previous: %w", err)
	}
	res, err := tx.Exec(
		`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, NULL)`,
		taskID, now,
	)
	if err != nil {
		return TimeEntry{}, fmt.Errorf("start timer insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return TimeEntry{}, fmt.Errorf("start timer id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return TimeEntry{}, fmt.Errorf("start timer commit: %w", err)
	}
	return TimeEntry{ID: id, TaskID: taskID, StartedAt: now}, nil
}

func (s *Store) StopTimer() error {
	_, err := s.db.Exec(
		`UPDATE time_entries SET ended_at = ? WHERE ended_at IS NULL`, s.now(),
	)
	if err != nil {
		return fmt.Errorf("stop timer: %w", err)
	}
	return nil
}

func (s *Store) RunningEntry() (*TimeEntry, error) {
	var e TimeEntry
	err := s.db.QueryRow(
		`SELECT id, task_id, started_at FROM time_entries
		 WHERE ended_at IS NULL ORDER BY started_at DESC LIMIT 1`,
	).Scan(&e.ID, &e.TaskID, &e.StartedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("running entry: %w", err)
	}
	return &e, nil
}

func (s *Store) TaskDuration(taskID int64) (time.Duration, error) {
	rows, err := s.db.Query(
		`SELECT started_at, ended_at FROM time_entries
		 WHERE task_id = ? AND ended_at IS NOT NULL`, taskID,
	)
	if err != nil {
		return 0, fmt.Errorf("task duration: %w", err)
	}
	defer rows.Close()
	var total time.Duration
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return 0, fmt.Errorf("scan duration: %w", err)
		}
		total += end.Sub(start)
	}
	return total, rows.Err()
}
