# History Button + Panel Balance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax.

**Goal:** Follow-up tweaks to the panel workspace: rebalance the right column to Details ~60% / Tasks ~40%, make Tasks the landing screen, and demote the Summary screen to a "History" view opened by an `h` key.

**Architecture:** Same single-`Model` Bubble Tea v2 design. Replace the fixed `detailPanelHeight` split with a proportional one computed by a shared helper (used by both `resizePanels` and `viewWorkspace` so they cannot diverge). Rename the `screenSummary` screen to `screenHistory`, change its access key from `s` to `h`, and set the initial screen to `screenTasks`. The existing summary store queries and day/week toggle are reused unchanged.

**Tech Stack:** Go, charm.land/bubbletea/v2, charm.land/bubbles/v2 (viewport, textarea), charm.land/lipgloss/v2, SQLite store.

## Global Constraints

- Module: github.com/liam-dev-c/what-was-next; package internal/tui.
- Right column (below the Projects sidebar) splits: Tasks ~40%, Details ~60% of the available height (after reserving 1 row for the help line). Both clamp to a sensible minimum so tiny terminals never produce zero/negative heights.
- The initial screen is `screenTasks` (was `screenSummary`).
- History is opened with `h` from the tasks screen; `esc` returns to tasks. The day/week toggle (`d`/`w`) is preserved. No "Summary" wording remains in user-facing help; no stale "p projects" hint.
- Keep `go build ./...` and `go test ./...` green before each commit.
- Git commits must NOT include a `Co-Authored-By` trailer.
- Branch panel-layout.

---

## Task A: Proportional right-column heights (Tasks 40% / Details 60%)

**Files:**
- Modify: `internal/tui/styles.go` (remove `detailPanelHeight` const, add ratio helper or const)
- Modify: `internal/tui/app.go` (`resizePanels`)
- Modify: `internal/tui/tasks.go` (`viewWorkspace`)
- Test: `internal/tui/panel_test.go`

**Interfaces:**
- Produces: `func (m Model) rightColumnHeights() (tasksH, detailH int)` — the single source of truth for the right column split, used by both `resizePanels` and `viewWorkspace`. Given `rightColH := m.height - 1` (help row), `detailH := max(3, rightColH*3/5)`, `tasksH := max(3, rightColH-detailH)`.

- [ ] **Step 1: Write the failing test**

```go
func TestRightColumnDetailsLargerThanTasks(t *testing.T) {
	m := newModel(t)
	mi, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = mi.(Model)
	tasksH, detailH := m.rightColumnHeights()
	if detailH <= tasksH {
		t.Fatalf("want Details taller than Tasks, got tasks=%d detail=%d", tasksH, detailH)
	}
	if tasksH+detailH > 40-1 {
		t.Fatalf("panels overflow height: tasks=%d detail=%d", tasksH, detailH)
	}
	// Detail viewport should be sized from detailH, not the old fixed 9.
	if m.detailVP.Height() <= 3 {
		t.Fatalf("detail viewport too small: %d", m.detailVP.Height())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestRightColumnDetailsLargerThanTasks -v`
Expected: FAIL — `m.rightColumnHeights undefined`.

- [ ] **Step 3: Add the helper and rewire sizing**

In `styles.go`, delete `detailPanelHeight = 9` (keep `projectsPanelWidth`, `minWorkspaceWidth`).

In `app.go`, add:

```go
// rightColumnHeights splits the right column (below the help row) into the
// Tasks panel (~40%) over the Details panel (~60%). Single source of truth so
// resizePanels and viewWorkspace never disagree.
func (m Model) rightColumnHeights() (tasksH, detailH int) {
	rightColH := m.height - 1 // reserve 1 row for the help line
	if rightColH < 6 {
		rightColH = 6
	}
	detailH = rightColH * 3 / 5
	if detailH < 3 {
		detailH = 3
	}
	tasksH = rightColH - detailH
	if tasksH < 3 {
		tasksH = 3
	}
	return tasksH, detailH
}
```

Rewrite `resizePanels` to use it:

```go
func (m *Model) resizePanels() {
	rightW := m.width - projectsPanelWidth
	if rightW < 1 {
		rightW = 1
	}
	innerW := rightW - 2
	if innerW < 1 {
		innerW = 1
	}
	tasksPanelH, detailPanelH := m.rightColumnHeights()
	taskInner := tasksPanelH - 2 - 1 // borders + title
	if taskInner < 1 {
		taskInner = 1
	}
	detailInner := detailPanelH - 2 - 1
	if detailInner < 1 {
		detailInner = 1
	}
	m.taskVP.SetWidth(innerW)
	m.taskVP.SetHeight(taskInner)
	m.detailVP.SetWidth(innerW)
	m.detailVP.SetHeight(detailInner)
	m.notesArea.SetWidth(innerW)
	m.notesArea.SetHeight(detailInner)
}
```

In `tasks.go` `viewWorkspace`, replace the height computation. Where it currently does `tasksPanelH := m.height - detailPanelHeight - 1` and passes `detailPanelHeight` to the Details panel, use:

```go
	tasksPanelH, detailPanelH := m.rightColumnHeights()
```

and pass `detailPanelH` (not the deleted const) to the `panel("Details", ...)` call, and `tasksPanelH` to the Tasks panel as before. The left Projects panel height stays `m.height - 1`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestRightColumn|TestViewTasks|TestWindowSize" -v`
Expected: PASS. Then `go build ./...` and `go test ./...` — Expected: green.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/styles.go internal/tui/app.go internal/tui/tasks.go internal/tui/panel_test.go
git commit -m "Rebalance right column to Details 60% / Tasks 40%"
```

---

## Task B: Tasks as landing screen; Summary → History behind `h`

**Files:**
- Modify: `internal/tui/app.go` (initial screen; rename `screenSummary`→`screenHistory`)
- Modify: `internal/tui/tasks.go` (`s`→`h` binding; `tasksHelp`)
- Modify: `internal/tui/summary.go` (help text, esc/return, remove stale "p projects", user-facing "History")
- Modify: tests referencing `screenSummary`/`s` navigation

**Interfaces:**
- Produces: `screenHistory` screen constant (replaces `screenSummary`); `h` opens it from the tasks screen; initial screen is `screenTasks`.

- [ ] **Step 1: Write/adjust the failing tests**

In `tasks_test.go`, add:

```go
func TestHistoryKeyOpensHistory(t *testing.T) {
	m := newModel(t)
	m.screen = screenTasks
	mi, _ := m.updateTasks(key('h'))
	if mi.(Model).screen != screenHistory {
		t.Fatal("want screenHistory after 'h'")
	}
}
```

Update `TestNewSelectsDefaultProject` in `app_test.go` to expect `screenTasks` as the initial screen (was `screenSummary`). Update any `summary_test.go` / other tests that reference `screenSummary` to `screenHistory`, and any that press `s` to open it to press `h`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/tui/ -run "TestHistoryKeyOpensHistory|TestNewSelectsDefaultProject" -v`
Expected: FAIL — `screenHistory` undefined / initial screen mismatch.

- [ ] **Step 3: Implement**

In `app.go`:
- In the `screen` const block, rename `screenSummary` → `screenHistory`.
- In `New`, change `screen: screenSummary` → `screen: screenTasks`. Keep `m.loadSummary()` priming (History still shows the daily snapshot on open; harmless to prime).
- In `Update`/`View` dispatch, rename the `case screenSummary:` branches to `case screenHistory:`.

In `tasks.go` `updateTasksPanel` (and any other `updateTasks` global handling): change the `case "s":` block that sets `m.screen = screenSummary` to `case "h":` setting `m.screen = screenHistory` (keep the `m.summaryPeriod = periodDay; m.loadSummary()` priming). Update `tasksHelp()` to replace `s summary` with `h history`.

In `summary.go`:
- `updateSummary`: keep `d`/`w` toggle; ensure `esc` (and drop `t` or keep) returns to `screenTasks`. Change any `m.screen = screenTasks` targets to remain tasks.
- `summaryHelp` const: replace with `"\nd day · w week · esc back · , settings · q quit"` (no "t tasks", no "p projects", no "summary").
- Titles: keep "Today — …" / "This week — …" (they are the history content). Optionally prefix nothing; user reaches it via History.

Grep to confirm no remaining `screenSummary` references: `grep -rn screenSummary internal/`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run "TestHistoryKeyOpensHistory|TestNewSelectsDefaultProject" -v`
Expected: PASS. Then `go build ./...` and full `go test ./...` — Expected: green. Confirm `grep -rn screenSummary internal/` returns nothing.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/app.go internal/tui/tasks.go internal/tui/summary.go internal/tui/app_test.go internal/tui/tasks_test.go internal/tui/summary_test.go
git commit -m "Land on Tasks; open Summary as History via h key"
```

---

## Self-Review

- **Spec coverage:** Details>Tasks split (Task A), landing=Tasks + `h` History (Task B), stale "p projects"/"summary" wording removed (Task B Step 3). Covered.
- **Placeholder scan:** none.
- **Type consistency:** `rightColumnHeights()` returns `(tasksH, detailH)` used identically in `resizePanels` and `viewWorkspace`; `screenHistory` replaces `screenSummary` everywhere (grep-verified in Step 4).
- **Note:** the day/week snapshot loading (`loadSummary`/`loadWeek`) and store queries are unchanged; only the screen name, access key, initial screen, and help text change.
