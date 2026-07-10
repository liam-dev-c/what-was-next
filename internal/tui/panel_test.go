package tui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
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
