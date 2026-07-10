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
