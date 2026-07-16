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
	editing     bool
	editID      int64 // 0 == adding a new task
	input       textinput.Model
	taggingTask bool // input is editing a task's comma-separated tags

	// panel workspace state
	focus            focusArea
	addingProject    bool // input is naming a new project, not a task
	showAllCompleted bool // c: reveal completed tasks finished before today
	notesEditing     bool
	notesArea        textarea.Model
	taskVP           viewport.Model
	detailVP         viewport.Model

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

	// dataVersion is the last-seen SQLite PRAGMA data_version. The tick reloads
	// store-backed state whenever it changes, i.e. when another process (the MCP
	// server) has committed. See refreshIfChanged.
	dataVersion int64
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
	// Prime the change-detection baseline so the first tick doesn't reload for
	// changes that predate startup. Best-effort: on error we leave it at 0 and
	// the first tick simply reloads once.
	m.dataVersion, _ = s.DataVersion()
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
	if n := m.visibleCount(); m.cursor >= n {
		m.cursor = max(0, n-1)
	}
	return nil
}

// visibleCount is the number of task rows currently shown in the list (open
// tasks plus the visible slice of completed ones). The cursor indexes this set.
func (m Model) visibleCount() int {
	vis, _ := m.visibleTasks()
	return len(vis)
}

// refreshIfChanged reloads store-backed state when another process (e.g. the
// MCP server) has committed a change since the last tick. It is a no-op while
// the user is mid-edit so typing is never interrupted — the baseline is left
// stale so the next tick after the edit ends picks the change up.
func (m *Model) refreshIfChanged() {
	if m.editing || m.notesEditing || m.addingProject {
		return
	}
	v, err := m.store.DataVersion()
	if err != nil || v == m.dataVersion {
		return
	}
	m.dataVersion = v
	m.reloadPreservingSelection()
}

// reloadPreservingSelection reloads projects and tasks after an external change,
// keeping the active project and selected task fixed by id (their indices may
// have shifted) rather than snapping the cursor to a clamped position.
func (m *Model) reloadPreservingSelection() {
	activeProjectID := m.activeProject().ID
	var selectedTaskID int64
	if t, ok := m.selectedTask(); ok {
		selectedTaskID = t.ID
	}

	if err := m.reloadProjects(); err != nil {
		m.setStatus(err)
		return
	}
	if activeProjectID != 0 {
		for i, p := range m.projects {
			if p.ID == activeProjectID {
				m.active = i
				break
			}
		}
	}
	if m.projCursor >= len(m.projects) {
		m.projCursor = max(0, len(m.projects)-1)
	}

	if err := m.reloadTasks(); err != nil {
		m.setStatus(err)
		return
	}
	if selectedTaskID != 0 {
		vis, _ := m.visibleTasks()
		for i, t := range vis {
			if t.ID == selectedTaskID {
				m.cursor = i
				break
			}
		}
	}

	// Keep the history screen's snapshot current too.
	if m.screen == screenHistory {
		if m.summaryPeriod == periodWeek {
			m.loadWeek()
		} else {
			m.loadSummary()
		}
	}

	if m.width > 0 {
		m.syncTaskScroll()
	}
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
		m.refreshIfChanged()
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
