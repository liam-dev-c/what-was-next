# MCP Server Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Expose what-was-next's project/task operations to Claude via a stdio MCP server, plus a `what-was-next mcp install` command that registers it with Claude Code.

**Architecture:** A new `internal/mcpserver` package wraps `*store.Store` and registers 10 MCP tools using the official Go SDK. `main.go` gains subcommand dispatch: bare command runs the TUI (unchanged), `mcp` serves over stdio, `mcp install` shells out to `claude mcp add`. SQLite gets WAL mode so the TUI and server can share the DB file.

**Tech Stack:** Go 1.26, `github.com/modelcontextprotocol/go-sdk@v1.6.1`, existing `internal/store` (modernc.org/sqlite), bubbletea/v2 TUI.

## Global Constraints

- Go version floor: 1.26 (already in `go.mod`).
- Add exactly one new dependency: `github.com/modelcontextprotocol/go-sdk@v1.6.1`. No CLI framework — dispatch is a plain `switch`.
- `internal/mcpserver` depends only on `*store.Store` and the MCP SDK. No `database/sql`, no SQL.
- IDs are `int64`, matching `store`.
- Code must pass `gofmt -l` (no output) and `go vet ./...`. Match existing import grouping: stdlib, then third-party, then `github.com/liam-dev-c/what-was-next/...`.
- **Commit messages must NOT include a `Co-Authored-By` trailer.**
- Run `go test ./...` green before every commit.

## File Structure

- Create `internal/mcpserver/server.go` — `New`, `Serve`, result helpers, `noArgs`.
- Create `internal/mcpserver/projects.go` — the 4 project tools.
- Create `internal/mcpserver/tasks.go` — the 6 task tools + `parseDirection`.
- Create `internal/mcpserver/install.go` — `Install`, `installArgs`.
- Create `internal/mcpserver/server_test.go` — in-memory client/server integration tests.
- Create `internal/mcpserver/install_test.go` — `installArgs` / scope-validation tests.
- Create `internal/store/wal_test.go` — asserts WAL mode is active.
- Modify `internal/store/store.go` — enable WAL in `Open`.
- Modify `main.go` — subcommand dispatch.
- Create `main_test.go` — `command` / `scopeFlag` routing tests.
- Modify `README.md` — "Claude / MCP" section.

---

### Task 1: Enable WAL mode on the database

**Files:**
- Modify: `internal/store/store.go` (inside `Open`, after the `foreign_keys` pragma)
- Test: `internal/store/wal_test.go`

**Interfaces:**
- Consumes: `store.Open(path string) (*Store, error)` (existing).
- Produces: no API change; `Open` now sets `journal_mode=WAL` for file-backed DBs.

- [ ] **Step 1: Write the failing test**

Create `internal/store/wal_test.go`:

```go
package store

import (
	"path/filepath"
	"testing"
)

func TestOpenEnablesWAL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wal.db")
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	var mode string
	if err := s.db.QueryRow(`PRAGMA journal_mode`).Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want \"wal\"", mode)
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/store/ -run TestOpenEnablesWAL -v`
Expected: FAIL — `journal_mode = "delete", want "wal"`.

- [ ] **Step 3: Add the WAL pragma**

In `internal/store/store.go`, immediately after the existing foreign-keys block:

```go
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec("PRAGMA journal_mode = WAL;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable WAL: %w", err)
	}
```

Note: for `:memory:` DBs this pragma is a harmless no-op (returns `"memory"`); existing in-memory tests are unaffected.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/store/ -run TestOpenEnablesWAL -v`
Expected: PASS.

- [ ] **Step 5: Run the full store suite and commit**

Run: `go test ./internal/store/`
Expected: PASS (existing in-memory tests still pass).

```bash
gofmt -w internal/store/store.go internal/store/wal_test.go
git add internal/store/store.go internal/store/wal_test.go
git commit -m "Enable WAL mode so TUI and MCP server can share the DB"
```

---

### Task 2: mcpserver scaffold + project tools

**Files:**
- Create: `internal/mcpserver/server.go`
- Create: `internal/mcpserver/projects.go`
- Test: `internal/mcpserver/server_test.go` (project tests; task tests added in Task 3)
- Modify: `go.mod`, `go.sum` (via `go get`)

**Interfaces:**
- Consumes: `*store.Store` and its methods `ListProjects`, `CreateProject`, `RenameProject`, `DeleteProject`.
- Produces:
  - `mcpserver.New(s *store.Store) *mcp.Server`
  - `mcpserver.Serve(ctx context.Context, s *store.Store) error`
  - Unexported helpers `jsonResult`, `textResult`, type `noArgs`, and `addProjectTools(srv *mcp.Server, s *store.Store)`.
  - Registered tools: `list_projects`, `create_project`, `rename_project`, `delete_project`.

- [ ] **Step 1: Add the SDK dependency**

Run:
```bash
go get github.com/modelcontextprotocol/go-sdk@v1.6.1
```
Expected: `go.mod` now requires `github.com/modelcontextprotocol/go-sdk v1.6.1`.

- [ ] **Step 2: Write the server scaffold**

Create `internal/mcpserver/server.go`:

```go
// Package mcpserver exposes what-was-next's store operations as MCP tools so
// agents such as Claude can manage projects and tasks. It depends only on
// *store.Store; no SQL lives here.
package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

const (
	serverName    = "what-was-next"
	serverVersion = "0.1.0"
)

// noArgs is the input type for tools that take no arguments.
type noArgs struct{}

// New builds an MCP server exposing project and task tools backed by s.
func New(s *store.Store) *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{
		Name:    serverName,
		Version: serverVersion,
	}, nil)
	addProjectTools(srv, s)
	addTaskTools(srv, s)
	return srv
}

// Serve runs the MCP server over stdio until ctx is cancelled or stdin closes.
func Serve(ctx context.Context, s *store.Store) error {
	return New(s).Run(ctx, &mcp.StdioTransport{})
}

// jsonResult marshals v to a JSON text tool result.
func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, nil, fmt.Errorf("marshal result: %w", err)
	}
	return textResult(string(b))
}

// textResult returns a plain-text tool result.
func textResult(msg string) (*mcp.CallToolResult, any, error) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: msg}},
	}, nil, nil
}
```

Note: `New` references `addTaskTools`, which is created in Task 3. The package will not compile until Task 3's `tasks.go` exists. Both files are written before the first `go test` of this package (Step 5 stubs `addTaskTools` so Task 2 can be tested in isolation).

- [ ] **Step 3: Write the project tools**

Create `internal/mcpserver/projects.go`:

```go
package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

func addProjectTools(srv *mcp.Server, s *store.Store) {
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_projects",
		Description: "List all projects with their ids, names, and creation times.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, _ noArgs) (*mcp.CallToolResult, any, error) {
		projects, err := s.ListProjects()
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(projects)
	})

	type createArgs struct {
		Name string `json:"name" jsonschema:"name of the new project"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_project",
		Description: "Create a new project and return it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createArgs) (*mcp.CallToolResult, any, error) {
		if args.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		p, err := s.CreateProject(args.Name)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(p)
	})

	type renameArgs struct {
		ID   int64  `json:"id" jsonschema:"id of the project to rename"`
		Name string `json:"name" jsonschema:"new project name"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "rename_project",
		Description: "Rename an existing project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args renameArgs) (*mcp.CallToolResult, any, error) {
		if args.Name == "" {
			return nil, nil, fmt.Errorf("name is required")
		}
		if err := s.RenameProject(args.ID, args.Name); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("renamed project %d to %q", args.ID, args.Name))
	})

	type deleteArgs struct {
		ID int64 `json:"id" jsonschema:"id of the project to delete; its tasks are deleted too"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_project",
		Description: "Delete a project and all of its tasks.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args deleteArgs) (*mcp.CallToolResult, any, error) {
		if err := s.DeleteProject(args.ID); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("deleted project %d", args.ID))
	})
}
```

- [ ] **Step 4: Write the integration-test harness and project tests**

Create `internal/mcpserver/server_test.go`:

```go
package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

// newSession wires a client to a server backed by an in-memory store.
func newSession(t *testing.T, s *store.Store) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	serverT, clientT := mcp.NewInMemoryTransports()
	if _, err := New(s).Connect(ctx, serverT, nil); err != nil {
		t.Fatalf("server connect: %v", err)
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "0"}, nil)
	sess, err := client.Connect(ctx, clientT, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = sess.Close() })
	return sess
}

func newStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// call invokes a tool and fails the test if the tool reports an error.
func call(t *testing.T, sess *mcp.ClientSession, name string, args map[string]any) string {
	t.Helper()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("call %s: %v", name, err)
	}
	if res.IsError {
		t.Fatalf("tool %s reported error: %s", name, resultText(res))
	}
	return resultText(res)
}

// callErr invokes a tool expecting it to report a tool error.
func callErr(t *testing.T, sess *mcp.ClientSession, name string, args map[string]any) {
	t.Helper()
	res, err := sess.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return // transport/protocol error also counts as failure to execute
	}
	if !res.IsError {
		t.Fatalf("tool %s: expected error, got %s", name, resultText(res))
	}
}

func resultText(res *mcp.CallToolResult) string {
	if len(res.Content) == 0 {
		return ""
	}
	if tc, ok := res.Content[0].(*mcp.TextContent); ok {
		return tc.Text
	}
	return ""
}

func TestCreateAndListProjects(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_project", map[string]any{"name": "Work"})
	out := call(t, sess, "list_projects", map[string]any{})
	if !strings.Contains(out, "Work") {
		t.Fatalf("list_projects missing Work: %s", out)
	}
}

func TestRenameProject(t *testing.T) {
	s := newStore(t)
	sess := newSession(t, s)
	call(t, sess, "rename_project", map[string]any{"id": 1, "name": "Renamed"})
	out := call(t, sess, "list_projects", map[string]any{})
	if !strings.Contains(out, "Renamed") {
		t.Fatalf("rename not reflected: %s", out)
	}
}

func TestDeleteProjectCascades(t *testing.T) {
	s := newStore(t)
	sess := newSession(t, s)
	call(t, sess, "delete_project", map[string]any{"id": 1})
	out := call(t, sess, "list_projects", map[string]any{})
	if strings.Contains(out, "Inbox") {
		t.Fatalf("project 1 not deleted: %s", out)
	}
}

func TestCreateProjectRequiresName(t *testing.T) {
	sess := newSession(t, newStore(t))
	callErr(t, sess, "create_project", map[string]any{"name": ""})
}
```

Note: `store.Open(":memory:")` seeds a default "Inbox" project with id 1 — the rename/delete tests rely on that id.

- [ ] **Step 5: Add a temporary stub so the package compiles, then run tests**

Task 3 provides the real `addTaskTools`. To test Task 2 in isolation, temporarily add this stub at the bottom of `internal/mcpserver/projects.go`:

```go
// TEMP stub — replaced by tasks.go in Task 3.
func addTaskTools(srv *mcp.Server, s *store.Store) {}
```

Run: `go test ./internal/mcpserver/ -v`
Expected: PASS (4 project tests).

- [ ] **Step 6: Tidy and commit**

```bash
go mod tidy
gofmt -w internal/mcpserver/
go vet ./internal/mcpserver/
git add go.mod go.sum internal/mcpserver/server.go internal/mcpserver/projects.go internal/mcpserver/server_test.go
git commit -m "Add MCP server scaffold and project tools"
```

---

### Task 3: Task tools

**Files:**
- Create: `internal/mcpserver/tasks.go`
- Modify: `internal/mcpserver/projects.go` (remove the TEMP `addTaskTools` stub)
- Test: `internal/mcpserver/server_test.go` (append task tests)

**Interfaces:**
- Consumes: `*store.Store` methods `ListTasks`, `CreateTask`, `UpdateTask`, `SetTaskDone`, `MoveTask`, `DeleteTask`; the `noArgs`/`jsonResult`/`textResult` helpers from Task 2.
- Produces: `addTaskTools(srv *mcp.Server, s *store.Store)`, `parseDirection(string) (int, error)`, and tools `list_tasks`, `create_task`, `update_task`, `set_task_done`, `move_task`, `delete_task`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/mcpserver/server_test.go`:

```go
func TestParseDirection(t *testing.T) {
	cases := []struct {
		in      string
		want    int
		wantErr bool
	}{
		{"up", -1, false},
		{"down", 1, false},
		{"sideways", 0, true},
		{"", 0, true},
	}
	for _, c := range cases {
		got, err := parseDirection(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseDirection(%q): expected error", c.in)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("parseDirection(%q) = %d, %v; want %d, nil", c.in, got, err, c.want)
		}
	}
}

func TestCreateAndListTasks(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "Write docs"})
	out := call(t, sess, "list_tasks", map[string]any{"project_id": 1})
	if !strings.Contains(out, "Write docs") {
		t.Fatalf("list_tasks missing task: %s", out)
	}
}

func TestUpdateTask(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "Old"})
	call(t, sess, "update_task", map[string]any{"id": 1, "title": "New", "notes": "hi"})
	out := call(t, sess, "list_tasks", map[string]any{"project_id": 1})
	if !strings.Contains(out, "New") || !strings.Contains(out, "hi") {
		t.Fatalf("update not reflected: %s", out)
	}
}

func TestSetTaskDone(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "T"})
	call(t, sess, "set_task_done", map[string]any{"id": 1, "done": true})
	out := call(t, sess, "list_tasks", map[string]any{"project_id": 1})
	if !strings.Contains(out, `"Done":true`) {
		t.Fatalf("task not marked done: %s", out)
	}
}

func TestMoveTask(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "First"})
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "Second"})
	// Move the second task up; it should now sort before the first.
	call(t, sess, "move_task", map[string]any{"id": 2, "direction": "up"})
	out := call(t, sess, "list_tasks", map[string]any{"project_id": 1})
	if strings.Index(out, "Second") > strings.Index(out, "First") {
		t.Fatalf("move up did not reorder: %s", out)
	}
}

func TestMoveTaskBadDirection(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "T"})
	callErr(t, sess, "move_task", map[string]any{"id": 1, "direction": "sideways"})
}

func TestDeleteTask(t *testing.T) {
	sess := newSession(t, newStore(t))
	call(t, sess, "create_task", map[string]any{"project_id": 1, "title": "Doomed"})
	call(t, sess, "delete_task", map[string]any{"id": 1})
	out := call(t, sess, "list_tasks", map[string]any{"project_id": 1})
	if strings.Contains(out, "Doomed") {
		t.Fatalf("task not deleted: %s", out)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run 'Task|ParseDirection' -v`
Expected: compile error — `parseDirection` undefined (and the task tools are not registered yet).

- [ ] **Step 3: Remove the stub and write the task tools**

Delete the TEMP `addTaskTools` stub from `internal/mcpserver/projects.go`.

Create `internal/mcpserver/tasks.go`:

```go
package mcpserver

import (
	"context"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/liam-dev-c/what-was-next/internal/store"
)

// parseDirection maps a move direction to the store's delta (-1 up, +1 down).
func parseDirection(dir string) (int, error) {
	switch dir {
	case "up":
		return -1, nil
	case "down":
		return 1, nil
	default:
		return 0, fmt.Errorf("direction must be \"up\" or \"down\", got %q", dir)
	}
}

func addTaskTools(srv *mcp.Server, s *store.Store) {
	type listArgs struct {
		ProjectID int64 `json:"project_id" jsonschema:"id of the project whose tasks to list"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List the tasks in a project, in sort order.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args listArgs) (*mcp.CallToolResult, any, error) {
		tasks, err := s.ListTasks(args.ProjectID)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(tasks)
	})

	type createArgs struct {
		ProjectID int64  `json:"project_id" jsonschema:"id of the project to add the task to"`
		Title     string `json:"title" jsonschema:"task title"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "create_task",
		Description: "Create a task in a project and return it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args createArgs) (*mcp.CallToolResult, any, error) {
		if args.Title == "" {
			return nil, nil, fmt.Errorf("title is required")
		}
		task, err := s.CreateTask(args.ProjectID, args.Title)
		if err != nil {
			return nil, nil, err
		}
		return jsonResult(task)
	})

	type updateArgs struct {
		ID    int64  `json:"id" jsonschema:"id of the task to update"`
		Title string `json:"title" jsonschema:"new title"`
		Notes string `json:"notes" jsonschema:"new notes; pass the full notes text, may be empty"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "update_task",
		Description: "Replace a task's title and notes. Both fields are overwritten.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args updateArgs) (*mcp.CallToolResult, any, error) {
		if args.Title == "" {
			return nil, nil, fmt.Errorf("title is required")
		}
		if err := s.UpdateTask(args.ID, args.Title, args.Notes); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("updated task %d", args.ID))
	})

	type doneArgs struct {
		ID   int64 `json:"id" jsonschema:"id of the task"`
		Done bool  `json:"done" jsonschema:"true to complete the task, false to reopen it"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "set_task_done",
		Description: "Mark a task done or reopen it.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args doneArgs) (*mcp.CallToolResult, any, error) {
		if err := s.SetTaskDone(args.ID, args.Done); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("set task %d done=%v", args.ID, args.Done))
	})

	type moveArgs struct {
		ID        int64  `json:"id" jsonschema:"id of the task to move"`
		Direction string `json:"direction" jsonschema:"\"up\" or \"down\""`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "move_task",
		Description: "Move a task up or down one position within its project.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args moveArgs) (*mcp.CallToolResult, any, error) {
		delta, err := parseDirection(args.Direction)
		if err != nil {
			return nil, nil, err
		}
		if err := s.MoveTask(args.ID, delta); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("moved task %d %s", args.ID, args.Direction))
	})

	type deleteArgs struct {
		ID int64 `json:"id" jsonschema:"id of the task to delete"`
	}
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "delete_task",
		Description: "Delete a task.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, args deleteArgs) (*mcp.CallToolResult, any, error) {
		if err := s.DeleteTask(args.ID); err != nil {
			return nil, nil, err
		}
		return textResult(fmt.Sprintf("deleted task %d", args.ID))
	})
}
```

- [ ] **Step 4: Run the full package suite to verify it passes**

Run: `go test ./internal/mcpserver/ -v`
Expected: PASS (all project + task tests). If `TestSetTaskDone` fails on the `"Done":true` substring, print the raw output and confirm the JSON field name matches `store.Task` (`Done`); adjust the assertion to the actual marshaled key if the struct uses a `json:` tag.

- [ ] **Step 5: Tidy and commit**

```bash
gofmt -w internal/mcpserver/
go vet ./internal/mcpserver/
git add internal/mcpserver/tasks.go internal/mcpserver/projects.go internal/mcpserver/server_test.go
git commit -m "Add MCP task tools"
```

---

### Task 4: Install command

**Files:**
- Create: `internal/mcpserver/install.go`
- Test: `internal/mcpserver/install_test.go`

**Interfaces:**
- Consumes: nothing from earlier tasks (standalone).
- Produces:
  - `mcpserver.Install(scope string) error` — validates scope, resolves the binary path, runs `claude mcp add`.
  - `installArgs(binPath, scope string) []string` — the argument vector passed to `claude`.

- [ ] **Step 1: Write the failing tests**

Create `internal/mcpserver/install_test.go`:

```go
package mcpserver

import (
	"reflect"
	"testing"
)

func TestInstallArgs(t *testing.T) {
	got := installArgs("/usr/local/bin/what-was-next", "user")
	want := []string{
		"mcp", "add", "--scope", "user",
		"what-was-next", "--", "/usr/local/bin/what-was-next", "mcp",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installArgs = %v, want %v", got, want)
	}
}

func TestInstallRejectsBadScope(t *testing.T) {
	if err := Install("bogus"); err == nil {
		t.Fatal("Install(\"bogus\"): expected error, got nil")
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/mcpserver/ -run Install -v`
Expected: compile error — `installArgs` / `Install` undefined.

- [ ] **Step 3: Write the install command**

Create `internal/mcpserver/install.go`:

```go
package mcpserver

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var validScopes = map[string]bool{"user": true, "project": true, "local": true}

// Install registers this binary as an MCP server named "what-was-next" with the
// Claude Code CLI at the given scope (user|project|local).
func Install(scope string) error {
	if !validScopes[scope] {
		return fmt.Errorf("invalid scope %q (want user, project, or local)", scope)
	}
	bin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	args := installArgs(bin, scope)

	if _, err := exec.LookPath("claude"); err != nil {
		fmt.Println("The `claude` CLI was not found on your PATH.")
		fmt.Println("Install Claude Code, then register the server manually with:")
		fmt.Printf("  claude %s\n", strings.Join(args, " "))
		return fmt.Errorf("claude CLI not found on PATH")
	}

	cmd := exec.Command("claude", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("register MCP server: %w", err)
	}
	fmt.Println("Registered what-was-next as an MCP server. Restart Claude Code to use it.")
	return nil
}

// installArgs builds the `claude` argument vector that registers this binary.
func installArgs(binPath, scope string) []string {
	return []string{
		"mcp", "add", "--scope", scope,
		"what-was-next", "--", binPath, "mcp",
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/mcpserver/ -run Install -v`
Expected: PASS (both tests).

- [ ] **Step 5: Tidy and commit**

```bash
gofmt -w internal/mcpserver/install.go internal/mcpserver/install_test.go
go vet ./internal/mcpserver/
git add internal/mcpserver/install.go internal/mcpserver/install_test.go
git commit -m "Add mcp install command that registers with Claude Code"
```

---

### Task 5: Wire subcommand dispatch into main

**Files:**
- Modify: `main.go`
- Test: `main_test.go`

**Interfaces:**
- Consumes: `mcpserver.Serve`, `mcpserver.Install`, and the existing `store.Open`, `tui.New`, `dbPath`.
- Produces:
  - `command(args []string) cmd` — classifies args into `cmdTUI` / `cmdMCPServe` / `cmdMCPInstall`.
  - `scopeFlag(args []string) string` — reads `--scope <value>` from args, default `"user"`.
  - `run(args []string) error` dispatches; `runTUI()` and `runMCPServe()` do the work.

- [ ] **Step 1: Write the failing tests**

Create `main_test.go`:

```go
package main

import "testing"

func TestCommand(t *testing.T) {
	cases := []struct {
		args []string
		want cmd
	}{
		{nil, cmdTUI},
		{[]string{}, cmdTUI},
		{[]string{"mcp"}, cmdMCPServe},
		{[]string{"mcp", "install"}, cmdMCPInstall},
		{[]string{"mcp", "install", "--scope", "project"}, cmdMCPInstall},
		{[]string{"nonsense"}, cmdTUI},
	}
	for _, c := range cases {
		if got := command(c.args); got != c.want {
			t.Errorf("command(%v) = %d, want %d", c.args, got, c.want)
		}
	}
}

func TestScopeFlag(t *testing.T) {
	if got := scopeFlag([]string{"mcp", "install"}); got != "user" {
		t.Errorf("default scope = %q, want \"user\"", got)
	}
	if got := scopeFlag([]string{"mcp", "install", "--scope", "project"}); got != "project" {
		t.Errorf("scope = %q, want \"project\"", got)
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test . -run 'TestCommand|TestScopeFlag' -v`
Expected: compile error — `command` / `scopeFlag` / `cmd` undefined.

- [ ] **Step 3: Rewrite main.go with dispatch**

Replace the contents of `main.go` with:

```go
// Command what-was-next is a terminal task manager and time tracker.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	tea "charm.land/bubbletea/v2"

	"github.com/liam-dev-c/what-was-next/internal/mcpserver"
	"github.com/liam-dev-c/what-was-next/internal/store"
	"github.com/liam-dev-c/what-was-next/internal/tui"
)

type cmd int

const (
	cmdTUI cmd = iota
	cmdMCPServe
	cmdMCPInstall
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "what-was-next:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	switch command(args) {
	case cmdMCPInstall:
		return mcpserver.Install(scopeFlag(args))
	case cmdMCPServe:
		return runMCPServe()
	default:
		return runTUI()
	}
}

// command classifies CLI args into a subcommand.
func command(args []string) cmd {
	if len(args) >= 1 && args[0] == "mcp" {
		if len(args) >= 2 && args[1] == "install" {
			return cmdMCPInstall
		}
		return cmdMCPServe
	}
	return cmdTUI
}

// scopeFlag reads "--scope <value>" from args, defaulting to "user".
func scopeFlag(args []string) string {
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--scope" {
			return args[i+1]
		}
	}
	return "user"
}

func runTUI() error {
	path, err := dbPath()
	if err != nil {
		return err
	}
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()

	model, err := tui.New(s)
	if err != nil {
		return err
	}
	p := tea.NewProgram(model)
	_, err = p.Run()
	return err
}

func runMCPServe() error {
	path, err := dbPath()
	if err != nil {
		return err
	}
	s, err := store.Open(path)
	if err != nil {
		return err
	}
	defer s.Close()
	return mcpserver.Serve(context.Background(), s)
}

// dbPath resolves ~/.config/what-was-next/what-was-next.db, honoring
// XDG_CONFIG_HOME, and ensures the directory exists.
func dbPath() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home dir: %w", err)
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "what-was-next")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create config dir: %w", err)
	}
	return filepath.Join(dir, "what-was-next.db"), nil
}
```

- [ ] **Step 4: Run the tests and the full build to verify**

Run: `go test . -run 'TestCommand|TestScopeFlag' -v`
Expected: PASS.

Run: `go build ./... && go test ./...`
Expected: build succeeds; all packages PASS.

- [ ] **Step 5: Smoke-test the server manually**

Run:
```bash
go build -o /tmp/wwn . && printf '%s\n' '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}' '{"jsonrpc":"2.0","method":"notifications/initialized"}' '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | /tmp/wwn mcp
```
Expected: JSON responses; the `tools/list` result lists all 10 tool names (`list_projects`, `create_project`, `rename_project`, `delete_project`, `list_tasks`, `create_task`, `update_task`, `set_task_done`, `move_task`, `delete_task`). The process then exits when stdin closes.

- [ ] **Step 6: Commit**

```bash
gofmt -w main.go main_test.go
git add main.go main_test.go
git commit -m "Add mcp/mcp-install subcommand dispatch"
```

---

### Task 6: Document the MCP server in the README

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes: nothing (documentation only).
- Produces: a "Claude / MCP" section.

- [ ] **Step 1: Add the section**

Append to `README.md`, after the "Usage" section:

```markdown
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
- Tasks: `list_tasks`, `create_task`, `update_task`, `set_task_done`,
  `move_task`, `delete_task`

`delete_project` also deletes that project's tasks.

**Note:** if the what-was-next TUI is already running when Claude changes your
data, the TUI won't update live — it re-reads on the next navigation or
keypress.

To run the server directly (Claude Code does this for you): `what-was-next mcp`.
```

- [ ] **Step 2: Verify and commit**

Run: `go build ./... && go test ./...`
Expected: unchanged — still green (docs-only change).

```bash
git add README.md
git commit -m "Document the MCP server in the README"
```

---

## Self-Review Notes

- **Spec coverage:** Command dispatch (Task 5), `internal/mcpserver` boundary (Tasks 2–3), all 10 tools (Tasks 2–3), WAL mode (Task 1), setup command via `claude mcp add` with `--scope` and missing-CLI fallback (Task 4), testing at store/mcpserver/dispatch levels (Tasks 1–5), README docs incl. no-live-refresh note (Task 6). All spec sections map to a task.
- **Type consistency:** `mcpserver.New`/`Serve`/`Install` and helpers `installArgs`/`parseDirection` are used with identical signatures across tasks. `command`/`scopeFlag`/`cmd` defined and consumed in Task 5 only.
- **Known risk:** `CallToolResult.IsError` and the `store.Task` JSON key `Done` are assumed by tests; Task 3 Step 4 calls out how to adjust the `set_task_done` assertion if the marshaled key differs. If `IsError` is absent in v1.6.1, switch the error-case helpers to assert on the returned Go error instead.
