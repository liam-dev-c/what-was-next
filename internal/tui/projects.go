package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.updateProjectInput(msg)
	}
	switch msg.String() {
	case "j", "down":
		if m.projCursor < len(m.projects)-1 {
			m.projCursor++
		}
	case "k", "up":
		if m.projCursor > 0 {
			m.projCursor--
		}
	case "a":
		ti := textinput.New()
		ti.Focus()
		m.input = ti
		m.editing = true
		m.editID = 0
		return m, textinput.Blink
	case "enter", " ":
		m.active = m.projCursor
		m.cursor = 0
		m.setStatus(m.reloadTasks())
		m.screen = screenTasks
	case "esc", "p":
		m.screen = screenTasks
	}
	return m, nil
}

func (m Model) updateProjectInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.input.Value())
		if name != "" {
			_, err := m.store.CreateProject(name)
			m.setStatus(err)
			m.setStatus(m.reloadProjects())
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

func (m Model) viewProjects() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Projects"))
	b.WriteString("\n")
	for i, p := range m.projects {
		cursor := "  "
		if i == m.projCursor {
			cursor = "> "
		}
		marker := "  "
		if i == m.active {
			marker = "* "
		}
		line := cursor + marker + p.Name
		if i == m.projCursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if m.editing {
		b.WriteString("\nNew project: " + m.input.View() + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	b.WriteString(helpStyle.Render(
		"\nj/k move · enter select · a add · esc back"))
	return b.String()
}
