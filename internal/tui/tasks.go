package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.updateTaskInput(msg)
	}
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
	case "enter", " ":
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
		m.screen = screenProjects
	case "s":
		m.loadSummary()
		m.screen = screenSummary
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

func (m Model) updateTaskInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
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
	case tea.KeyEsc:
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
	var b strings.Builder
	b.WriteString(titleStyle.Render("what was next — " + m.activeProject().Name))
	b.WriteString("\n")

	running, _ := m.store.RunningEntry()
	for i, t := range m.tasks {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		box := "[ ]"
		if t.Done {
			box = "[x]"
		}
		clock := "  "
		if running != nil && running.TaskID == t.ID {
			clock = "⏱ "
		}
		suffix := ""
		if d, ok := m.elapsedFor(t.ID); ok {
			suffix = "  (" + fmtDuration(d) + ")"
		}
		line := fmt.Sprintf("%s%s %s%s%s", cursor, box, clock, t.Title, suffix)
		switch {
		case i == m.cursor:
			line = selectedStyle.Render(line)
		case t.Done:
			line = doneStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render("No tasks yet — press 'a' to add one.\n"))
	}

	if m.editing {
		verb := "New task"
		if m.editID != 0 {
			verb = "Edit task"
		}
		b.WriteString("\n" + verb + ": " + m.input.View() + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	b.WriteString(helpStyle.Render(
		"\na add · e edit · enter done · d del · J/K move · t timer · p projects · s summary · q quit"))
	return b.String()
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
