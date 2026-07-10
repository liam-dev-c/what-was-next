// Command what-was-next is a terminal task manager and time tracker.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/liam-dev-c/what-was-next/internal/mcpserver"
	"github.com/liam-dev-c/what-was-next/internal/store"
	"github.com/liam-dev-c/what-was-next/internal/tui"
)

type cmd int

const (
	cmdTUI cmd = iota
	cmdMCPServe
	cmdMCPInstall
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "what-was-next:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch command(args) {
	case cmdMCPInstall:
		return mcpserver.Install(scopeFlag(args))
	case cmdMCPServe:
		return runMCPServe()
	default:
		return runTUI()
	}
}

// command classifies CLI args into a subcommand.
func command(args []string) cmd {
	if len(args) >= 1 && args[0] == "mcp" {
		if len(args) >= 2 && args[1] == "install" {
			return cmdMCPInstall
		}
		return cmdMCPServe
	}
	return cmdTUI
}

// scopeFlag reads "--scope <value>" from args, defaulting to "user".
func scopeFlag(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--scope" {
			return args[i+1]
		}
	}
	return "user"
}

func runTUI() error {
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

func runMCPServe() error {
	path, err := dbPath()
	if err != nil {
		return err
	}
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()
	return mcpserver.Serve(context.Background(), s)
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
