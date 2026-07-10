# Panel Layout Tasks Workspace Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn the tasks screen into a three-panel workspace (Projects sidebar · Tasks · Details) with a softer dark-mode palette, editable per-task notes, and vertical scrolling.

**Architecture:** Keep the Bubble Tea v2 single-`Model` design. The `screenTasks` view composes three Lipgloss bordered panels via `JoinHorizontal`/`JoinVertical`. A `focus` field routes list keys to either the Projects or Tasks panel; the standalone Projects *screen* is removed and folded into the panel. Tasks and Details bodies render through `bubbles/v2/viewport` models for scrolling; notes edit through a `bubbles/v2/textarea`. Summary and Settings remain full-screen.

**Tech Stack:** Go, `charm.land/bubbletea/v2`, `charm.land/bubbles/v2` (`viewport`, `textarea`, `textinput`), `charm.land/lipgloss/v2`, SQLite store (`internal/store`).

## Global Constraints

- Module path: `github.com/liam-dev-c/what-was-next`; packages `internal/tui`, `internal/store`.
- Bubble Tea v2: no `WithAltScreen` option — `View()` sets `v.AltScreen = true`. Key events arrive as `tea.KeyPressMsg`; `msg.String()` gives the key, `msg.Code` gives special keys (`tea.KeyEnter`, `tea.KeyEscape`).
- Colours are truecolor hex via `lipgloss.Color("#RRGGBB")`. Ghostty renders truecolor.
- Git commits: **no `Co-Authored-By` trailer** (per repo convention).
- Every change keeps `go build ./...` and `go test ./...` green before commit.
- Work on branch `panel-layout` (already checked out; the spec is committed there).
- `store.Task` fields: `ID int64, ProjectID int64, Title string, Notes string, Done bool, SortOrder int64, CreatedAt time.Time, DoneAt *time.Time`. `store.Project`: `ID int64, Name string, CreatedAt time.Time`.
- Persist notes via `store.UpdateTask(id int64, title, notes string) error` (updates both columns — always pass the existing title).

---

## File Structure

- `internal/tui/styles.go` — **Modify.** Named palette vars (hex) + panel/border/title styles + a `panel(...)` render helper and layout constants.
- `internal/tui/app.go` — **Modify.** `Model` gains `focus`, `addingProject`, `notesEditing`, `notesArea textarea.Model`, `taskVP`/`detailVP viewport.Model`. `WindowSizeMsg` sizes the viewports/textarea. Router drops `screenProjects`.
- `internal/tui/tasks.go` — **Modify.** Focus-aware key routing; three-panel composition in `viewTasks`; narrow-terminal fallback; notes-edit mode; unified input commit.
- `internal/tui/details.go` — **Create.** `detailBody(...)` builds the Details panel content string.
- `internal/tui/projects.go` — **Modify.** Delete the standalone screen (`updateProjects`, `updateProjectInput`, `viewProjects`); keep a `projectsBody(...)` helper that renders the Projects panel list.
- `internal/tui/projects_test.go`, `internal/tui/tasks_test.go` — **Modify.** Rewrite project-nav tests to drive the panel via `updateTasks`; update the `p`-key test.
- New tests: `internal/tui/details_test.go`, `internal/tui/panel_test.go`.

---

### Task 1: Palette and panel styles

**Files:**
- Modify: `internal/tui/styles.go`
- Test: `internal/tui/panel_test.go` (create)

**Interfaces:**
- Produces:
  - Colour vars: `accentColor, dimColor, borderColor, errorColor, successColor, faintColor lipgloss.Color`.
  - Styles (unchanged names so existing views keep compiling): `titleStyle, selectedStyle, doneStyle, statusStyle, helpStyle, faintStyle lipgloss.Style`.
  - New styles: `panelTitleStyle, borderStyle, borderFocusStyle lipgloss.Style`.
  - Layout consts: `projectsPanelWidth = 24`, `minWorkspaceWidth = 80`, `detailPanelHeight = 9`.
  - Helper: `func panel(title, body string, focused bool, width, height int) string` — renders a rounded-border box `width`×`height` cells wide/tall, accent border when `focused` else dim border, with `title` as a styled header line above `body`.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/panel_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestPanel -v`
Expected: FAIL — `undefined: panel`.

- [ ] **Step 3: Replace `internal/tui/styles.go` with the new palette and helper**

```go
package tui

import "charm.land/lipgloss/v2"

// Soft truecolor palette tuned for a dark background (dark-mode Ghostty).
var (
	accentColor  = lipgloss.Color("#C8A2FF") // lavender: titles, selection, focus
	dimColor     = lipgloss.Color("#9BA3B4") // light slate: secondary text
	borderColor  = lipgloss.Color("#4A4E5A") // muted grey: unfocused panel frame
	errorColor   = lipgloss.Color("#F2A0A0") // rose: errors/status
	successColor = lipgloss.Color("#A7E0B8") // mint: done / running timer
	faintColor   = lipgloss.Color("#6B7280") // slate: hints/labels
)

const (
	projectsPanelWidth = 24 // total cells incl. borders
	minWorkspaceWidth  = 80 // below this, fall back to single column
	detailPanelHeight  = 9  // total rows incl. borders for the Details panel
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(accentColor).
			Bold(true)

	doneStyle = lipgloss.NewStyle().
			Foreground(faintColor).
			Strikethrough(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(errorColor)

	helpStyle = lipgloss.NewStyle().
			Foreground(faintColor).
			MarginTop(1)

	faintStyle = lipgloss.NewStyle().
			Foreground(faintColor)

	panelTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor)

	borderFocusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor)
)

// panel renders body inside a rounded border sized to width×height cells,
// with title as a styled header line. Border is accent-coloured when focused.
func panel(title, body string, focused bool, width, height int) string {
	style := borderStyle
	if focused {
		style = borderFocusStyle
	}
	inner := width - 2   // left+right border
	rows := height - 2   // top+bottom border
	head := panelTitleStyle.Render(title)
	content := head + "\n" + body
	return style.Width(inner).Height(rows).Render(content)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestPanel -v`
Expected: PASS. Then `go build ./...` — Expected: builds (existing views still reference `titleStyle` etc.).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/styles.go internal/tui/panel_test.go
git commit -m "Add soft palette and panel render helper"
```

---

### Task 2: Model state and window sizing

**Files:**
- Modify: `internal/tui/app.go`
- Test: `internal/tui/panel_test.go` (add)

**Interfaces:**
- Consumes: `panel`, layout consts, `viewport`, `textarea` (Task 1).
- Produces on `Model`:
  - `focus focusArea` where `type focusArea int` with `focusTasks focusArea = iota` then `focusProjects`.
  - `addingProject bool`, `notesEditing bool`, `notesArea textarea.Model`.
  - `taskVP viewport.Model`, `detailVP viewport.Model`.
  - `func (m *Model) resizePanels()` — sizes the two viewports and the notes textarea from `m.width`/`m.height`.

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/panel_test.go`:

```go
import tea "charm.land/bubbletea/v2" // add to existing imports

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestWindowSizeSizesViewports -v`
Expected: FAIL — `m.taskVP undefined`.

- [ ] **Step 3: Add state and sizing to `internal/tui/app.go`**

Add imports `"charm.land/bubbles/v2/textarea"` and `"charm.land/bubbles/v2/viewport"`.

Add above `Model`:

```go
// focusArea selects which panel receives list navigation keys on the tasks
// workspace.
type focusArea int

const (
	focusTasks focusArea = iota
	focusProjects
)
```

Add fields to `Model` (near the input state):

```go
	// panel workspace state
	focus         focusArea
	addingProject bool // input is naming a new project, not a task
	notesEditing  bool
	notesArea     textarea.Model
	taskVP        viewport.Model
	detailVP      viewport.Model
```

In `New`, before `return m, nil`, initialise viewports and textarea:

```go
	m.taskVP = viewport.New()
	m.detailVP = viewport.New()
	m.notesArea = textarea.New()
```

Add the sizing helper:

```go
// resizePanels lays out the three panels from the current terminal size.
// Left: Projects (fixed). Right column: Tasks over Details, each a bordered
// panel whose inner viewport is (panel - 2 border - 1 title) tall.
func (m *Model) resizePanels() {
	rightW := m.width - projectsPanelWidth
	if rightW < 1 {
		rightW = 1
	}
	innerW := rightW - 2 // borders
	if innerW < 1 {
		innerW = 1
	}
	detailInner := detailPanelHeight - 2 - 1 // borders + title
	if detailInner < 1 {
		detailInner = 1
	}
	// Tasks panel gets the remaining height; reserve 1 row for the help line.
	tasksPanelH := m.height - detailPanelHeight - 1
	taskInner := tasksPanelH - 2 - 1
	if taskInner < 1 {
		taskInner = 1
	}
	m.taskVP.SetWidth(innerW)
	m.taskVP.SetHeight(taskInner)
	m.detailVP.SetWidth(innerW)
	m.detailVP.SetHeight(detailInner)
	m.notesArea.SetWidth(innerW)
	m.notesArea.SetHeight(detailInner)
}
```

In `Update`, extend the `tea.WindowSizeMsg` case:

```go
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resizePanels()
		return m, nil
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestWindowSizeSizesViewports -v`
Expected: PASS. Then `go test ./internal/tui/` — Expected: all existing tests still pass (views unchanged).

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/panel_test.go
git commit -m "Add panel/notes/viewport state and window sizing"
```

---

### Task 3: Projects panel body + Details panel body

**Files:**
- Modify: `internal/tui/projects.go`
- Create: `internal/tui/details.go`
- Test: `internal/tui/details_test.go` (create)

**Interfaces:**
- Consumes: `Model`, `fmtDuration`, `m.elapsedFor` (app.go), styles (Task 1).
- Produces:
  - `func (m Model) projectsBody() string` — the list of project names with cursor (`▸`) and active (`●`) markers, styled.
  - `func (m Model) detailBody(t store.Task) string` — Details content: title, status (`● open`/`✓ done`), created/completed dates, tracked time with live indicator, and notes (or a `n to add notes` hint).

- [ ] **Step 1: Write the failing test**

Create `internal/tui/details_test.go`:

```go
package tui

import (
	"strings"
	"testing"
)

func TestDetailBodyShowsStatusAndTime(t *testing.T) {
	m := newModel(t)
	tk, _ := m.store.CreateTask(m.activeProject().ID, "Ship it")
	m.reloadTasks()
	body := m.detailBody(m.tasks[0])
	if !strings.Contains(body, "Ship it") {
		t.Fatalf("detail missing title: %s", body)
	}
	if !strings.Contains(body, "open") {
		t.Fatalf("detail missing status: %s", body)
	}
	if !strings.Contains(strings.ToLower(body), "notes") {
		t.Fatalf("detail missing notes section: %s", body)
	}
	_ = tk
}

func TestDetailBodyDoneStatus(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Done task")
	m.reloadTasks()
	m.store.SetTaskDone(m.tasks[0].ID, true)
	m.reloadTasks()
	if !strings.Contains(m.detailBody(m.tasks[0]), "done") {
		t.Fatalf("want done status, got: %s", m.detailBody(m.tasks[0]))
	}
}

func TestProjectsBodyMarksActive(t *testing.T) {
	m := newModel(t) // has default "Inbox" project active
	body := m.projectsBody()
	if !strings.Contains(body, "Inbox") {
		t.Fatalf("projects body missing Inbox: %s", body)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run "TestDetailBody|TestProjectsBody" -v`
Expected: FAIL — `m.detailBody undefined`, `m.projectsBody undefined`.

- [ ] **Step 3: Create `internal/tui/details.go`**

```go
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
```

Note: `faintStyle.Foreground(...)` returns a copy; it does not mutate `faintStyle`.

- [ ] **Step 4: Rewrite `internal/tui/projects.go` to a panel-body helper**

Replace the entire file with:

```go
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
```

(The old `updateProjects`/`updateProjectInput`/`viewProjects` are removed; their logic moves into `tasks.go` in Task 5.)

- [ ] **Step 5: Delete the projects-screen tests that reference removed funcs**

Replace `internal/tui/projects_test.go` with a placeholder so the package compiles; real panel tests are added in Task 5:

```go
package tui

// Projects are now a panel on the tasks screen; navigation tests live in
// tasks_test.go (TestProjectsPanelFocusAndSelect, TestAddProjectViaPanel).
```

- [ ] **Step 6: Update the router so the package builds**

The router in `app.go` still lists `case screenProjects`. Remove `screenProjects` usage now: in `app.go` `View()` and `Update()`, delete the `screenProjects` branches. In the `screen` const block, delete `screenProjects`. In `tasks.go`/`summary.go` the `case "p"` handlers still set `m.screen = screenProjects` — Task 5 replaces those; for now change them to `m.focus = focusProjects` (in `tasks.go`) and remove the `p` case from `summary.go`.

Concretely, in `app.go` remove:

```go
	case screenProjects:
		content = m.viewProjects()
```
and
```go
		case screenProjects:
			return m.updateProjects(msg)
```
and the `screenProjects` line from the `const` block.

- [ ] **Step 7: Run tests to verify they pass**

Run: `go build ./... && go test ./internal/tui/ -run "TestDetailBody|TestProjectsBody" -v`
Expected: PASS. (Other tests may fail to compile until Task 5 rewrites `tasks.go`; if so, proceed to Task 5 before running the full suite — but `go build ./...` of non-test code must pass here.)

If `go build ./...` fails because `tasks.go`/`summary.go` still reference `screenProjects`, apply the Step 6 edits until it builds.

- [ ] **Step 8: Commit**

```bash
git add internal/tui/details.go internal/tui/projects.go internal/tui/details_test.go internal/tui/projects_test.go internal/tui/app.go internal/tui/summary.go internal/tui/tasks.go
git commit -m "Add projects/details panel bodies; remove standalone projects screen"
```

---

### Task 4: Three-panel composition + narrow fallback

**Files:**
- Modify: `internal/tui/tasks.go` (the `viewTasks` function only)
- Test: `internal/tui/panel_test.go` (add)

**Interfaces:**
- Consumes: `panel`, `projectsBody`, `detailBody`, viewports, `resizePanels` (Tasks 1–3).
- Produces:
  - `func (m Model) viewTasks() string` — composes Projects | (Tasks / Details) when `m.width >= minWorkspaceWidth`, else `m.viewTasksNarrow()`.
  - `func (m Model) viewTasksNarrow() string` — today's single-column list, restyled.
  - `func (m Model) taskListBody() string` — the task rows (extracted from the old `viewTasks`).

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/panel_test.go`:

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run "TestViewTasks" -v`
Expected: FAIL (compile error or missing panels — old `viewTasks` is single-column).

- [ ] **Step 3: Rewrite `viewTasks` and extract helpers in `internal/tui/tasks.go`**

Replace the existing `viewTasks` function with:

```go
func (m Model) viewTasks() string {
	if m.width > 0 && m.width < minWorkspaceWidth {
		return m.viewTasksNarrow()
	}
	return m.viewWorkspace()
}

func (m Model) viewWorkspace() string {
	// Left panel: projects.
	left := panel("Projects", m.projectsBody(), m.focus == focusProjects,
		projectsPanelWidth, m.height-1)

	rightW := m.width - projectsPanelWidth
	tasksPanelH := m.height - detailPanelHeight - 1

	// Tasks panel (scrolling viewport).
	tvp := m.taskVP
	tvp.SetContent(m.taskListBody())
	tasksPanel := panel("Tasks · "+m.activeProject().Name, tvp.View(),
		m.focus == focusTasks, rightW, tasksPanelH)

	// Details panel (scrolling viewport or notes editor).
	var detailContent string
	if m.notesEditing {
		detailContent = m.notesArea.View()
	} else if t, ok := m.selectedTask(); ok {
		dvp := m.detailVP
		dvp.SetContent(m.detailBody(t))
		detailContent = dvp.View()
	} else {
		detailContent = faintStyle.Render("No task selected.")
	}
	detailPanel := panel("Details", detailContent, false, rightW, detailPanelHeight)

	right := lipgloss.JoinVertical(lipgloss.Left, tasksPanel, detailPanel)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	return body + "\n" + helpStyle.Render(m.tasksHelp())
}

func (m Model) tasksHelp() string {
	if m.notesEditing {
		return "editing notes · ctrl+s save · esc cancel"
	}
	return "tab focus · j/k move · a add · e edit · n notes · t timer · s summary · , settings · q"
}

// taskListBody renders the task rows for the active project.
func (m Model) taskListBody() string {
	var b strings.Builder
	running, _ := m.store.RunningEntry()
	for i, t := range m.tasks {
		cursor := "  "
		if m.focus == focusTasks && i == m.cursor {
			cursor = "▸ "
		}
		box := "[ ]"
		if t.Done {
			box = "[x]"
		}
		clock := ""
		if running != nil && running.TaskID == t.ID {
			clock = " ⏱"
		}
		suffix := ""
		if d, ok := m.elapsedFor(t.ID); ok {
			suffix = "  (" + fmtDuration(d) + ")"
		}
		line := fmt.Sprintf("%s%s %s%s%s", cursor, box, t.Title, suffix, clock)
		switch {
		case m.focus == focusTasks && i == m.cursor:
			line = selectedStyle.Render(line)
		case t.Done:
			line = doneStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(m.tasks) == 0 {
		b.WriteString(faintStyle.Render("No tasks yet — press 'a' to add one."))
	}
	if m.editing {
		verb := "New task"
		if m.addingProject {
			verb = "New project"
		} else if m.editID != 0 {
			verb = "Edit task"
		}
		b.WriteString("\n" + verb + ": " + m.input.View())
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	return b.String()
}

// viewTasksNarrow is the single-column fallback for terminals below
// minWorkspaceWidth: the task list plus help, restyled with the new palette.
func (m Model) viewTasksNarrow() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("what was next — " + m.activeProject().Name))
	b.WriteString("\n")
	b.WriteString(m.taskListBody())
	b.WriteString(helpStyle.Render("\n" + m.tasksHelp()))
	return b.String()
}
```

Add `"charm.land/lipgloss/v2"` to the imports of `tasks.go`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestViewTasks" -v`
Expected: PASS. Then `go build ./...` — Expected: builds.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/tasks.go internal/tui/panel_test.go
git commit -m "Compose three-panel tasks workspace with narrow fallback"
```

---

### Task 5: Focus switching and folded project navigation

**Files:**
- Modify: `internal/tui/tasks.go` (`updateTasks`, input commit), `internal/tui/summary.go` (`p` handler)
- Test: `internal/tui/tasks_test.go`, `internal/tui/projects_test.go`

**Interfaces:**
- Consumes: `focusArea`, `addingProject`, store `CreateProject`/`CreateTask`/`reloadProjects`/`reloadTasks`.
- Produces:
  - `updateTasks` routes `tab`/`shift+tab` to toggle `m.focus`; list keys act on the focused panel.
  - `func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd)` — unified commit: creates a project when `m.addingProject`, else creates/updates a task.

- [ ] **Step 1: Write the failing tests**

Replace `internal/tui/projects_test.go` placeholder with:

```go
package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestProjectsPanelFocusAndSelect(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	p, _ := m.store.CreateProject("Work")
	m.store.CreateTask(p.ID, "Work task")
	m.reloadProjects()

	// Tab to focus the projects panel.
	mi, _ := m.updateTasks(tea.KeyPressMsg{Code: tea.KeyTab})
	m = mi.(Model)
	if m.focus != focusProjects {
		t.Fatal("want projects focused after tab")
	}
	// Move to "Work" (index 1) and select it.
	mi, _ = m.updateTasks(key('j'))
	m = mi.(Model)
	mi, _ = m.updateTasks(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	if m.activeProject().Name != "Work" {
		t.Fatalf("want active 'Work', got %q", m.activeProject().Name)
	}
	if m.focus != focusTasks {
		t.Fatal("want focus back on tasks after selecting a project")
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Work task" {
		t.Fatalf("want Work's tasks, got %+v", m.tasks)
	}
}

func TestAddProjectViaPanel(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.focus = focusProjects
	mi, _ := m.updateTasks(key('a'))
	m = mi.(Model)
	if !m.editing || !m.addingProject {
		t.Fatal("want project-add input active")
	}
	for _, r := range "Side" {
		mi, _ = m.Update(key(r))
		m = mi.(Model)
	}
	mi, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m = mi.(Model)
	found := false
	for _, p := range m.projects {
		if p.Name == "Side" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want 'Side' project created")
	}
}
```

Update `TestSwitchToProjectsAndSummary` in `internal/tui/tasks_test.go` to reflect `p` → focus (not screen):

```go
func TestSwitchToProjectsFocusAndSummary(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.updateTasks(key('p'))
	if mi.(Model).focus != focusProjects {
		t.Fatal("want projects focused after 'p'")
	}
	mi, _ = m.updateTasks(key('s'))
	if mi.(Model).screen != screenSummary {
		t.Fatal("want screenSummary after 's'")
	}
}
```

Delete the old `TestSwitchToProjectsAndSummary` (the `screenProjects` version).

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestProjectsPanel|TestAddProjectViaPanel|TestSwitchToProjectsFocus" -v`
Expected: FAIL — focus/`addingProject` routing not implemented.

- [ ] **Step 3: Implement focus routing in `updateTasks`**

Replace the body of `updateTasks` (the non-editing branch) with focus-aware routing. Full function:

```go
func (m Model) updateTasks(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	if m.notesEditing {
		return m.updateNotes(msg) // added in Task 6
	}
	if m.editing {
		return m.updateInput(msg)
	}
	m.status = ""
	switch msg.String() {
	case "tab":
		m.toggleFocus()
		return m, nil
	case "shift+tab":
		m.toggleFocus()
		return m, nil
	case "s":
		m.summaryPeriod = periodDay
		m.loadSummary()
		m.screen = screenSummary
		return m, nil
	case ",":
		m.screen = screenSettings
		return m, nil
	}
	if m.focus == focusProjects {
		return m.updateProjectsPanel(msg)
	}
	return m.updateTasksPanel(msg)
}

func (m *Model) toggleFocus() {
	if m.focus == focusTasks {
		m.focus = focusProjects
		m.projCursor = m.active
	} else {
		m.focus = focusTasks
	}
}

// updateProjectsPanel handles keys when the Projects panel is focused.
func (m Model) updateProjectsPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.projCursor < len(m.projects)-1 {
			m.projCursor++
		}
	case "k", "up":
		if m.projCursor > 0 {
			m.projCursor--
		}
	case "enter", "space":
		m.active = m.projCursor
		m.cursor = 0
		m.setStatus(m.reloadTasks())
		m.focus = focusTasks
	case "a":
		m.beginInput(0, "", true)
		return m, textinput.Blink
	}
	return m, nil
}

// updateTasksPanel handles keys when the Tasks panel is focused.
func (m Model) updateTasksPanel(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}
	case "a":
		m.beginInput(0, "", false)
		return m, textinput.Blink
	case "e":
		if t, ok := m.selectedTask(); ok {
			m.beginInput(t.ID, t.Title, false)
			return m, textinput.Blink
		}
	case "n":
		return m.beginNotes() // added in Task 6
	case "enter", "space":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.SetTaskDone(t.ID, !t.Done))
			m.setStatus(m.reloadTasks())
		}
	case "d":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.DeleteTask(t.ID))
			m.setStatus(m.reloadTasks())
		}
	case "J":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.MoveTask(t.ID, 1))
			m.setStatus(m.reloadTasks())
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		}
	case "K":
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.MoveTask(t.ID, -1))
			m.setStatus(m.reloadTasks())
			if m.cursor > 0 {
				m.cursor--
			}
		}
	case "t":
		if t, ok := m.selectedTask(); ok {
			m.toggleTimer(t.ID)
		}
	case "p":
		m.focus = focusProjects
		m.projCursor = m.active
	}
	return m, nil
}
```

Replace `beginEdit` with a `beginInput` that carries the project/task flag, and add the unified commit `updateInput` (replaces `updateTaskInput`):

```go
func (m *Model) beginInput(id int64, initial string, project bool) {
	ti := textinput.New()
	ti.SetValue(initial)
	ti.Focus()
	ti.CursorEnd()
	m.input = ti
	m.editing = true
	m.editID = id
	m.addingProject = project
}

func (m Model) updateInput(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.Code {
	case tea.KeyEnter:
		val := strings.TrimSpace(m.input.Value())
		if val != "" {
			if m.addingProject {
				_, err := m.store.CreateProject(val)
				m.setStatus(err)
				m.setStatus(m.reloadProjects())
			} else if m.editID == 0 {
				_, err := m.store.CreateTask(m.activeProject().ID, val)
				m.setStatus(err)
				m.setStatus(m.reloadTasks())
			} else {
				m.setStatus(m.store.UpdateTask(m.editID, val, m.notesOf(m.editID)))
				m.setStatus(m.reloadTasks())
			}
		}
		m.editing = false
		m.addingProject = false
		return m, nil
	case tea.KeyEscape:
		m.editing = false
		m.addingProject = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// notesOf returns the current notes for a task id (preserved when editing the
// title so UpdateTask does not clobber them).
func (m Model) notesOf(id int64) string {
	for _, t := range m.tasks {
		if t.ID == id {
			return t.Notes
		}
	}
	return ""
}
```

Delete the old `beginEdit` and `updateTaskInput`. In `summary.go` `updateSummary`, delete the `case "p":` block (projects are no longer a screen).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: PASS for all (Task 6 funcs `updateNotes`/`beginNotes` are referenced — add temporary stubs if running before Task 6, see note). To keep this task self-contained, add minimal stubs now at the bottom of `tasks.go`:

```go
// Notes editing is implemented in Task 6; stubs keep the package compiling.
func (m Model) updateNotes(tea.KeyPressMsg) (tea.Model, tea.Cmd) { m.notesEditing = false; return m, nil }
func (m Model) beginNotes() (tea.Model, tea.Cmd)                 { return m, nil }
```

- [ ] **Step 5: Commit**

```bash
git add internal/tui/tasks.go internal/tui/summary.go internal/tui/tasks_test.go internal/tui/projects_test.go
git commit -m "Add panel focus switching and folded project navigation"
```

---

### Task 6: Editable notes

**Files:**
- Modify: `internal/tui/tasks.go` (replace the Task 5 stubs)
- Test: `internal/tui/details_test.go` (add)

**Interfaces:**
- Consumes: `notesArea textarea.Model`, `notesEditing`, `selectedTask`, `store.UpdateTask`.
- Produces:
  - `func (m Model) beginNotes() (tea.Model, tea.Cmd)` — seeds `notesArea` with the selected task's notes, focuses it, sets `notesEditing`.
  - `func (m Model) updateNotes(msg tea.KeyPressMsg) (tea.Model, tea.Cmd)` — `ctrl+s` saves via `UpdateTask` (title preserved) and reloads; `esc` cancels; otherwise forwards to the textarea.

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/details_test.go`:

```go
import tea "charm.land/bubbletea/v2" // add to imports

func TestNotesEditSaves(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()

	mi, _ := m.updateTasks(key('n'))
	m = mi.(Model)
	if !m.notesEditing {
		t.Fatal("want notesEditing after 'n'")
	}
	for _, r := range "hello" {
		mi, _ = m.updateTasks(key(r))
		m = mi.(Model)
	}
	mi, _ = m.updateTasks(tea.KeyPressMsg{Code: 's', Mod: tea.ModCtrl})
	m = mi.(Model)
	if m.notesEditing {
		t.Fatal("want editing off after ctrl+s")
	}
	if m.tasks[0].Notes != "hello" {
		t.Fatalf("want notes 'hello', got %q", m.tasks[0].Notes)
	}
}

func TestNotesEditCancel(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	mi, _ := m.updateTasks(key('n'))
	m = mi.(Model)
	mi, _ = m.updateTasks(key('x'))
	m = mi.(Model)
	mi, _ = m.updateTasks(tea.KeyPressMsg{Code: tea.KeyEscape})
	m = mi.(Model)
	if m.tasks[0].Notes != "" {
		t.Fatalf("want notes unchanged on cancel, got %q", m.tasks[0].Notes)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run "TestNotesEdit" -v`
Expected: FAIL — stubs don't save.

- [ ] **Step 3: Replace the stubs in `internal/tui/tasks.go`**

```go
func (m Model) beginNotes() (tea.Model, tea.Cmd) {
	t, ok := m.selectedTask()
	if !ok {
		return m, nil
	}
	ta := textarea.New()
	ta.SetWidth(m.detailVP.Width())
	ta.SetHeight(m.detailVP.Height())
	ta.SetValue(t.Notes)
	cmd := ta.Focus()
	m.notesArea = ta
	m.notesEditing = true
	return m, cmd
}

func (m Model) updateNotes(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.Code == 's' && msg.Mod == tea.ModCtrl:
		if t, ok := m.selectedTask(); ok {
			m.setStatus(m.store.UpdateTask(t.ID, t.Title, m.notesArea.Value()))
			m.setStatus(m.reloadTasks())
		}
		m.notesEditing = false
		m.notesArea.Blur()
		return m, nil
	case msg.Code == tea.KeyEscape:
		m.notesEditing = false
		m.notesArea.Blur()
		return m, nil
	}
	var cmd tea.Cmd
	m.notesArea, cmd = m.notesArea.Update(msg)
	return m, cmd
}
```

Add `"charm.land/bubbles/v2/textarea"` to `tasks.go` imports.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestNotesEdit" -v`
Expected: PASS. Then full suite `go test ./...` — Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/tasks.go internal/tui/details_test.go
git commit -m "Add editable per-task notes in the Details panel"
```

---

### Task 7: Scroll the task list to follow the cursor

**Files:**
- Modify: `internal/tui/tasks.go` (viewport y-offset sync)
- Test: `internal/tui/panel_test.go` (add)

**Interfaces:**
- Consumes: `taskVP viewport.Model`, `taskListBody`, `m.cursor`.
- Produces:
  - `func (m *Model) syncTaskScroll()` — sets `taskVP` content and scrolls so the selected row stays visible; called after cursor moves and before rendering.

- [ ] **Step 1: Write the failing test**

Add to `internal/tui/panel_test.go`:

```go
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
	if m.taskVP.YOffset == 0 {
		t.Fatal("want task viewport scrolled to keep cursor visible")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestTaskScrollFollowsCursor -v`
Expected: FAIL — `YOffset` stays 0 (nothing syncs scroll).

- [ ] **Step 3: Add `syncTaskScroll` and call it after cursor moves**

Add to `tasks.go`:

```go
// syncTaskScroll refreshes the task viewport content and scrolls so the
// selected row (m.cursor) stays within the visible window.
func (m *Model) syncTaskScroll() {
	m.taskVP.SetContent(m.taskListBody())
	h := m.taskVP.Height()
	if h < 1 {
		return
	}
	top := m.taskVP.YOffset
	if m.cursor < top {
		m.taskVP.SetYOffset(m.cursor)
	} else if m.cursor >= top+h {
		m.taskVP.SetYOffset(m.cursor - h + 1)
	}
}
```

In `updateTasksPanel`, after each `m.cursor` change (the `j`/`k`/`J`/`K` cases), call `m.syncTaskScroll()`. Simplest: call it once at the end of `updateTasksPanel` before `return m, nil`:

```go
	}
	m.syncTaskScroll()
	return m, nil
}
```

In `viewWorkspace`, the Tasks panel already calls `tvp.SetContent(m.taskListBody())` on a copy; change it to use the synced offset by reading from `m.taskVP` directly:

```go
	tasksPanel := panel("Tasks · "+m.activeProject().Name, m.taskVP.View(),
		m.focus == focusTasks, rightW, tasksPanelH)
```

and drop the local `tvp` copy (content is now set in `syncTaskScroll`). Ensure `syncTaskScroll` is also called after `reloadTasks` in New/select paths (add `m.syncTaskScroll()` at the end of `updateProjectsPanel`'s `enter` case, guarded by `m.width > 0`).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run TestTaskScrollFollowsCursor -v`
Expected: PASS. Then `go test ./...` — Expected: PASS.

- [ ] **Step 5: Visual smoke test**

Run: `go build -o what-was-next . && ./what-was-next`
Verify by eye: three panels render, `tab` moves the accent border, `j/k` scroll a long task list, `n` opens the notes editor, `ctrl+s` saves, colours look soft on your Ghostty dark theme. Resize below ~80 cols → single-column fallback. Press `q` to quit.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/tasks.go internal/tui/panel_test.go
git commit -m "Scroll task list viewport to follow the cursor"
```

---

## Self-Review

**Spec coverage:**
- §1 panelled workspace → Tasks 3–4. §2 Details (status/timestamps/time/notes) → Tasks 3, 6. §3 focus/nav → Task 5. §4 Summary/Settings unchanged → untouched (verified: only the `p` case removed from summary). §5 palette → Task 1. §6 narrow fallback → Task 4. Scrolling → Tasks 2, 7. All covered.

**Placeholder scan:** No TBD/TODO; every code step is complete. The Task 5 stubs are explicitly temporary and replaced in Task 6 with a stated reason.

**Type consistency:** `beginInput(id, initial, project)` used consistently (Task 5); `updateInput` replaces `updateTaskInput` everywhere; `focusTasks`/`focusProjects` consistent; `syncTaskScroll` signature stable; `panel(title, body, focused, width, height)` identical across Tasks 1/4/7; viewport methods (`SetWidth/SetHeight/SetContent/SetYOffset/YOffset/View/Width/Height`) match `bubbles/v2` v2.1.1; textarea methods (`SetWidth/SetHeight/SetValue/Value/Focus/Blur/Update/View`) match. `tea.ModCtrl` used for ctrl+s detection.

**Note for implementer:** exact panel geometry (border/title row accounting) may need a one-line tune during Task 7's visual smoke test; adjust `detailPanelHeight` or the `-1` help-row reservations if a panel is one row too tall. Tests assert behaviour and width bounds, not exact glyph positions.
```
