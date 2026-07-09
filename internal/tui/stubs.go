package tui

import tea "github.com/charmbracelet/bubbletea"

// Temporary stubs — replaced by Tasks 7–10. Delete each as its task lands.
func (m Model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd)    { return m, nil }
func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m, nil }
func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd)  { return m, nil }
func (m Model) viewTasks() string                                  { return "tasks" }
func (m Model) viewProjects() string                               { return "projects" }
func (m Model) viewSummary() string                                { return "summary" }
