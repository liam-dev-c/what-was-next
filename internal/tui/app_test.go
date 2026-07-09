package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/liam-dev-c/what-was-next/internal/store"
)

func newModel(t *testing.T) Model {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	m, err := New(s)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestNewSelectsDefaultProject(t *testing.T) {
	m := newModel(t)
	if m.activeProject().Name != "Inbox" {
		t.Fatalf("want active project 'Inbox', got %q", m.activeProject().Name)
	}
	if m.screen != screenTasks {
		t.Fatalf("want initial screen screenTasks, got %v", m.screen)
	}
}

func TestQuitKey(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("want quit command on 'q', got nil")
	}
}
