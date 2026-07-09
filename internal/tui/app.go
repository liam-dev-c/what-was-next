// Package tui is the Bubble Tea terminal UI. It holds view state only and
// delegates every mutation to the store, reloading after each change.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/liam-dev-c/what-was-next/internal/store"
)

type screen int

const (
	screenTasks screen = iota
	screenProjects
	screenSummary
)

// Model is the root Bubble Tea model. Screen-specific state is added by the
// task/project/timer/summary tasks; Update/View dispatch on m.screen.
type Model struct {
	store    *store.Store
	screen   screen
	projects []store.Project
	active   int // index into projects

	tasks  []store.Task
	cursor int // selected task index on the tasks screen
	status string

	// input state (task add/edit) — populated in Task 7
	editing bool
	editID  int64 // 0 == adding a new task
	input   textInput

	// project switcher cursor — populated in Task 8
	projCursor int

	// summary snapshot — populated in Task 10
	summary store.DailySummary

	width  int
	height int
}

// textInput is a thin alias so app.go compiles before Task 7 wires the real
// bubbles/textinput; Task 7 replaces this with textinput.Model.
type textInput = struct{ Value string }

func New(s *store.Store) (Model, error) {
	m := Model{store: s, screen: screenTasks}
	if err := m.reloadProjects(); err != nil {
		return Model{}, err
	}
	if err := m.reloadTasks(); err != nil {
		return Model{}, err
	}
	return m, nil
}

func (m *Model) reloadProjects() error {
	projects, err := m.store.ListProjects()
	if err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	m.projects = projects
	if m.active >= len(m.projects) {
		m.active = 0
	}
	return nil
}

func (m *Model) reloadTasks() error {
	if len(m.projects) == 0 {
		m.tasks = nil
		return nil
	}
	tasks, err := m.store.ListTasks(m.activeProject().ID)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	m.tasks = tasks
	if m.cursor >= len(m.tasks) {
		m.cursor = max(0, len(m.tasks)-1)
	}
	return nil
}

func (m Model) activeProject() store.Project {
	if len(m.projects) == 0 {
		return store.Project{}
	}
	return m.projects[m.active]
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		// Global quit (only when not typing in an input; Task 7 guards this).
		if !m.editing && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}
		switch m.screen {
		case screenTasks:
			return m.updateTasks(msg)
		case screenProjects:
			return m.updateProjects(msg)
		case screenSummary:
			return m.updateSummary(msg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenProjects:
		return m.viewProjects()
	case screenSummary:
		return m.viewSummary()
	default:
		return m.viewTasks()
	}
}
