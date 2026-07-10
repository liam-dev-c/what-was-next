# MCP Server for what-was-next — Design

**Date:** 2026-07-10
**Status:** Approved, ready for implementation plan

## Goal

Let Claude manage what-was-next data — create/rename/delete projects, and
create/update/complete/reorder/delete tasks — by exposing the existing `store`
operations through a Model Context Protocol (MCP) server. Add a setup command
that registers the server with Claude Code automatically.

## Scope

In scope:

- A stdio MCP server exposing 10 tools over the existing `store` package.
- Subcommand dispatch in the binary (`mcp`, `mcp install`), with the bare
  command still launching the TUI.
- A setup command that runs `claude mcp add` to register the server.
- WAL mode on the SQLite database so the TUI and MCP server can share it.

Out of scope (explicit YAGNI for now, easy to add later):

- Time tracking (start/stop timer) tools.
- Summary / history read tools.
- Live auto-refresh of a running TUI when the DB changes externally.

## Architecture

### Command dispatch

`main.go` currently launches the TUI unconditionally. Add lightweight
subcommand dispatch — a plain `switch` on `os.Args[1]`, no CLI framework, to
keep dependencies minimal:

- `what-was-next` → TUI (unchanged default).
- `what-was-next mcp` → run the stdio MCP server.
- `what-was-next mcp install` → register the server with Claude Code.

`dbPath()` moves out of `main.go` into a shared location (e.g. a small exported
helper) so all entry points open the same database at
`~/.config/what-was-next/what-was-next.db` (honoring `XDG_CONFIG_HOME`).

### New package: `internal/mcpserver`

Holds the server construction and tool definitions. It depends **only** on
`*store.Store` — the same clean boundary the TUI uses. No `database/sql` and no
SQL leaks into this package.

- Uses the official Go MCP SDK: `github.com/modelcontextprotocol/go-sdk/mcp`.
- Exposes a constructor that takes a `*store.Store` and returns a configured
  server ready to serve over stdio.
- Each tool handler validates/parses typed args, calls exactly one `store`
  method, and returns a text or JSON result.

### Data flow

```
Claude Code ──stdio(MCP)──▶ what-was-next mcp ──▶ internal/mcpserver ──▶ store ──▶ SQLite
```

The TUI is a separate process reading/writing the same SQLite file.

## Tools

Each tool is a thin wrapper over an existing `store` method. IDs are `int64`,
matching the store. Read tools return enough for Claude to act on results
(IDs plus names/titles and done state).

| Tool | Args | Store call | Notes |
|------|------|-----------|-------|
| `list_projects` | – | `ListProjects` | Returns id, name, created_at. |
| `create_project` | `name` | `CreateProject` | Returns the created project. |
| `rename_project` | `id`, `name` | `RenameProject` | |
| `delete_project` | `id` | `DeleteProject` | Cascades to tasks (existing FK `ON DELETE CASCADE`). |
| `list_tasks` | `project_id` | `ListTasks` | Returns id, title, notes, done, sort_order. |
| `create_task` | `project_id`, `title` | `CreateTask` | Returns the created task. |
| `update_task` | `id`, `title`, `notes` | `UpdateTask` | Updates title and notes together, matching the store signature. |
| `set_task_done` | `id`, `done` (bool) | `SetTaskDone` | Complete or reopen a task. |
| `move_task` | `id`, `direction` (`up`\|`down`) | `MoveTask` | Maps `up`→-1, `down`→+1. No-op at ends. |
| `delete_task` | `id` | `DeleteTask` | |

No new `store` methods are required — every operation already exists.

### Error handling

- Invalid/malformed args → MCP tool error with a clear message; do not call the
  store.
- `move_task` `direction` other than `up`/`down` → tool error.
- Store errors propagate as MCP tool errors with the store's wrapped message.
- Referencing a nonexistent id: the store's `UPDATE`/`DELETE` calls are no-ops
  (affect zero rows). Where it adds value, handlers may check `RowsAffected` and
  return a "not found" error so Claude gets clear feedback; otherwise the
  no-op/empty-result behavior is acceptable. This is decided per tool during
  implementation.

## Concurrency

The TUI and MCP server are separate processes on the same SQLite file.

- **Enable WAL mode** in `store.Open` via `PRAGMA journal_mode=WAL` so a write
  from the MCP server does not block or collide with the running TUI.
- **A running TUI does not auto-refresh** when Claude changes the DB. The TUI
  re-queries on navigation/keypress, so external changes appear on the next
  refresh, not live. This is documented behavior, not a bug; a file-watcher is
  out of scope.

## Setup command

`what-was-next mcp install` registers the server with Claude Code:

```
claude mcp add what-was-next -- <abs-path-to-binary> mcp
```

- The absolute binary path is resolved with `os.Executable()`.
- A `--scope` flag (`user` | `project` | `local`) passes through to
  `claude mcp add`, defaulting to `user`.
- On success, print a confirmation and a hint to restart/reconnect Claude Code.
- If the `claude` CLI is not found on `PATH`, fail with a clear message and
  print the equivalent command for manual registration.

## Testing

- `internal/mcpserver`: table-driven tests per tool against an in-memory store
  (`store.Open(":memory:")`), asserting the store side effect (e.g. after
  `create_task`, `ListTasks` shows it) and argument-validation errors
  (bad `direction`, missing required args).
- `store`: add a test that WAL mode is active after `Open` (query
  `PRAGMA journal_mode`).
- Dispatch: a small test that unknown/`mcp` args route correctly (extract the
  dispatch decision into a testable function rather than testing `main`).

## Documentation

Update `README.md` with a short "Claude / MCP" section: what the server does,
`what-was-next mcp install` to set it up, the list of tools, and the note that a
running TUI won't live-refresh.
