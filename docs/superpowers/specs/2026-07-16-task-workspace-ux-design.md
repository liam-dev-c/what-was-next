# Task workspace UX: completed-fade + Details focus

Date: 2026-07-16

Two independent improvements to the tasks workspace. They share files
(`internal/tui/tasks.go`, `details.go`, `app.go`) but ship on separate branches.

## Feature A — Completed tasks fade at midnight

### Behaviour
- Open tasks render first, in their manual `sort_order`.
- Completed tasks render below a faint divider, sorted by completion time,
  most-recent first.
- By default only tasks completed **today** (local calendar day) appear in the
  completed group. Anything completed on a prior day is hidden once local
  midnight passes.
- `c` toggles "show all completed" for the session (default off). When on, the
  completed group shows every done task, still ordered by completion time
  newest-first.

### Why no timers / migration
The TUI re-renders every second and recomputes the view from `time.Now()`, so
"today" rolls over at midnight for free. Filtering and sorting happen entirely
in the TUI view layer using the `DoneAt` field the store already returns.
`internal/store` and the MCP server are untouched.

### Implementation
- New `Model` field `showAllCompleted bool` (session state).
- New helper `visibleTasks() (tasks []store.Task, doneStart int)`:
  - partitions `m.tasks` into open (kept in `sort_order`) and completed,
  - drops completed tasks whose `DoneAt` is before local start-of-today unless
    `showAllCompleted`,
  - stable-sorts the completed group by `DoneAt` descending,
  - returns `open ++ completed` and the index where the completed group starts
    (`len(open)`; equals `len(tasks)` when there are none).
- `m.cursor` now indexes into `visibleTasks()`, not `m.tasks`.
  - `selectedTask()` reads `visibleTasks()[cursor]`.
  - Cursor clamping (in `reloadTasks`, `reloadPreservingSelection`, after `c`)
    uses the visible count.
  - `j`/`k` bounds use the visible count.
- `taskListBody()` iterates the visible slice and emits a faint divider line
  before the element at `doneStart` (only when a completed group exists).
- `c` handler flips `showAllCompleted` then re-clamps the cursor and calls
  `syncTaskScroll`.

## Feature B — Three-section focus, edit-in-Details

### Behaviour
- `tab` / `shift+tab` cycle focus **Projects → Tasks → Details** (and reverse).
- Tasks list is navigation-only: `j/k` select · `a` add · `J/K` reorder ·
  `c` toggle completed · `enter` jump into Details for the selected task.
- Details owns the whole selected task once focused: `e` edit title · `n` notes ·
  `g` tags · `enter`/`space` toggle done · `t` timer · `d` delete · `j/k` scroll
  notes · `esc` back to Tasks. Delete returns focus to Tasks.
- Details' `(n to edit)` / `(g to edit)` hints render only while Details is
  focused. The Details panel border highlights when focused.
- Help line is focus-aware: each panel shows only its own keys.

### Implementation
- Add `focusDetails` to the `focusArea` enum.
- Replace `toggleFocus` with `cycleFocus(dir int)` cycling the three states;
  wire `tab` forward and `shift+tab` backward in `updateTasks`.
- `updateTasksPanel` keeps only `j/k`, `a`, `J/K`, `c`, `enter`→Details.
  Remove `e/n/g/enter-done/d/t/p`.
- New `updateDetailsPanel` handles `e/n/g/enter/space/t/d/j/k/esc` on the
  selected task; `d` returns focus to Tasks.
- `viewWorkspace` passes `m.focus == focusDetails` to the Details `panel(...)`.
- `detailBody` takes a `focused bool` so the edit hints are conditional.
- `tasksHelp` returns per-focus key lists.
- Drop the `p` shortcut.

### Deferred polish (not in scope)
When editing title/tags from Details, the text input still renders in its
current spot under the task list rather than inside the Details panel. Moving it
into Details is a later polish.

## Testing
- `visibleTasks` partition/filter/sort: pure-ish logic, unit-tested with tasks
  whose `DoneAt` is today, yesterday, and null; assert order and the
  `showAllCompleted` toggle.
- Divider appears only when a completed group is present.
- `cycleFocus` transitions in both directions across all three states.
- Details-focus actions: `e/n/g` enter their edit modes; `enter` toggles done;
  `d` deletes and returns focus to Tasks.
- Update existing tests that encode old behaviour: `TestToggleDone` and
  `TestDeleteTask` (now require Details focus), `TestSwitchToProjectsFocusAndHistory`
  (`p` removed → use `tab`).

## Branches
- `feature/completed-fade` — Feature A.
- `feature/details-focus` — Feature B.
Each merges into `main` and is deleted, per the repo workflow.
