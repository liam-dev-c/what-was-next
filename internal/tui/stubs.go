package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

// Temporary stubs — replaced by Tasks 8–10. Delete each as its task lands.
func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m, nil }
func (m Model) viewSummary() string                               { return "summary" }

// loadSummary is called by tasks.go's 's' key handler (Task 7's brief), but is
// formally introduced by Task 10 alongside summary.go. Implemented here ahead
// of schedule, verbatim to Task 10's spec, so Task 7 compiles; Task 10 should
// remove this copy when it lands the real summary.go.
func (m *Model) loadSummary() {
	sum, err := m.store.DailySummary(time.Now())
	if err != nil {
		m.setStatus(err)
		return
	}
	m.summary = sum
}
