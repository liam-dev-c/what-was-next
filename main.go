// Command what-was-next is a terminal task manager and time tracker.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"
	"github.com/liam-dev-c/what-was-next/internal/store"
	"github.com/liam-dev-c/what-was-next/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "what-was-next:", err)
		os.Exit(1)
	}
}

func run() error {
	path, err := dbPath()
	if err != nil {
		return err
	}
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()

	model, err := tui.New(s)
	if err != nil {
		return err
	}
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}

// dbPath resolves ~/.config/what-was-next/what-was-next.db, honoring
// XDG_CONFIG_HOME, and ensures the directory exists.
func dbPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "what-was-next")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return filepath.Join(dir, "what-was-next.db"), nil
}
