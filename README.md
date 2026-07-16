# what-was-next

A simple terminal task manager and time tracker — a small take on Super Productivity.

## Features

- Task list per project (add, edit, complete, delete, reorder)
- Tags on tasks (comma-separated; shown in the list and details)
- Per-task time tracking with a live timer
- Projects to group tasks
- Summary of completed tasks and time tracked — the landing screen — with a
  day/week toggle (weekly view includes a per-day breakdown)
- Settings, including which day the week starts on

## Install

```bash
go install github.com/liam-dev-c/what-was-next@latest
```

## Usage

Run `what-was-next`. Data is stored at `~/.config/what-was-next/what-was-next.db`
(honoring `XDG_CONFIG_HOME`).

The summary opens by default. It groups tasks by your **local** calendar day (or
week). Timestamps are stored in UTC and converted to your machine's timezone
when computing "today" and "this week". The week starts on the day chosen in
settings (Monday by default).

### Keys

The tasks screen has three panels — **Projects** (left), **Tasks** (top-right),
and **Details** (bottom-right). `tab` / `shift+tab` cycle focus between them.
Navigation lives in the Tasks list; everything that edits a single task lives in
Details, which you enter with `enter` (or `tab`).

Global (any panel):

| Key | Action |
|-----|--------|
| `tab` / `shift+tab` | cycle panel focus |
| `h` | history / summary |
| `,` | settings |
| `q` | quit |

Tasks panel (navigation):

| Key | Action |
|-----|--------|
| `j` / `k` | move selection |
| `enter` | open the task in Details |
| `a` | add task |
| `J` / `K` | reorder task down / up |
| `c` | show/hide completed tasks |

Completed tasks drop below a divider, newest-first, and hide once their
completion day has passed; `c` reveals every completed task.

Details panel (acts on the selected task):

| Key | Action |
|-----|--------|
| `e` | edit title |
| `n` | edit notes |
| `g` | edit tags (comma-separated) |
| `enter` / `space` | toggle done |
| `t` | start/stop timer |
| `d` | delete task |
| `j` / `k` | scroll notes |
| `esc` | back to the Tasks list |

Projects panel:

| Key | Action |
|-----|--------|
| `j` / `k` | move selection |
| `enter` | switch to project |
| `a` | add project |

History screen (open with `h`):

| Key | Action |
|-----|--------|
| `d` / `w` | day / week view |
| `esc` | back to tasks |
| `,` | settings |
| `q` | quit |

## Claude / MCP

what-was-next ships an MCP server so Claude can manage your projects and tasks.

Register it with Claude Code (one time):

```bash
what-was-next mcp install
```

This runs `claude mcp add` under the hood. Pass `--scope project` or
`--scope local` to change where it's registered (default `user`). If the
`claude` CLI isn't installed, the command prints the equivalent registration
command to run manually. Restart Claude Code afterward.

The server exposes these tools:

- Projects: `list_projects`, `create_project`, `rename_project`, `delete_project`
- Tasks: `list_tasks`, `create_task` (optional `tags`), `update_task`,
  `set_task_done`, `set_task_tags`, `move_task`, `delete_task`

`delete_project` also deletes that project's tasks.

**Note:** if the what-was-next TUI is already running when Claude changes your
data, the TUI won't update live — it re-reads on the next navigation or
keypress.

To run the server directly (Claude Code does this for you): `what-was-next mcp`.
