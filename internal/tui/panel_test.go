package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func TestPanelExactSizeWhenPadded(t *testing.T) {
	// A panel with short content must still render to its full requested size
	// (lipgloss v2 sizing is border-inclusive). This guards panel alignment.
	out := panel("T", "one line", false, 24, 12)
	if h := lipgloss.Height(out); h != 12 {
		t.Fatalf("padded panel height = %d, want 12", h)
	}
	if w := lipgloss.Width(out); w != 24 {
		t.Fatalf("padded panel width = %d, want 24", w)
	}
}

func TestWorkspacePanelsAligned(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.store.CreateProject("Personal") // short projects list → padded left panel
	m.reloadProjects()
	m.store.CreateTask(m.activeProject().ID, "Ship it")
	m.reloadTasks()
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m = mi.(Model)
	tasksH, detailH := m.rightColumnHeights()
	left := panel("Projects", m.projectsBody(), false, projectsPanelWidth, tasksH+detailH)
	if lipgloss.Height(left) != tasksH+detailH {
		t.Fatalf("left panel height = %d, want %d (== right column)", lipgloss.Height(left), tasksH+detailH)
	}
}

func TestPanelRendersTitleWithinWidth(t *testing.T) {
	out := panel("Projects", "Work\nPersonal", true, 20, 6)
	if !strings.Contains(out, "Projects") {
		t.Fatalf("panel missing title, got:\n%s", out)
	}
	if !strings.Contains(out, "Work") {
		t.Fatalf("panel missing body, got:\n%s", out)
	}
	if w := lipgloss.Width(out); w > 20 {
		t.Fatalf("panel width %d exceeds requested 20", w)
	}
}

func TestPanelFocusChangesBorderColor(t *testing.T) {
	if panel("T", "b", true, 12, 4) == panel("T", "b", false, 12, 4) {
		t.Fatal("focused and unfocused panels should differ")
	}
}

func TestWindowSizeSizesViewports(t *testing.T) {
	m := newModel(t)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mi.(Model)
	if m.taskVP.Width() <= 0 || m.taskVP.Height() <= 0 {
		t.Fatalf("task viewport not sized: %dx%d", m.taskVP.Width(), m.taskVP.Height())
	}
	if m.detailVP.Width() <= 0 || m.detailVP.Height() <= 0 {
		t.Fatalf("detail viewport not sized: %dx%d", m.detailVP.Width(), m.detailVP.Height())
	}
	// Right column width = total - projects panel.
	if m.taskVP.Width() > 120-projectsPanelWidth {
		t.Fatalf("task viewport too wide: %d", m.taskVP.Width())
	}
}

func TestViewTasksWideHasThreePanels(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mi.(Model)
	out := m.viewTasks()
	for _, want := range []string{"Projects", "Tasks", "Details"} {
		if !strings.Contains(out, want) {
			t.Fatalf("wide view missing %q panel:\n%s", want, out)
		}
	}
}

func TestViewTasksNarrowFallsBack(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	m = mi.(Model)
	out := m.viewTasks()
	if w := lipgloss.Width(out); w > 50 {
		t.Fatalf("narrow view width %d exceeds 50", w)
	}
}

func TestTasksPanelRendersOnFirstView(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.store.CreateTask(m.activeProject().ID, "Visible task")
	m.reloadTasks()
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mi.(Model)
	out := m.viewTasks() // no key press first
	// The Details panel also renders the selected task's title, so assert on
	// the task-list row form ("[ ] <title>", produced only by taskListBody)
	// rather than the bare title, which would pass even with a blank Tasks
	// panel.
	if !strings.Contains(out, "[ ] Visible task") {
		t.Fatalf("tasks panel should render task content on first view:\n%s", out)
	}
}

func TestNarrowRendersNotesEditor(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 20})
	m = mi.(Model)
	m.focus = focusDetails // notes editing lives in the Details panel now
	mi, _ = m.updateTasks(key('n'))
	m = mi.(Model)
	out := m.viewTasks() // narrow path
	// the textarea renders a cursor/prompt line; assert notesEditing help + non-empty editor region.
	if !strings.Contains(out, "editing notes") {
		t.Fatalf("narrow view should show notes-editing help:\n%s", out)
	}
}

func TestRightColumnDetailsLargerThanTasks(t *testing.T) {
	m := newModel(t)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mi.(Model)
	tasksH, detailH := m.rightColumnHeights()
	if detailH <= tasksH {
		t.Fatalf("want Details taller than Tasks, got tasks=%d detail=%d", tasksH, detailH)
	}
	if tasksH+detailH > 40-1 {
		t.Fatalf("panels overflow height: tasks=%d detail=%d", tasksH, detailH)
	}
	// Detail viewport should be sized from detailH, not the old fixed 9.
	if m.detailVP.Height() <= 3 {
		t.Fatalf("detail viewport too small: %d", m.detailVP.Height())
	}
}

func TestWorkspaceFitsHeightAndShowsHelp(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.store.CreateTask(m.activeProject().ID, "Task one")
	m.reloadTasks()
	for _, h := range []int{20, 24, 40} {
		mi, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: h})
		mm := mi.(Model)
		mm.syncTaskScroll()
		out := mm.viewTasks()
		lines := strings.Split(out, "\n")
		if len(lines) > h {
			t.Fatalf("height %d: rendered %d lines, overflows terminal", h, len(lines))
		}
		last := ""
		for _, l := range lines {
			if strings.TrimSpace(stripHelpANSI(l)) != "" {
				last = l
			}
		}
		if !strings.Contains(last, "history") || !strings.Contains(last, "settings") {
			t.Fatalf("height %d: help line not the last visible line, got %q", h, last)
		}
	}
}

// stripHelpANSI removes ANSI escapes so the assertion sees plain text.
func stripHelpANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

func TestTaskScrollFollowsCursor(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 16})
	m = mi.(Model)
	for i := 0; i < 30; i++ {
		m.store.CreateTask(m.activeProject().ID, "Task "+string(rune('a'+i%26)))
	}
	m.reloadTasks()
	// Move cursor to the bottom.
	for i := 0; i < 29; i++ {
		mi, _ = m.updateTasks(key('j'))
		m = mi.(Model)
	}
	if m.taskVP.YOffset() == 0 {
		t.Fatal("want task viewport scrolled to keep cursor visible")
	}
}
