// Package tui is the Bubble Tea terminal UI. It holds view state only and
// delegates every mutation to the store, reloading after each change.
package tui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"github.com/liam-dev-c/what-was-next/internal/store"
)

type screen int

const (
	screenTasks screen = iota
	screenHistory
	screenSettings
)

// period selects which window the summary screen shows.
type period int

const (
	periodDay period = iota
	periodWeek
)

// focusArea selects which panel receives list navigation keys on the tasks
// workspace.
type focusArea int

const (
	focusTasks focusArea = iota
	focusProjects
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
	input   textinput.Model

	// panel workspace state
	focus         focusArea
	addingProject bool // input is naming a new project, not a task
	notesEditing  bool
	notesArea     textarea.Model
	taskVP        viewport.Model
	detailVP      viewport.Model

	// project switcher cursor — populated in Task 8
	projCursor int

	// summary snapshots — populated in Task 10; week/period added later
	summary       store.DailySummary
	week          store.WeekSummary
	summaryPeriod period

	// settings — first day of the week, editable on the settings screen
	weekStart time.Weekday

	width  int
	height int
}

// storeTask is an alias so other files in this package can name store.Task
// without importing the store package a second time under a new name.
type storeTask = store.Task

func New(s *store.Store) (Model, error) {
	m := Model{store: s, screen: screenTasks}
	if err := m.reloadProjects(); err != nil {
		return Model{}, err
	}
	if err := m.reloadTasks(); err != nil {
		return Model{}, err
	}
	weekStart, err := s.WeekStart()
	if err != nil {
		return Model{}, fmt.Errorf("load settings: %w", err)
	}
	m.weekStart = weekStart
	// Tasks is the landing screen; prime the daily snapshot so History (opened
	// via 'h') shows Today's data immediately.
	m.loadSummary()
	m.taskVP = viewport.New()
	m.detailVP = viewport.New()
	m.notesArea = textarea.New()
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

// rightColumnHeights splits the right column (below the help row) into the
// Tasks panel (~40%) over the Details panel (~60%). Single source of truth so
// resizePanels and viewWorkspace never disagree.
func (m Model) rightColumnHeights() (tasksH, detailH int) {
	rightColH := m.height - 2 // reserve 2 rows for the help block (margin + text)
	if rightColH < 6 {
		rightColH = 6
	}
	detailH = rightColH * 3 / 5
	if detailH < 3 {
		detailH = 3
	}
	tasksH = rightColH - detailH
	if tasksH < 3 {
		tasksH = 3
	}
	return tasksH, detailH
}

// resizePanels lays out the three panels from the current terminal size.
// Left: Projects (fixed). Right column: Tasks over Details, each a bordered
// panel whose inner viewport is (panel - 2 border - 1 title) tall.
func (m *Model) resizePanels() {
	rightW := m.width - projectsPanelWidth
	if rightW < 1 {
		rightW = 1
	}
	innerW := rightW - 2 // borders
	if innerW < 1 {
		innerW = 1
	}
	tasksPanelH, detailPanelH := m.rightColumnHeights()
	taskInner := tasksPanelH - 2 - 1 // borders + title
	if taskInner < 1 {
		taskInner = 1
	}
	detailInner := detailPanelH - 2 - 1
	if detailInner < 1 {
		detailInner = 1
	}
	m.taskVP.SetWidth(innerW)
	m.taskVP.SetHeight(taskInner)
	m.detailVP.SetWidth(innerW)
	m.detailVP.SetHeight(detailInner)
	m.notesArea.SetWidth(innerW)
	m.notesArea.SetHeight(detailInner)
}

type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// elapsedFor returns total tracked time for a task, including the live segment
// if its timer is currently running.
func (m Model) elapsedFor(taskID int64) (time.Duration, bool) {
	closed, err := m.store.TaskDuration(taskID)
	if err != nil {
		return 0, false
	}
	total := closed
	running, err := m.store.RunningEntry()
	if err == nil && running != nil && running.TaskID == taskID {
		total += time.Since(running.StartedAt)
	}
	if total == 0 && closed == 0 {
		// Distinguish "no time at all" from "exactly zero": ok only if entries exist.
		return 0, running != nil && running.TaskID == taskID
	}
	return total, true
}

func (m Model) Init() tea.Cmd { return tickCmd() }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		return m, tickCmd()
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizePanels()
		return m, nil
	case tea.KeyPressMsg:
		// Global quit (only when not typing in an input; Task 7 guards this).
		if !m.editing && !m.notesEditing && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}
		switch m.screen {
		case screenTasks:
			return m.updateTasks(msg)
		case screenHistory:
			return m.updateSummary(msg)
		case screenSettings:
			return m.updateSettings(msg)
		}
	}
	return m, nil
}

func (m Model) View() tea.View {
	var content string
	switch m.screen {
	case screenHistory:
		content = m.viewSummary()
	case screenSettings:
		content = m.viewSettings()
	default:
		content = m.viewTasks()
	}
	v := tea.NewView(content)
	// v2 replaces the tea.WithAltScreen program option with a per-view field.
	v.AltScreen = true
	return v
}

func (m *Model) setStatus(err error) {
	if err != nil {
		m.status = err.Error()
	}
}
