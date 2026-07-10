# Design: panelled task workspace + softer dark palette

Date: 2026-07-10

## Goal

Make the tasks screen "nicer" with a multi-panel layout — a persistent
Projects sidebar on the left, and a Tasks + Details split on the right — and
lighten the colour palette so it reads well in a dark-mode Ghostty terminal.
The Details panel gives tasks a home for richer information (notably editable
notes, which the store already supports but the UI never surfaces).

## Current state

The TUI (`internal/tui`, Bubble Tea v2 + Lipgloss v2) is a single full-screen
router: `Model.View` picks one of `viewTasks` / `viewProjects` / `viewSummary`
/ `viewSettings` and that screen fills the terminal. Navigation is modal —
`p` jumps to a Projects screen, `s` to Summary, `,` to Settings, `esc`/`t`
back. Styling lives in `styles.go` as a handful of ANSI-256 colours (magenta
`212`, dim grey `240`, red `203`).

`store.Task` already has a `Notes` field and `UpdateTask(id, title, notes)`
persists it, but the UI always passes `""` and never displays notes.

## Design

### 1. Panelled tasks workspace

The tasks screen becomes a three-panel layout:

```
┌ Projects ─────┐┌ Tasks · Work ───────────────────────┐
│▸ Work         ││ ▸ [ ] Ship release          1h02m ⏱ │
│  Personal     ││   [x] Write changelog               │
│  Errands      ││   [ ] Fix timer bug                 │
│               │└─────────────────────────────────────┘
│               │┌ Details ────────────────────────────┐
│               ││ Ship release              ● open    │
│               ││ tracked 1h02m · timer running (t)   │
│               ││ created 9 Jul                       │
│  + add (a)    ││ notes:  Cut v2 tag, run release (n) │
└───────────────┘└─────────────────────────────────────┘
 tab focus · j/k move · a add · e edit · n notes · t timer · s summary · , settings · q
```

- **Left — Projects:** fixed width (~22 cols). Lists projects; the active one
  is marked. `a` adds a project (existing input flow).
- **Right column** splits vertically:
  - **Tasks (top):** the task list for the active project, with checkbox,
    title, tracked time, and a live ⏱ marker when a timer runs — as today.
    **Vertically scrollable** when the list is taller than the panel: the
    selected task is kept in view (the viewport follows the cursor).
  - **Details (bottom):** reflects the currently-selected task (see §2).
    **Vertically scrollable** when the notes/content exceed the panel height.

Panels are Lipgloss bordered boxes laid out with `lipgloss.JoinHorizontal` /
`JoinVertical`, sized from `m.width` / `m.height`.

### 2. Details panel

For the selected task it shows all four:

- **Status:** ● open / ✓ done.
- **Timestamps:** created date, and completed date when done.
- **Time tracked:** total via `elapsedFor`, with a live ⏱ / "timer running"
  indicator when its timer is active.
- **Notes:** the task's `Notes`. Press `n` to edit inline using a
  `bubbles/v2/textarea` (multi-line). **`ctrl+s` saves** (via `UpdateTask`,
  preserving the title), **`esc` cancels**. This is the primary "add more
  info to a task" capability.

### 3. Focus & navigation

- Two focusable panels: **Projects** and **Tasks**. `tab` / `shift+tab`
  toggles focus between them. The focused panel gets an **accent-coloured
  border**; the unfocused panel a dim border.
- `j/k` (and arrows) move the cursor within the **focused** panel.
- Selecting a task (moving the Tasks cursor) updates the Details panel live.
  When the Tasks list is longer than its panel, moving the cursor past the
  edge scrolls the viewport to keep the selection visible.
- In the Details panel, when notes/content overflow, they scroll vertically —
  while editing notes the `textarea` scrolls with the cursor; when viewing, the
  content is shown in a scrolling `viewport` that follows the selected task.
- Choosing a project (`enter` in Projects, or moving its cursor) makes it
  active and refreshes the Tasks list. Existing per-task keys (`e` edit,
  `enter`/`space` done, `d` delete, `J/K` move, `t` timer) act on the Tasks
  panel as today.
- `n` enters notes-edit mode in the Details panel (Tasks focused, task
  selected).

The legacy `p` key (jump to a standalone Projects screen) is retired in favour
of `tab` focus; the standalone `screenProjects` view is no longer needed for
normal use, though project **adding** still uses the shared text-input flow.

### 4. Summary & Settings unchanged

`s` (Summary, with its day/week toggle) and `,` (Settings) remain
**full-screen views**; `esc` returns to the panelled tasks workspace. This
keeps the change scoped to the tasks workspace and preserves existing summary
and settings behaviour and tests.

### 5. Palette (soft truecolor)

Centralise named colours in `styles.go` and replace the ANSI indices with hex
tuned for a dark background:

| role                                   | old          | new               |
|----------------------------------------|--------------|-------------------|
| accent (title / selection / focus border) | `212`     | `#C8A2FF` lavender |
| dim text                               | `240` (dark) | `#9BA3B4` light slate |
| panel border (unfocused)               | —            | `#4A4E5A` muted grey |
| error / status                         | `203`        | `#F2A0A0` rose    |
| success (done / timer)                 | —            | `#A7E0B8` mint    |
| faint (labels / hints)                 | `240`        | `#6B7280` slate   |

### 6. Graceful degradation

The three-panel layout needs width. Below a minimum (~80 cols) it **falls back
to today's single-column task list**, restyled with the new palette. No
horizontal scrolling; panels never overflow the terminal width. Very short
terminals shrink the Details panel first, then the panel content clips rather
than pushing the help line off-screen.

## Components touched

- `internal/tui/styles.go` — named palette vars (hex), panel/border styles,
  focused vs unfocused border styles.
- `internal/tui/tasks.go` — split into task-list rendering + the new panel
  composition; focus state; notes-edit mode; task-list `viewport` scrolling.
- `internal/tui/app.go` — `Model` gains `focus` (projects|tasks), notes editor
  state, and `bubbles/v2/viewport` models for the Tasks and Details panels
  (resized on `WindowSizeMsg`); `viewTasks` composes the three panels and
  dispatches the narrow-terminal fallback.
- `internal/tui/projects.go` — projects rendered as the left panel; keep the
  add-project input flow.
- New: a small detail-panel render helper (in `tasks.go` or a new
  `details.go`).

## Non-goals

- No change to Summary or Settings layout/behaviour.
- No new store schema (Notes already exists).
- No mouse support, no resizable panels, no theming/light-mode toggle.

## Testing

- Palette/style constants render without panic; focused vs unfocused borders
  differ.
- Notes edit: `n` opens editor, typed text + `ctrl+s` persists via
  `UpdateTask` and shows in Details; `esc` discards.
- Focus: `tab` moves focus; `j/k` act on the focused panel only.
- Scrolling: a task list taller than its panel keeps the selected task in
  view as the cursor moves past the bottom/top edge; long notes overflow into
  a scrolling region rather than pushing panel borders off-screen.
- Narrow width (`WindowSizeMsg` < 80 cols) renders the single-column fallback
  without error and without exceeding the width.
- Existing tasks/projects/summary/settings tests continue to pass (adjusted
  for the retired `p`-to-screen navigation where needed).
```
