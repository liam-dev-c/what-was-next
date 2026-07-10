package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) updateTasks(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.notesEditing {
		return m.updateNotes(msg) // added in Task 6
	}
	if m.editing {
		return m.updateInput(msg)
	}
	m.status = ""
	switch msg.String() {
	case "tab":
		m.toggleFocus()
		return m, nil
	case "shift+tab":
		m.toggleFocus()
		return m, nil
	case "h":
		m.summaryPeriod = periodDay
		m.loadSummary()
		m.screen = screenHistory
		return m, nil
	case ",":
		m.screen = screenSettings
		return m, nil
	}
	if m.focus == focusProjects {
		return m.updateProjectsPanel(msg)
	}
	return m.updateTasksPanel(msg)
}

func (m *Model) toggleFocus() {
	if m.focus == focusTasks {
		m.focus = focusProjects
		m.projCursor = m.active
	} else {
		m.focus = focusTasks
	}
}

// updateProjectsPanel handles keys when the Projects panel is focused.
func (m Model) updateProjectsPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.projCursor < len(m.projects)-1 {
			m.projCursor++
		}
	case "k", "up":
		if m.projCursor > 0 {
			m.projCursor--
		}
	case "enter", "space":
		m.active = m.projCursor
		m.cursor = 0
		m.setStatus(m.reloadTasks())
		m.focus = focusTasks
		if m.width > 0 {
			m.syncTaskScroll()
		}
	case "a":
		m.beginInput(0, "", true)
		return m, textinput.Blink
	}
	return m, nil
}

// updateTasksPanel handles keys when the Tasks panel is focused.
func (m Model) updateTasksPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "a":
		m.beginInput(0, "", false)
		return m, textinput.Blink
	case "e":
		if t, ok := m.selectedTask(); ok {
			m.beginInput(t.ID, t.Title, false)
			return m, textinput.Blink
		}
	case "n":
		return m.beginNotes() // added in Task 6
	case "enter", "space":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.SetTaskDone(t.ID, !t.Done))
			m.setStatus(m.reloadTasks())
		}
	case "d":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.DeleteTask(t.ID))
			m.setStatus(m.reloadTasks())
		}
	case "J":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.MoveTask(t.ID, 1))
			m.setStatus(m.reloadTasks())
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		}
	case "K":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.MoveTask(t.ID, -1))
			m.setStatus(m.reloadTasks())
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case "t":
		if t, ok := m.selectedTask(); ok {
			m.toggleTimer(t.ID)
		}
	case "p":
		m.focus = focusProjects
		m.projCursor = m.active
	}
	m.syncTaskScroll()
	return m, nil
}

// syncTaskScroll refreshes the task viewport content and scrolls so the
// selected row (m.cursor) stays within the visible window.
func (m *Model) syncTaskScroll() {
	m.taskVP.SetContent(m.taskListBody())
	h := m.taskVP.Height()
	if h < 1 {
		return
	}
	top := m.taskVP.YOffset()
	if m.cursor < top {
		m.taskVP.SetYOffset(m.cursor)
	} else if m.cursor >= top+h {
		m.taskVP.SetYOffset(m.cursor - h + 1)
	}
}

func (m *Model) toggleTimer(taskID int64) {
	running, err := m.store.RunningEntry()
	if err != nil {
		m.setStatus(err)
		return
	}
	if running != nil && running.TaskID == taskID {
		m.setStatus(m.store.StopTimer())
		return
	}
	_, err = m.store.StartTimer(taskID)
	m.setStatus(err)
}

func (m *Model) beginInput(id int64, initial string, project bool) {
	ti := textinput.New()
	ti.SetValue(initial)
	ti.Focus()
	ti.CursorEnd()
	m.input = ti
	m.editing = true
	m.editID = id
	m.addingProject = project
}

func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			if m.addingProject {
				_, err := m.store.CreateProject(val)
				m.setStatus(err)
				m.setStatus(m.reloadProjects())
			} else if m.editID == 0 {
				_, err := m.store.CreateTask(m.activeProject().ID, val)
				m.setStatus(err)
				m.setStatus(m.reloadTasks())
			} else {
				m.setStatus(m.store.UpdateTask(m.editID, val, m.notesOf(m.editID)))
				m.setStatus(m.reloadTasks())
			}
		}
		m.editing = false
		m.addingProject = false
		return m, nil
	case tea.KeyEscape:
		m.editing = false
		m.addingProject = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// notesOf returns the current notes for a task id (preserved when editing the
// title so UpdateTask does not clobber them).
func (m Model) notesOf(id int64) string {
	for _, t := range m.tasks {
		if t.ID == id {
			return t.Notes
		}
	}
	return ""
}

func (m Model) selectedTask() (task, bool) {
	if m.cursor < 0 || m.cursor >= len(m.tasks) {
		return task{}, false
	}
	return m.tasks[m.cursor], true
}

// task is an alias for store.Task used above; declared here to keep imports local.
type task = storeTask

func (m Model) viewTasks() string {
	if m.width > 0 && m.width < minWorkspaceWidth {
		return m.viewTasksNarrow()
	}
	return m.viewWorkspace()
}

func (m Model) viewWorkspace() string {
	// Left panel: projects.
	left := panel("Projects", m.projectsBody(), m.focus == focusProjects,
		projectsPanelWidth, m.height-1)

	rightW := m.width - projectsPanelWidth
	tasksPanelH, detailPanelH := m.rightColumnHeights()

	// Tasks panel: render from a fresh-content copy of the viewport so the
	// list (and any live elapsed times) refreshes every render, not just on
	// key events. The copy inherits m.taskVP's YOffset, which syncTaskScroll
	// still maintains on key events.
	tvp := m.taskVP
	tvp.SetContent(m.taskListBody())
	tasksPanel := panel("Tasks · "+m.activeProject().Name, tvp.View(),
		m.focus == focusTasks, rightW, tasksPanelH)

	// Details panel (scrolling viewport or notes editor).
	var detailContent string
	if m.notesEditing {
		detailContent = m.notesArea.View()
	} else if t, ok := m.selectedTask(); ok {
		dvp := m.detailVP
		dvp.SetContent(m.detailBody(t))
		detailContent = dvp.View()
	} else {
		detailContent = faintStyle.Render("No task selected.")
	}
	detailPanel := panel("Details", detailContent, false, rightW, detailPanelH)

	right := lipgloss.JoinVertical(lipgloss.Left, tasksPanel, detailPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return body + "\n" + helpStyle.Render(m.tasksHelp())
}

func (m Model) tasksHelp() string {
	if m.notesEditing {
		return "editing notes · ctrl+s save · esc cancel"
	}
	return "tab focus · j/k move · a add · e edit · n notes · t timer · h history · , settings · q"
}

// taskListBody renders the task rows for the active project.
func (m Model) taskListBody() string {
	var b strings.Builder
	running, _ := m.store.RunningEntry()
	for i, t := range m.tasks {
		cursor := "  "
		if m.focus == focusTasks && i == m.cursor {
			cursor = "▸ "
		}
		box := "[ ]"
		if t.Done {
			box = "[x]"
		}
		clock := ""
		if running != nil && running.TaskID == t.ID {
			clock = " ⏱"
		}
		suffix := ""
		if d, ok := m.elapsedFor(t.ID); ok {
			suffix = "  (" + fmtDuration(d) + ")"
		}
		line := fmt.Sprintf("%s%s %s%s%s", cursor, box, t.Title, suffix, clock)
		switch {
		case m.focus == focusTasks && i == m.cursor:
			line = selectedStyle.Render(line)
		case t.Done:
			line = doneStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(m.tasks) == 0 {
		b.WriteString(faintStyle.Render("No tasks yet — press 'a' to add one."))
	}
	if m.editing {
		verb := "New task"
		if m.addingProject {
			verb = "New project"
		} else if m.editID != 0 {
			verb = "Edit task"
		}
		b.WriteString("\n" + verb + ": " + m.input.View())
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	return b.String()
}

// viewTasksNarrow is the single-column fallback for terminals below
// minWorkspaceWidth: the task list plus help, restyled with the new palette.
func (m Model) viewTasksNarrow() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("what was next — " + m.activeProject().Name))
	b.WriteString("\n")
	b.WriteString(m.taskListBody())
	if m.notesEditing {
		b.WriteString("\n" + m.notesArea.View() + "\n")
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render(m.tasksHelp()))
	return lipgloss.NewStyle().Width(m.width).Render(b.String())
}

func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	mnt := d / time.Minute
	d -= mnt * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, mnt)
	}
	return fmt.Sprintf("%dm%02ds", mnt, s)
}

func (m Model) beginNotes() (tea.Model, tea.Cmd) {
	t, ok := m.selectedTask()
	if !ok {
		return m, nil
	}
	ta := textarea.New()
	ta.SetWidth(m.detailVP.Width())
	ta.SetHeight(m.detailVP.Height())
	ta.SetValue(t.Notes)
	cmd := ta.Focus()
	m.notesArea = ta
	m.notesEditing = true
	return m, cmd
}

func (m Model) updateNotes(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == 's' && msg.Mod == tea.ModCtrl:
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.UpdateTask(t.ID, t.Title, m.notesArea.Value()))
			m.setStatus(m.reloadTasks())
		}
		m.notesEditing = false
		m.notesArea.Blur()
		return m, nil
	case msg.Code == tea.KeyEscape:
		m.notesEditing = false
		m.notesArea.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.notesArea, cmd = m.notesArea.Update(msg)
	return m, cmd
}
