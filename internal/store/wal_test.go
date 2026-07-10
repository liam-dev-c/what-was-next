package store

import (
	"path/filepath"
	"testing"
)

func TestOpenEnablesWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var mode string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want \"wal\"", mode)
	}
}

func TestOpenSetsBusyTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "busy.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	var ms int
	if err := s.db.QueryRow(`PRAGMA busy_timeout`).Scan(&ms); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if ms != 5000 {
		t.Fatalf("busy_timeout = %d, want 5000", ms)
	}
}
