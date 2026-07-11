package store

import (
	"database/sql"
	"fmt"
)

func (s *Store) CreateTask(projectID int64, title string) (Task, error) {
	now := s.now()
	var nextOrder int64
	err := s.db.QueryRow(
		`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM tasks WHERE project_id = ?`,
		projectID,
	).Scan(&nextOrder)
	if err != nil {
		return Task{}, fmt.Errorf("next sort order: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO tasks (project_id, title, notes, done, sort_order, created_at)
		 VALUES (?, ?, '', 0, ?, ?)`,
		projectID, title, nextOrder, now,
	)
	if err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("create task id: %w", err)
	}
	return Task{
		ID: id, ProjectID: projectID, Title: title,
		SortOrder: nextOrder, CreatedAt: now,
	}, nil
}

func (s *Store) ListTasks(projectID int64) ([]Task, error) {
	rows, err := s.db.Query(
		`SELECT id, project_id, title, notes, done, sort_order, created_at, done_at
		 FROM tasks WHERE project_id = ? ORDER BY sort_order`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	var out []Task
	for rows.Next() {
		var t Task
		var doneAt sql.NullTime
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan task: %w", err)
		}
		if doneAt.Valid {
			dt := doneAt.Time
			t.DoneAt = &dt
		}
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	// Close before the tags query: the store caps open connections at 1, so a
	// nested query while these rows are open would deadlock.
	rows.Close()
	if err := s.attachTags(projectID, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) UpdateTask(id int64, title, notes string) error {
	res, err := s.db.Exec(
		`UPDATE tasks SET title = ?, notes = ? WHERE id = ?`, title, notes, id,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("update task rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) SetTaskDone(id int64, done bool) error {
	if done {
		res, err := s.db.Exec(
			`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`, s.now(), id,
		)
		if err != nil {
			return fmt.Errorf("set task done: %w", err)
		}
		n, err := res.RowsAffected()
		if err != nil {
			return fmt.Errorf("set task done rows: %w", err)
		}
		if n == 0 {
			return ErrNotFound
		}
		return nil
	}
	res, err := s.db.Exec(
		`UPDATE tasks SET done = 0, done_at = NULL WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("set task undone: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("set task undone rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteTask(id int64) error {
	res, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete task rows: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// MoveTask swaps the sort_order of the task with its neighbor in the same
// project. delta = -1 moves up (toward the top), +1 moves down. No-op at ends.
func (s *Store) MoveTask(id int64, delta int) error {
	if delta != -1 && delta != 1 {
		return fmt.Errorf("move task: delta must be -1 or 1, got %d", delta)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("move task begin: %w", err)
	}
	defer tx.Rollback()

	var projectID, sortOrder int64
	err = tx.QueryRow(
		`SELECT project_id, sort_order FROM tasks WHERE id = ?`, id,
	).Scan(&projectID, &sortOrder)
	if err != nil {
		return fmt.Errorf("move task lookup: %w", err)
	}

	// Find the adjacent task in the move direction.
	var neighborID, neighborOrder int64
	var q string
	if delta < 0 {
		q = `SELECT id, sort_order FROM tasks
		     WHERE project_id = ? AND sort_order < ?
		     ORDER BY sort_order DESC LIMIT 1`
	} else {
		q = `SELECT id, sort_order FROM tasks
		     WHERE project_id = ? AND sort_order > ?
		     ORDER BY sort_order ASC LIMIT 1`
	}
	err = tx.QueryRow(q, projectID, sortOrder).Scan(&neighborID, &neighborOrder)
	if err == sql.ErrNoRows {
		return nil // at an end; nothing to swap
	}
	if err != nil {
		return fmt.Errorf("move task neighbor: %w", err)
	}

	if _, err := tx.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, neighborOrder, id); err != nil {
		return fmt.Errorf("move task swap self: %w", err)
	}
	if _, err := tx.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, sortOrder, neighborID); err != nil {
		return fmt.Errorf("move task swap neighbor: %w", err)
	}
	return tx.Commit()
}
