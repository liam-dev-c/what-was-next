package store

import (
	"testing"
	"time"
)

// fixedClock returns a deterministic UTC time for tests.
func fixedClock() time.Time {
	return time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.now = fixedClock
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenAndClose(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestOpenSeedsDefaultProject(t *testing.T) {
	s := newTestStore(t)
	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("want 1 project after Open, got %d", len(projects))
	}
	if projects[0].Name != "Inbox" {
		t.Errorf("want project 'Inbox', got %q", projects[0].Name)
	}
}
