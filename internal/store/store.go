// Package store is the SQLite persistence layer and domain model for
// what-was-next. It owns all SQL; nothing above it touches database/sql.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Project struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type Task struct {
	ID        int64
	ProjectID int64
	Title     string
	Notes     string
	Done      bool
	SortOrder int64
	CreatedAt time.Time
	DoneAt    *time.Time
}

type TimeEntry struct {
	ID        int64
	TaskID    int64
	StartedAt time.Time
	EndedAt   *time.Time
}

// Store wraps the database. now is injectable so tests are deterministic.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

const schema = `
CREATE TABLE IF NOT EXISTS projects (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS tasks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	title      TEXT NOT NULL,
	notes      TEXT NOT NULL DEFAULT '',
	done       INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL,
	done_at    TIMESTAMP
);
CREATE TABLE IF NOT EXISTS time_entries (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	started_at TIMESTAMP NOT NULL,
	ended_at   TIMESTAMP
);
CREATE TABLE IF NOT EXISTS settings (
	key   TEXT PRIMARY KEY,
	value TEXT NOT NULL
);
`

// Open opens (creating if needed) the SQLite DB at path, runs migrations,
// and seeds a default "Inbox" project when none exist. path may be ":memory:".
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	db.SetMaxOpenConns(1)
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	s := &Store{db: db, now: func() time.Time { return time.Now().UTC() }}
	if err := s.seedDefaultProject(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) seedDefaultProject() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n); err != nil {
		return fmt.Errorf("count projects: %w", err)
	}
	if n > 0 {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO projects (name, created_at) VALUES (?, ?)`,
		"Inbox", s.now(),
	)
	if err != nil {
		return fmt.Errorf("seed default project: %w", err)
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }
