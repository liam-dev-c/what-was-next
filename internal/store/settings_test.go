package store

import (
	"testing"
	"time"
)

func TestWeekStartDefaultsToMonday(t *testing.T) {
	s := newTestStore(t)
	d, err := s.WeekStart()
	if err != nil {
		t.Fatalf("WeekStart: %v", err)
	}
	if d != time.Monday {
		t.Fatalf("want default Monday, got %s", d)
	}
}

func TestSetAndGetWeekStart(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetWeekStart(time.Sunday); err != nil {
		t.Fatalf("SetWeekStart: %v", err)
	}
	d, err := s.WeekStart()
	if err != nil {
		t.Fatalf("WeekStart: %v", err)
	}
	if d != time.Sunday {
		t.Fatalf("want Sunday after set, got %s", d)
	}
}

func TestSetSettingUpserts(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetSetting("k", "1"); err != nil {
		t.Fatalf("SetSetting: %v", err)
	}
	if err := s.SetSetting("k", "2"); err != nil {
		t.Fatalf("SetSetting overwrite: %v", err)
	}
	v, ok, err := s.GetSetting("k")
	if err != nil {
		t.Fatalf("GetSetting: %v", err)
	}
	if !ok || v != "2" {
		t.Fatalf("want latest value \"2\", got %q ok=%v", v, ok)
	}
}
