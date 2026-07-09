# what-was-next — TUI Design

A terminal UI task manager and time tracker: a simple take on Super Productivity.

## Purpose

Track tasks and the time spent on them from the terminal. Answer "what was I doing,
what's next" by keeping a task list per project, tracking time per task, and showing a
daily recap.

## Scope (v1)

- **Task list (CRUD)** — add, edit, complete, delete, reorder.
- **Time tracking** — start/stop a timer on a task; accumulate time spent.
- **Projects / grouping** — organize tasks under projects; switch the active project.
- **Daily summary / history** — today's completed tasks and time totals.

Out of scope for v1: pomodoro, sync, reminders/scheduling, sub-tasks, tags, themes
beyond a single built-in style.

## Stack

- `github.com/charmbracelet/bubbletea` — Elm-style event loop.
- `github.com/charmbracelet/bubbles` — list, textinput, table components.
- `github.com/charmbracelet/lipgloss` — styling.
- `modernc.org/sqlite` — pure-Go SQLite driver (no cgo, cross-compiles cleanly).
- Module path: `github.com/liam-dev-c/what-was-next`. Go 1.26.

## Package layout

```
what-was-next/
  main.go                  # entry: resolve data dir, open DB, run Bubble Tea program
  internal/
    store/                 # SQLite persistence + domain types
      store.go             # Open(), schema migration
      task.go              # Task CRUD, complete, reorder
      project.go           # Project CRUD
      timelog.go           # start/stop timer, time entries
      summary.go           # daily totals / history queries
    tui/
      app.go               # root model, screen routing, global keys
      tasks.go             # task-list screen (add/edit/complete/delete/reorder)
      projects.go          # project switcher
      timer.go             # active-timer view + start/stop
      summary.go           # daily recap screen
      styles.go            # lipgloss theme
```

## Data model (SQLite)

- `projects(id INTEGER PK, name TEXT NOT NULL, created_at TIMESTAMP)`
- `tasks(id INTEGER PK, project_id INTEGER FK, title TEXT NOT NULL, notes TEXT,
  done INTEGER, sort_order INTEGER, created_at TIMESTAMP, done_at TIMESTAMP NULL)`
- `time_entries(id INTEGER PK, task_id INTEGER FK, started_at TIMESTAMP,
  ended_at TIMESTAMP NULL)` — a row with `ended_at` NULL is the running timer;
  duration = `ended_at - started_at`.

Times stored as UTC. A default project is created on first run so the task list is
never orphaned.

## Storage location

Single database file at `~/.config/what-was-next/what-was-next.db` (respecting
`XDG_CONFIG_HOME` when set). The directory is created on first run.

## Screens & flow

- **Task list** is the home screen, scoped to the selected project.
- **Project switcher** changes the active scope.
- **Timer view** shows the currently running task and elapsed time.
- **Summary** shows today's completed tasks and time totals per task/project.

Key bindings (task list):

| Key       | Action                                  |
|-----------|-----------------------------------------|
| `a`       | add task                                |
| `e`       | edit selected task                      |
| `enter`/`space` | toggle done                       |
| `d`       | delete selected task                    |
| `J`/`K`   | reorder selected task down/up           |
| `t`       | start/stop timer on selected task       |
| `p`       | open project switcher                   |
| `s`       | open daily summary                      |
| `?`       | help                                    |
| `q`       | quit                                    |

Only one timer runs at a time; starting a timer on another task stops the current one.

## Data flow

SQLite is the single source of truth. Store methods take/return domain structs and own
all SQL. The TUI holds view state only and reloads the relevant slice from the store
after each mutation — no in-memory cache to drift out of sync. The store package has no
dependency on the TUI and is unit-tested directly against an in-memory / temp-file DB.

## Error handling

- Store `Open()` failures (bad path, migration error) abort startup with a clear message.
- Runtime store errors surface as a transient status-line message in the TUI; the app
  stays usable.
- A single running timer is enforced at the store layer (start stops any open entry in
  one transaction), so a crash mid-session can't leave two timers running.

## Testing

- `internal/store` unit tests against a temp-file SQLite DB: task CRUD, reorder,
  timer start/stop enforcement, summary aggregation.
- TUI models tested where practical by feeding messages to `Update` and asserting state
  transitions; heavy rendering left to manual verification.
