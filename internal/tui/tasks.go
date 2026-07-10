package tui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m Model) updateTasks(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.updateTaskInput(msg)
	}
	m.status = ""
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
		m.beginEdit(0, "")
		return m, textinput.Blink
	case "e":
		if t, ok := m.selectedTask(); ok {
			m.beginEdit(t.ID, t.Title)
			return m, textinput.Blink
		}
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
		m.projCursor = m.active
		m.focus = focusProjects
	case "s":
		m.summaryPeriod = periodDay
		m.loadSummary()
		m.screen = screenSummary
	case ",":
		m.screen = screenSettings
	}
	return m, nil
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

func (m *Model) beginEdit(id int64, initial string) {
	ti := textinput.New()
	ti.SetValue(initial)
	ti.Focus()
	ti.CursorEnd()
	m.input = ti
	m.editing = true
	m.editID = id
}

func (m Model) updateTaskInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		title := strings.TrimSpace(m.input.Value())
		if title != "" {
			if m.editID == 0 {
				_, err := m.store.CreateTask(m.activeProject().ID, title)
				m.setStatus(err)
			} else {
				m.setStatus(m.store.UpdateTask(m.editID, title, ""))
			}
			m.setStatus(m.reloadTasks())
		}
		m.editing = false
		return m, nil
	case tea.KeyEscape:
		m.editing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
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
	tasksPanelH := m.height - detailPanelHeight - 1

	// Tasks panel (scrolling viewport).
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
	detailPanel := panel("Details", detailContent, false, rightW, detailPanelHeight)

	right := lipgloss.JoinVertical(lipgloss.Left, tasksPanel, detailPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return body + "\n" + helpStyle.Render(m.tasksHelp())
}

func (m Model) tasksHelp() string {
	if m.notesEditing {
		return "editing notes · ctrl+s save · esc cancel"
	}
	return "tab focus · j/k move · a add · e edit · n notes · t timer · s summary · , settings · q"
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
