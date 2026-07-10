package store

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"
)

// settingWeekStart is the key under which the first-day-of-week preference is
// stored, as the integer form of a time.Weekday (0=Sunday .. 6=Saturday).
const settingWeekStart = "week_start"

// GetSetting returns the stored value for key. ok is false when the key is
// unset, letting callers fall back to a default without treating it as an error.
func (s *Store) GetSetting(key string) (value string, ok bool, err error) {
	err = s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get setting %q: %w", key, err)
	}
	return value, true, nil
}

// SetSetting upserts the value for key.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value`,
		key, value,
	)
	if err != nil {
		return fmt.Errorf("set setting %q: %w", key, err)
	}
	return nil
}

// WeekStart returns the configured first day of the week, defaulting to Monday
// when unset or unparseable.
func (s *Store) WeekStart() (time.Weekday, error) {
	v, ok, err := s.GetSetting(settingWeekStart)
	if err != nil {
		return time.Monday, err
	}
	if !ok {
		return time.Monday, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 || n > 6 {
		return time.Monday, nil
	}
	return time.Weekday(n), nil
}

// SetWeekStart persists the first day of the week (time.Sunday .. time.Saturday).
func (s *Store) SetWeekStart(d time.Weekday) error {
	return s.SetSetting(settingWeekStart, strconv.Itoa(int(d)))
}
