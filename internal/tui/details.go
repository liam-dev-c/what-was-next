package tui

import (
	"fmt"
	"strings"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

// detailBody builds the Details panel content for the selected task. The edit
// hints render only when the panel is focused, since editing lives there.
func (m Model) detailBody(t store.Task, focused bool) string {
	var b strings.Builder

	status := successText("● open")
	if t.Done {
		status = faintStyle.Render("✓ done")
	}
	fmt.Fprintf(&b, "%s   %s\n", selectedStyle.Render(t.Title), status)

	// Time tracked, with a live marker when the timer is running.
	if d, ok := m.elapsedFor(t.ID); ok {
		line := "tracked " + fmtDuration(d)
		if r, err := m.store.RunningEntry(); err == nil && r != nil && r.TaskID == t.ID {
			line += "  " + successText("⏱ running (t)")
		}
		b.WriteString(line + "\n")
	} else {
		b.WriteString(faintStyle.Render("no time tracked (t to start)") + "\n")
	}

	// Timestamps.
	created := "created " + t.CreatedAt.Local().Format("Mon 2 Jan")
	if t.DoneAt != nil {
		created += "  ·  completed " + t.DoneAt.Local().Format("Mon 2 Jan")
	}
	b.WriteString(faintStyle.Render(created) + "\n")

	// Tags.
	b.WriteString("\n" + selectedStyle.Render("tags") + editHint(focused, "g") + "\n")
	if len(t.Tags) == 0 {
		b.WriteString(faintStyle.Render("  —") + "\n")
	} else {
		b.WriteString("  " + tagLabel(t.Tags) + "\n")
	}

	// Notes.
	b.WriteString("\n" + selectedStyle.Render("notes") + editHint(focused, "n") + "\n")
	if strings.TrimSpace(t.Notes) == "" {
		b.WriteString(faintStyle.Render("  —") + "\n")
	} else {
		b.WriteString(t.Notes + "\n")
	}
	return b.String()
}

// editHint renders the faint "(k to edit)" cue next to a section label, but
// only while the Details panel is focused — that is the only place editing works.
func editHint(focused bool, key string) string {
	if !focused {
		return ""
	}
	return faintStyle.Render("  (" + key + " to edit)")
}

// tagLabel renders tags as a space-separated "#name" list.
func tagLabel(tags []string) string {
	return "#" + strings.Join(tags, " #")
}

// successText colours a short fragment with the mint success colour.
// faintStyle.Foreground(...) returns a copy; it does not mutate faintStyle.
func successText(s string) string { return faintStyle.Foreground(successColor).Render(s) }
