package tui

import (
	"fmt"
	"sort"
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
		m.cycleFocus(1)
		return m, nil
	case "shift+tab":
		m.cycleFocus(-1)
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
	switch m.focus {
	case focusProjects:
		return m.updateProjectsPanel(msg)
	case focusDetails:
		return m.updateDetailsPanel(msg)
	default:
		return m.updateTasksPanel(msg)
	}
}

// cycleFocus advances focus by dir (+1 forward, -1 back) through focusOrder.
func (m *Model) cycleFocus(dir int) {
	idx := 0
	for i, f := range focusOrder {
		if f == m.focus {
			idx = i
			break
		}
	}
	idx = (idx + dir + len(focusOrder)) % len(focusOrder)
	m.setFocus(focusOrder[idx])
}

// setFocus moves focus to f, priming panel-specific state.
func (m *Model) setFocus(f focusArea) {
	m.focus = f
	if f == focusProjects {
		m.projCursor = m.active
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

// updateTasksPanel handles keys when the Tasks panel is focused. The list is
// navigation-only: selecting, adding, reordering, and the completed toggle.
// Everything that mutates a single task lives in the Details panel.
func (m Model) updateTasksPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < m.visibleCount()-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "c":
		m.showAllCompleted = !m.showAllCompleted
		if n := m.visibleCount(); m.cursor >= n {
			m.cursor = max(0, n-1)
		}
	case "a":
		m.beginInput(0, "", false)
		return m, textinput.Blink
	case "enter", "space":
		if _, ok := m.selectedTask(); ok {
			m.setFocus(focusDetails)
		}
	case "J":
		// Reorder only open tasks; completed tasks are ordered by completion.
		if t, ok := m.selectedTask(); ok && !t.Done {
			m.setStatus(m.store.MoveTask(t.ID, 1))
			m.setStatus(m.reloadTasks())
			if m.cursor < m.visibleCount()-1 {
				m.cursor++
			}
		}
	case "K":
		if t, ok := m.selectedTask(); ok && !t.Done {
			m.setStatus(m.store.MoveTask(t.ID, -1))
			m.setStatus(m.reloadTasks())
			if m.cursor > 0 {
				m.cursor--
			}
		}
	}
	m.syncTaskScroll()
	return m, nil
}

// updateDetailsPanel handles keys when the Details panel is focused. It owns
// every per-task action for the selected task.
func (m Model) updateDetailsPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	t, ok := m.selectedTask()
	if !ok {
		if msg.String() == "esc" {
			m.setFocus(focusTasks)
		}
		return m, nil
	}
	switch msg.String() {
	case "e":
		m.beginInput(t.ID, t.Title, false)
		return m, textinput.Blink
	case "n":
		return m.beginNotes()
	case "g":
		return m.beginTags()
	case "enter", "space":
		m.setStatus(m.store.SetTaskDone(t.ID, !t.Done))
		m.setStatus(m.reloadTasks())
	case "t":
		m.toggleTimer(t.ID)
	case "d":
		m.setStatus(m.store.DeleteTask(t.ID))
		m.setStatus(m.reloadTasks())
		m.setFocus(focusTasks)
	case "j", "down":
		m.scrollDetails(t, 1)
	case "k", "up":
		m.scrollDetails(t, -1)
	case "esc":
		m.setFocus(focusTasks)
	}
	return m, nil
}

// scrollDetails nudges the details viewport by delta lines, priming its content
// first so SetYOffset clamps against the real height.
func (m *Model) scrollDetails(t task, delta int) {
	m.detailVP.SetContent(m.detailBody(t, true))
	off := m.detailVP.YOffset() + delta
	if off < 0 {
		off = 0
	}
	m.detailVP.SetYOffset(off)
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

// beginTags opens the shared text input prefilled with the selected task's
// current tags as a comma-separated list. Enter replaces the tag set.
func (m Model) beginTags() (tea.Model, tea.Cmd) {
	t, ok := m.selectedTask()
	if !ok {
		return m, nil
	}
	ti := textinput.New()
	ti.SetValue(strings.Join(t.Tags, ", "))
	ti.Focus()
	ti.CursorEnd()
	m.input = ti
	m.editing = true
	m.editID = t.ID
	m.taggingTask = true
	return m, textinput.Blink
}

// parseTags splits a comma-separated tag input into names; the store handles
// trimming, de-duplication, and case-folding.
func parseTags(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		if m.taggingTask {
			// An empty value clears all tags, so this runs unconditionally.
			m.setStatus(m.store.SetTaskTags(m.editID, parseTags(m.input.Value())))
			m.setStatus(m.reloadTasks())
			m.editing = false
			m.taggingTask = false
			return m, nil
		}
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
		m.taggingTask = false
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

// taskSelected reports whether row i is the highlighted task. The selection is
// shown while either the Tasks list or the Details panel is focused, since both
// act on that task; it is hidden while Projects is focused.
func (m Model) taskSelected(i int) bool {
	return (m.focus == focusTasks || m.focus == focusDetails) && i == m.cursor
}

func (m Model) selectedTask() (task, bool) {
	vis, _ := m.visibleTasks()
	if m.cursor < 0 || m.cursor >= len(vis) {
		return task{}, false
	}
	return vis[m.cursor], true
}

// task is an alias for store.Task used above; declared here to keep imports local.
type task = storeTask

// visibleTasks is the current display order for the tasks list, computed from
// m.tasks against the wall clock and the show-all-completed toggle.
func (m Model) visibleTasks() (tasks []task, doneStart int) {
	return partitionTasks(m.tasks, time.Now(), m.showAllCompleted)
}

// startOfDay returns local midnight for t.
func startOfDay(t time.Time) time.Time {
	y, mo, d := t.Date()
	return time.Date(y, mo, d, 0, 0, 0, 0, t.Location())
}

// partitionTasks orders tasks for display: open tasks first in their existing
// sort_order, then completed tasks sorted by completion time (newest first).
// Completed tasks finished before today's local midnight are hidden unless
// showAll is set. doneStart is the index where the completed group begins,
// which equals len(visible) when no completed task is shown.
func partitionTasks(tasks []task, now time.Time, showAll bool) (visible []task, doneStart int) {
	today := startOfDay(now)
	var open, done []task
	for _, t := range tasks {
		if !t.Done {
			open = append(open, t)
			continue
		}
		if !showAll && (t.DoneAt == nil || t.DoneAt.Local().Before(today)) {
			continue
		}
		done = append(done, t)
	}
	sort.SliceStable(done, func(i, j int) bool { return doneAtAfter(done[i], done[j]) })
	visible = append(open, done...)
	return visible, len(open)
}

// doneAtAfter reports whether a was completed more recently than b. A nil
// DoneAt (should not happen for a done task, but guard anyway) sorts last.
func doneAtAfter(a, b task) bool {
	switch {
	case a.DoneAt == nil:
		return false
	case b.DoneAt == nil:
		return true
	default:
		return a.DoneAt.After(*b.DoneAt)
	}
}

func (m Model) viewTasks() string {
	if m.width > 0 && m.width < minWorkspaceWidth {
		return m.viewTasksNarrow()
	}
	return m.viewWorkspace()
}

func (m Model) viewWorkspace() string {
	rightW := m.width - projectsPanelWidth
	tasksPanelH, detailPanelH := m.rightColumnHeights()

	// Left panel: projects. Height matches the right column's total so both
	// sides stay aligned regardless of rounding/clamps in rightColumnHeights.
	leftH := tasksPanelH + detailPanelH
	left := panel("Projects", m.projectsBody(), m.focus == focusProjects,
		projectsPanelWidth, leftH)

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
		dvp.SetContent(m.detailBody(t, m.focus == focusDetails))
		detailContent = dvp.View()
	} else {
		detailContent = faintStyle.Render("No task selected.")
	}
	detailPanel := panel("Details", detailContent, m.focus == focusDetails, rightW, detailPanelH)

	right := lipgloss.JoinVertical(lipgloss.Left, tasksPanel, detailPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return body + "\n" + helpStyle.Render(m.tasksHelp())
}

func (m Model) tasksHelp() string {
	if m.notesEditing {
		return "editing notes · ctrl+s save · esc cancel"
	}
	switch m.focus {
	case focusProjects:
		return "tab focus · j/k move · enter select · a add project · h history · , settings · q"
	case focusDetails:
		return "tab focus · e title · n notes · g tags · enter done · t timer · d delete · j/k scroll · esc back · q"
	default:
		return "tab focus · j/k move · enter open · a add · J/K reorder · c completed · h history · , settings · q"
	}
}

// taskListBody renders the task rows for the active project.
func (m Model) taskListBody() string {
	var b strings.Builder
	running, _ := m.store.RunningEntry()
	vis, doneStart := m.visibleTasks()
	for i, t := range vis {
		if i == doneStart && doneStart < len(vis) {
			b.WriteString(faintStyle.Render("─── completed ───") + "\n")
		}
		cursor := "  "
		if m.taskSelected(i) {
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
		tags := ""
		if len(t.Tags) > 0 {
			tags = "  " + tagLabel(t.Tags)
		}
		line := fmt.Sprintf("%s%s %s%s%s%s", cursor, box, t.Title, suffix, tags, clock)
		switch {
		case m.taskSelected(i):
			line = selectedStyle.Render(line)
		case t.Done:
			line = doneStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(vis) == 0 {
		if len(m.tasks) == 0 {
			b.WriteString(faintStyle.Render("No tasks yet — press 'a' to add one."))
		} else {
			b.WriteString(faintStyle.Render("No open tasks — press 'c' to show completed."))
		}
	}
	if m.editing {
		verb := "New task"
		switch {
		case m.taggingTask:
			verb = "Tags (comma-separated)"
		case m.addingProject:
			verb = "New project"
		case m.editID != 0:
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
