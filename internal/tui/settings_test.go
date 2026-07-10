package tui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
)

func TestSettingsTogglePersistsWeekStart(t *testing.T) {
	m := newModel(t)
	if m.weekStart != time.Monday {
		t.Fatalf("want default week start Monday, got %s", m.weekStart)
	}
	m.screen = screenSettings

	mi, _ := m.updateSettings(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	if m.weekStart != time.Sunday {
		t.Fatalf("want Sunday after toggle, got %s", m.weekStart)
	}

	got, err := m.store.WeekStart()
	if err != nil {
		t.Fatalf("WeekStart: %v", err)
	}
	if got != time.Sunday {
		t.Fatalf("toggle should persist to the store, got %s", got)
	}
}

func TestSettingsEscReturnsToHistory(t *testing.T) {
	m := newModel(t)
	m.screen = screenSettings
	mi, _ := m.updateSettings(tea.KeyPressMsg{Code: tea.KeyEscape})
	if mi.(Model).screen != screenHistory {
		t.Fatal("want return to history on esc")
	}
}
