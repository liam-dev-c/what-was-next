package tui

import (
	"fmt"
	"strings"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

// detailBody builds the Details panel content for the selected task.
func (m Model) detailBody(t store.Task) string {
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

	// Notes.
	b.WriteString("\n" + selectedStyle.Render("notes") + faintStyle.Render("  (n to edit)") + "\n")
	if strings.TrimSpace(t.Notes) == "" {
		b.WriteString(faintStyle.Render("  —") + "\n")
	} else {
		b.WriteString(t.Notes + "\n")
	}
	return b.String()
}

// successText colours a short fragment with the mint success colour.
// faintStyle.Foreground(...) returns a copy; it does not mutate faintStyle.
func successText(s string) string { return faintStyle.Foreground(successColor).Render(s) }
