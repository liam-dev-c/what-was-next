package tui

import "strings"

// projectsBody renders the Projects panel list. The cursor row is marked with
// ▸ (accent) and the active project with ●.
func (m Model) projectsBody() string {
	var b strings.Builder
	for i, p := range m.projects {
		cursor := "  "
		if m.focus == focusProjects && i == m.projCursor {
			cursor = "▸ "
		}
		marker := "  "
		if i == m.active {
			marker = "● "
		}
		line := cursor + marker + p.Name
		if m.focus == focusProjects && i == m.projCursor {
			line = selectedStyle.Render(line)
		} else if i == m.active {
			line = faintStyle.Foreground(accentColor).Render(line)
		}
		b.WriteString(line + "\n")
	}
	b.WriteString(faintStyle.Render("\n+ add (a)"))
	return b.String()
}
