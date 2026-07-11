package store

import (
	"database/sql"
	"fmt"
	"strings"
)

// SetTaskTags replaces the full set of tags on a task. Passing an empty slice
// clears all tags. Tag names are trimmed, de-duplicated case-insensitively, and
// created on demand; names differing only in case collapse to the first seen.
// Returns ErrNotFound if the task does not exist.
func (s *Store) SetTaskTags(taskID int64, tags []string) error {
	var one int
	if err := s.db.QueryRow(`SELECT 1 FROM tasks WHERE id = ?`, taskID).Scan(&one); err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return fmt.Errorf("lookup task: %w", err)
	}
	clean := normalizeTags(tags)

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("set tags begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM task_tags WHERE task_id = ?`, taskID); err != nil {
		return fmt.Errorf("clear task tags: %w", err)
	}
	for _, name := range clean {
		if _, err := tx.Exec(
			`INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, name,
		); err != nil {
			return fmt.Errorf("insert tag: %w", err)
		}
		var tagID int64
		if err := tx.QueryRow(`SELECT id FROM tags WHERE name = ?`, name).Scan(&tagID); err != nil {
			return fmt.Errorf("lookup tag id: %w", err)
		}
		if _, err := tx.Exec(
			`INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?)`, taskID, tagID,
		); err != nil {
			return fmt.Errorf("link tag: %w", err)
		}
	}
	return tx.Commit()
}

// ListTags returns the distinct tag names currently applied to at least one
// task, sorted case-insensitively. Tags no longer used by any task are omitted.
func (s *Store) ListTags() ([]string, error) {
	rows, err := s.db.Query(
		`SELECT DISTINCT tg.name FROM tags tg
		 JOIN task_tags tt ON tt.tag_id = tg.id
		 ORDER BY tg.name COLLATE NOCASE`,
	)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// attachTags loads the tags for every task in the given project and sets each
// task's Tags field. Callers must ensure no other rows are open on the store's
// single connection when calling this.
func (s *Store) attachTags(projectID int64, tasks []Task) error {
	if len(tasks) == 0 {
		return nil
	}
	rows, err := s.db.Query(
		`SELECT tt.task_id, tg.name
		 FROM task_tags tt
		 JOIN tags tg ON tg.id = tt.tag_id
		 JOIN tasks t ON t.id = tt.task_id
		 WHERE t.project_id = ?
		 ORDER BY tg.name COLLATE NOCASE`,
		projectID,
	)
	if err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	defer rows.Close()
	byTask := make(map[int64][]string)
	for rows.Next() {
		var taskID int64
		var name string
		if err := rows.Scan(&taskID, &name); err != nil {
			return fmt.Errorf("scan task tag: %w", err)
		}
		byTask[taskID] = append(byTask[taskID], name)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("load tags: %w", err)
	}
	for i := range tasks {
		tasks[i].Tags = byTask[tasks[i].ID]
	}
	return nil
}

// normalizeTags trims whitespace, drops empties, and de-duplicates
// case-insensitively while preserving the first-seen spelling and order.
func normalizeTags(tags []string) []string {
	var out []string
	seen := make(map[string]bool)
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, t)
	}
	return out
}
