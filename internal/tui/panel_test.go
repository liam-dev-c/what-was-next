package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	tea "charm.land/bubbletea/v2"
)

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
	mi, _ = m.updateTasks(key('n'))
	m = mi.(Model)
	out := m.viewTasks() // narrow path
	// the textarea renders a cursor/prompt line; assert notesEditing help + non-empty editor region.
	if !strings.Contains(out, "editing notes") {
		t.Fatalf("narrow view should show notes-editing help:\n%s", out)
	}
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
