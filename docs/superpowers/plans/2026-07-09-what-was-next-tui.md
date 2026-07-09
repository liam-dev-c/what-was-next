# what-was-next TUI Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a terminal task manager + time tracker (a simple Super Productivity) with task CRUD, per-task time tracking, projects, and a daily summary.

**Architecture:** A `store` package owns all SQLite persistence and domain types and is unit-tested directly against a temp-file DB with an injectable clock. A `tui` package (Bubble Tea) holds view state only, calling the store for every mutation and reloading after. `main.go` wires them together.

**Tech Stack:** Go 1.26, `charmbracelet/bubbletea`, `charmbracelet/bubbles`, `charmbracelet/lipgloss`, `modernc.org/sqlite` (pure-Go, no cgo).

## Global Constraints

- Module path: `github.com/liam-dev-c/what-was-next`.
- Go version floor: 1.26.
- SQLite driver: `modernc.org/sqlite` only — registers driver name `"sqlite"`. No cgo.
- All timestamps stored and compared in **UTC**.
- The `store` package MUST NOT import the `tui` package (one-way dependency).
- Store methods own all SQL; the TUI never touches `database/sql`.
- Commit after every task (user preference: commit as you go). Use `git commit -m` with subject only — **never add a Co-Authored-By trailer**.

---

### Task 1: Module + store foundation (Open, schema, clock)

**Files:**
- Create: `go.mod` (via `go mod init`)
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Produces:
  - `type Store struct { db *sql.DB; now func() time.Time }`
  - `func Open(path string) (*Store, error)` — opens/creates the DB at `path`, runs migrations, seeds a default project named `"Inbox"` if no projects exist. `path` may be `":memory:"` for tests.
  - `func (s *Store) Close() error`
  - Unexported test helper `newTestStore(t *testing.T) *Store` lives in the test file.
  - Domain types (defined here so later tasks share them):
    ```go
    type Project struct {
        ID        int64
        Name      string
        CreatedAt time.Time
    }
    type Task struct {
        ID        int64
        ProjectID int64
        Title     string
        Notes     string
        Done      bool
        SortOrder int64
        CreatedAt time.Time
        DoneAt    *time.Time
    }
    type TimeEntry struct {
        ID        int64
        TaskID    int64
        StartedAt time.Time
        EndedAt   *time.Time
    }
    ```

- [ ] **Step 1: Init module and add deps**

Run:
```bash
cd /Users/liam.clarke/Documents/Github/what-was-next
go mod init github.com/liam-dev-c/what-was-next
go get modernc.org/sqlite@v1.53.0
go get github.com/charmbracelet/bubbletea@v1.3.10
go get github.com/charmbracelet/bubbles@v1.0.0
go get github.com/charmbracelet/lipgloss@v1.1.0
```
Expected: `go.mod` and `go.sum` created listing these modules.

- [ ] **Step 2: Write the failing test**

Create `internal/store/store_test.go`:
```go
package store

import (
	"testing"
	"time"
)

// fixedClock returns a deterministic UTC time for tests.
func fixedClock() time.Time {
	return time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
}

func newTestStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	s.now = fixedClock
	t.Cleanup(func() { s.Close() })
	return s
}

func TestOpenSeedsDefaultProject(t *testing.T) {
	s := newTestStore(t)
	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("want 1 seeded project, got %d", len(projects))
	}
	if projects[0].Name != "Inbox" {
		t.Errorf("want default project 'Inbox', got %q", projects[0].Name)
	}
}
```
Note: this test references `ListProjects`, implemented in Task 2. Write it now; it will not compile until Task 2. To keep Task 1 independently runnable, temporarily assert only on `Open`/`Close` — replace the body below and restore the fuller test in Task 2:
```go
func TestOpenAndClose(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}
```
Use `TestOpenAndClose` for Task 1; add `TestOpenSeedsDefaultProject` in Task 2.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestOpenAndClose -v`
Expected: FAIL — `undefined: Open` / `undefined: Store`.

- [ ] **Step 4: Write minimal implementation**

Create `internal/store/store.go`:
```go
// Package store is the SQLite persistence layer and domain model for
// what-was-next. It owns all SQL; nothing above it touches database/sql.
package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Project struct {
	ID        int64
	Name      string
	CreatedAt time.Time
}

type Task struct {
	ID        int64
	ProjectID int64
	Title     string
	Notes     string
	Done      bool
	SortOrder int64
	CreatedAt time.Time
	DoneAt    *time.Time
}

type TimeEntry struct {
	ID        int64
	TaskID    int64
	StartedAt time.Time
	EndedAt   *time.Time
}

// Store wraps the database. now is injectable so tests are deterministic.
type Store struct {
	db  *sql.DB
	now func() time.Time
}

const schema = `
CREATE TABLE IF NOT EXISTS projects (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	name       TEXT NOT NULL,
	created_at TIMESTAMP NOT NULL
);
CREATE TABLE IF NOT EXISTS tasks (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	project_id INTEGER NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
	title      TEXT NOT NULL,
	notes      TEXT NOT NULL DEFAULT '',
	done       INTEGER NOT NULL DEFAULT 0,
	sort_order INTEGER NOT NULL DEFAULT 0,
	created_at TIMESTAMP NOT NULL,
	done_at    TIMESTAMP
);
CREATE TABLE IF NOT EXISTS time_entries (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	task_id    INTEGER NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
	started_at TIMESTAMP NOT NULL,
	ended_at   TIMESTAMP
);
`

// Open opens (creating if needed) the SQLite DB at path, runs migrations,
// and seeds a default "Inbox" project when none exist. path may be ":memory:".
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	s := &Store{db: db, now: func() time.Time { return time.Now().UTC() }}
	if err := s.seedDefaultProject(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) seedDefaultProject() error {
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM projects`).Scan(&n); err != nil {
		return fmt.Errorf("count projects: %w", err)
	}
	if n > 0 {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO projects (name, created_at) VALUES (?, ?)`,
		"Inbox", s.now(),
	)
	if err != nil {
		return fmt.Errorf("seed default project: %w", err)
	}
	return nil
}

func (s *Store) Close() error { return s.db.Close() }
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/store/ -run TestOpenAndClose -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/store/store.go internal/store/store_test.go
git commit -m "Add store foundation: schema, Open, injectable clock"
```

---

### Task 2: Project CRUD

**Files:**
- Create: `internal/store/project.go`
- Test: `internal/store/project_test.go`

**Interfaces:**
- Consumes: `Store`, `Project` (Task 1).
- Produces:
  - `func (s *Store) CreateProject(name string) (Project, error)`
  - `func (s *Store) ListProjects() ([]Project, error)` — ordered by `id`.
  - `func (s *Store) RenameProject(id int64, name string) error`
  - `func (s *Store) DeleteProject(id int64) error` — cascades tasks/time entries.

- [ ] **Step 1: Write the failing test**

Create `internal/store/project_test.go`:
```go
package store

import "testing"

func TestCreateAndListProjects(t *testing.T) {
	s := newTestStore(t)
	p, err := s.CreateProject("Work")
	if err != nil {
		t.Fatalf("CreateProject: %v", err)
	}
	if p.ID == 0 || p.Name != "Work" {
		t.Fatalf("unexpected project: %+v", p)
	}
	projects, err := s.ListProjects()
	if err != nil {
		t.Fatalf("ListProjects: %v", err)
	}
	// Inbox (seeded) + Work
	if len(projects) != 2 {
		t.Fatalf("want 2 projects, got %d", len(projects))
	}
	if projects[1].Name != "Work" {
		t.Errorf("want second project 'Work', got %q", projects[1].Name)
	}
}

func TestRenameProject(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Wrok")
	if err := s.RenameProject(p.ID, "Work"); err != nil {
		t.Fatalf("RenameProject: %v", err)
	}
	projects, _ := s.ListProjects()
	if projects[1].Name != "Work" {
		t.Errorf("rename failed, got %q", projects[1].Name)
	}
}

func TestDeleteProject(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Temp")
	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}
	projects, _ := s.ListProjects()
	if len(projects) != 1 {
		t.Errorf("want 1 project after delete, got %d", len(projects))
	}
}
```
Also add `TestOpenSeedsDefaultProject` (from Task 1's note) to `store_test.go` now that `ListProjects` exists.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run TestCreateAndListProjects -v`
Expected: FAIL — `undefined: (*Store).CreateProject`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/store/project.go`:
```go
package store

import "fmt"

func (s *Store) CreateProject(name string) (Project, error) {
	now := s.now()
	res, err := s.db.Exec(
		`INSERT INTO projects (name, created_at) VALUES (?, ?)`, name, now,
	)
	if err != nil {
		return Project{}, fmt.Errorf("create project: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Project{}, fmt.Errorf("create project id: %w", err)
	}
	return Project{ID: id, Name: name, CreatedAt: now}, nil
}

func (s *Store) ListProjects() ([]Project, error) {
	rows, err := s.db.Query(
		`SELECT id, name, created_at FROM projects ORDER BY id`,
	)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	defer rows.Close()
	var out []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Name, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan project: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (s *Store) RenameProject(id int64, name string) error {
	_, err := s.db.Exec(`UPDATE projects SET name = ? WHERE id = ?`, name, id)
	if err != nil {
		return fmt.Errorf("rename project: %w", err)
	}
	return nil
}

func (s *Store) DeleteProject(id int64) error {
	_, err := s.db.Exec(`DELETE FROM projects WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run 'Project|SeedsDefault' -v`
Expected: PASS for all project + seed tests.

- [ ] **Step 5: Commit**

```bash
git add internal/store/project.go internal/store/project_test.go internal/store/store_test.go
git commit -m "Add project CRUD to store"
```

---

### Task 3: Task CRUD + reorder

**Files:**
- Create: `internal/store/task.go`
- Test: `internal/store/task_test.go`

**Interfaces:**
- Consumes: `Store`, `Task`, `CreateProject` (Tasks 1–2).
- Produces:
  - `func (s *Store) CreateTask(projectID int64, title string) (Task, error)` — appends with `sort_order = max(sort_order)+1` within the project.
  - `func (s *Store) ListTasks(projectID int64) ([]Task, error)` — ordered by `sort_order`.
  - `func (s *Store) UpdateTask(id int64, title, notes string) error`
  - `func (s *Store) SetTaskDone(id int64, done bool) error` — sets `done_at = now` when done, `NULL` when undone.
  - `func (s *Store) DeleteTask(id int64) error`
  - `func (s *Store) MoveTask(id int64, delta int) error` — `delta` is `-1` (up) or `+1` (down); swaps `sort_order` with the adjacent task in the same project. No-op at the ends.

- [ ] **Step 1: Write the failing test**

Create `internal/store/task_test.go`:
```go
package store

import "testing"

func projectID(t *testing.T, s *Store) int64 {
	t.Helper()
	projects, err := s.ListProjects()
	if err != nil || len(projects) == 0 {
		t.Fatalf("need a project: %v", err)
	}
	return projects[0].ID
}

func TestCreateAndListTasks(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	a, err := s.CreateTask(pid, "First")
	if err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	if a.Title != "First" || a.ProjectID != pid || a.Done {
		t.Fatalf("unexpected task: %+v", a)
	}
	if _, err := s.CreateTask(pid, "Second"); err != nil {
		t.Fatalf("CreateTask: %v", err)
	}
	tasks, err := s.ListTasks(pid)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(tasks) != 2 || tasks[0].Title != "First" || tasks[1].Title != "Second" {
		t.Fatalf("unexpected order: %+v", tasks)
	}
	if tasks[1].SortOrder <= tasks[0].SortOrder {
		t.Errorf("sort_order not increasing: %+v", tasks)
	}
}

func TestSetTaskDoneSetsDoneAt(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Finish me")
	if err := s.SetTaskDone(tk.ID, true); err != nil {
		t.Fatalf("SetTaskDone: %v", err)
	}
	tasks, _ := s.ListTasks(pid)
	if !tasks[0].Done || tasks[0].DoneAt == nil {
		t.Fatalf("want done with done_at set, got %+v", tasks[0])
	}
	if err := s.SetTaskDone(tk.ID, false); err != nil {
		t.Fatalf("SetTaskDone undo: %v", err)
	}
	tasks, _ = s.ListTasks(pid)
	if tasks[0].Done || tasks[0].DoneAt != nil {
		t.Fatalf("want undone with nil done_at, got %+v", tasks[0])
	}
}

func TestUpdateTask(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Old")
	if err := s.UpdateTask(tk.ID, "New", "some notes"); err != nil {
		t.Fatalf("UpdateTask: %v", err)
	}
	tasks, _ := s.ListTasks(pid)
	if tasks[0].Title != "New" || tasks[0].Notes != "some notes" {
		t.Fatalf("update failed: %+v", tasks[0])
	}
}

func TestDeleteTask(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Doomed")
	if err := s.DeleteTask(tk.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	tasks, _ := s.ListTasks(pid)
	if len(tasks) != 0 {
		t.Fatalf("want 0 tasks, got %d", len(tasks))
	}
}

func TestMoveTaskSwapsOrder(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	s.CreateTask(pid, "A")
	s.CreateTask(pid, "B")
	tasks, _ := s.ListTasks(pid)
	// Move B (index 1) up.
	if err := s.MoveTask(tasks[1].ID, -1); err != nil {
		t.Fatalf("MoveTask: %v", err)
	}
	tasks, _ = s.ListTasks(pid)
	if tasks[0].Title != "B" || tasks[1].Title != "A" {
		t.Fatalf("want B,A after move-up, got %s,%s", tasks[0].Title, tasks[1].Title)
	}
	// Moving the top task up is a no-op.
	if err := s.MoveTask(tasks[0].ID, -1); err != nil {
		t.Fatalf("MoveTask no-op: %v", err)
	}
	tasks, _ = s.ListTasks(pid)
	if tasks[0].Title != "B" {
		t.Fatalf("no-op move changed order: %+v", tasks)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run Task -v`
Expected: FAIL — `undefined: (*Store).CreateTask`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/store/task.go`:
```go
package store

import (
	"database/sql"
	"fmt"
)

func (s *Store) CreateTask(projectID int64, title string) (Task, error) {
	now := s.now()
	var nextOrder int64
	err := s.db.QueryRow(
		`SELECT COALESCE(MAX(sort_order), 0) + 1 FROM tasks WHERE project_id = ?`,
		projectID,
	).Scan(&nextOrder)
	if err != nil {
		return Task{}, fmt.Errorf("next sort order: %w", err)
	}
	res, err := s.db.Exec(
		`INSERT INTO tasks (project_id, title, notes, done, sort_order, created_at)
		 VALUES (?, ?, '', 0, ?, ?)`,
		projectID, title, nextOrder, now,
	)
	if err != nil {
		return Task{}, fmt.Errorf("create task: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Task{}, fmt.Errorf("create task id: %w", err)
	}
	return Task{
		ID: id, ProjectID: projectID, Title: title,
		SortOrder: nextOrder, CreatedAt: now,
	}, nil
}

func (s *Store) ListTasks(projectID int64) ([]Task, error) {
	rows, err := s.db.Query(
		`SELECT id, project_id, title, notes, done, sort_order, created_at, done_at
		 FROM tasks WHERE project_id = ? ORDER BY sort_order`,
		projectID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	defer rows.Close()
	var out []Task
	for rows.Next() {
		var t Task
		var doneAt sql.NullTime
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt,
		); err != nil {
			return nil, fmt.Errorf("scan task: %w", err)
		}
		if doneAt.Valid {
			dt := doneAt.Time
			t.DoneAt = &dt
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) UpdateTask(id int64, title, notes string) error {
	_, err := s.db.Exec(
		`UPDATE tasks SET title = ?, notes = ? WHERE id = ?`, title, notes, id,
	)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	return nil
}

func (s *Store) SetTaskDone(id int64, done bool) error {
	if done {
		_, err := s.db.Exec(
			`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`, s.now(), id,
		)
		if err != nil {
			return fmt.Errorf("set task done: %w", err)
		}
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE tasks SET done = 0, done_at = NULL WHERE id = ?`, id,
	)
	if err != nil {
		return fmt.Errorf("set task undone: %w", err)
	}
	return nil
}

func (s *Store) DeleteTask(id int64) error {
	_, err := s.db.Exec(`DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	return nil
}

// MoveTask swaps the sort_order of the task with its neighbor in the same
// project. delta = -1 moves up (toward the top), +1 moves down. No-op at ends.
func (s *Store) MoveTask(id int64, delta int) error {
	if delta != -1 && delta != 1 {
		return fmt.Errorf("move task: delta must be -1 or 1, got %d", delta)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("move task begin: %w", err)
	}
	defer tx.Rollback()

	var projectID, sortOrder int64
	err = tx.QueryRow(
		`SELECT project_id, sort_order FROM tasks WHERE id = ?`, id,
	).Scan(&projectID, &sortOrder)
	if err != nil {
		return fmt.Errorf("move task lookup: %w", err)
	}

	// Find the adjacent task in the move direction.
	var neighborID, neighborOrder int64
	var q string
	if delta < 0 {
		q = `SELECT id, sort_order FROM tasks
		     WHERE project_id = ? AND sort_order < ?
		     ORDER BY sort_order DESC LIMIT 1`
	} else {
		q = `SELECT id, sort_order FROM tasks
		     WHERE project_id = ? AND sort_order > ?
		     ORDER BY sort_order ASC LIMIT 1`
	}
	err = tx.QueryRow(q, projectID, sortOrder).Scan(&neighborID, &neighborOrder)
	if err == sql.ErrNoRows {
		return nil // at an end; nothing to swap
	}
	if err != nil {
		return fmt.Errorf("move task neighbor: %w", err)
	}

	if _, err := tx.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, neighborOrder, id); err != nil {
		return fmt.Errorf("move task swap self: %w", err)
	}
	if _, err := tx.Exec(`UPDATE tasks SET sort_order = ? WHERE id = ?`, sortOrder, neighborID); err != nil {
		return fmt.Errorf("move task swap neighbor: %w", err)
	}
	return tx.Commit()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run Task -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/task.go internal/store/task_test.go
git commit -m "Add task CRUD and reorder to store"
```

---

### Task 4: Time tracking

**Files:**
- Create: `internal/store/timelog.go`
- Test: `internal/store/timelog_test.go`

**Interfaces:**
- Consumes: `Store`, `TimeEntry`, `CreateTask` (Tasks 1,3).
- Produces:
  - `func (s *Store) StartTimer(taskID int64) (TimeEntry, error)` — in one transaction, stops any running entry (`ended_at = now`) then inserts a new open entry for `taskID`.
  - `func (s *Store) StopTimer() error` — closes the running entry if any; no-op otherwise.
  - `func (s *Store) RunningEntry() (*TimeEntry, error)` — the open entry (`ended_at IS NULL`), or `nil`.
  - `func (s *Store) TaskDuration(taskID int64) (time.Duration, error)` — sum of `ended_at - started_at` over that task's closed entries.

- [ ] **Step 1: Write the failing test**

Create `internal/store/timelog_test.go`:
```go
package store

import (
	"testing"
	"time"
)

// advancingClock returns times that step forward by one hour per call,
// starting from the fixed base. Lets us assert deterministic durations.
func advancingClock() func() time.Time {
	base := time.Date(2026, 7, 9, 9, 0, 0, 0, time.UTC)
	n := 0
	return func() time.Time {
		t := base.Add(time.Duration(n) * time.Hour)
		n++
		return t
	}
}

func TestStartTimerCreatesRunningEntry(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Track me")
	entry, err := s.StartTimer(tk.ID)
	if err != nil {
		t.Fatalf("StartTimer: %v", err)
	}
	if entry.TaskID != tk.ID || entry.EndedAt != nil {
		t.Fatalf("unexpected entry: %+v", entry)
	}
	running, err := s.RunningEntry()
	if err != nil {
		t.Fatalf("RunningEntry: %v", err)
	}
	if running == nil || running.TaskID != tk.ID {
		t.Fatalf("want running entry for task %d, got %+v", tk.ID, running)
	}
}

func TestStartTimerStopsPrevious(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	a, _ := s.CreateTask(pid, "A")
	b, _ := s.CreateTask(pid, "B")
	if _, err := s.StartTimer(a.ID); err != nil {
		t.Fatalf("StartTimer A: %v", err)
	}
	if _, err := s.StartTimer(b.ID); err != nil {
		t.Fatalf("StartTimer B: %v", err)
	}
	running, _ := s.RunningEntry()
	if running == nil || running.TaskID != b.ID {
		t.Fatalf("want B running, got %+v", running)
	}
	// Exactly one entry should be open.
	var open int
	s.db.QueryRow(`SELECT COUNT(*) FROM time_entries WHERE ended_at IS NULL`).Scan(&open)
	if open != 1 {
		t.Fatalf("want 1 open entry, got %d", open)
	}
}

func TestTaskDurationSumsClosedEntries(t *testing.T) {
	s := newTestStore(t)
	s.now = advancingClock() // 9:00, 10:00, 11:00, ...
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Work") // consumes 9:00 (created_at)
	// start -> 10:00, stop -> 11:00  => 1h
	if _, err := s.StartTimer(tk.ID); err != nil {
		t.Fatalf("StartTimer: %v", err)
	}
	if err := s.StopTimer(); err != nil {
		t.Fatalf("StopTimer: %v", err)
	}
	d, err := s.TaskDuration(tk.ID)
	if err != nil {
		t.Fatalf("TaskDuration: %v", err)
	}
	if d != time.Hour {
		t.Fatalf("want 1h, got %s", d)
	}
}

func TestStopTimerNoRunningIsNoop(t *testing.T) {
	s := newTestStore(t)
	if err := s.StopTimer(); err != nil {
		t.Fatalf("StopTimer with none running should be nil, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run Timer -v`
Expected: FAIL — `undefined: (*Store).StartTimer`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/store/timelog.go`:
```go
package store

import (
	"database/sql"
	"fmt"
	"time"
)

// StartTimer stops any running entry and starts a new one for taskID,
// atomically, so at most one timer is ever running.
func (s *Store) StartTimer(taskID int64) (TimeEntry, error) {
	now := s.now()
	tx, err := s.db.Begin()
	if err != nil {
		return TimeEntry{}, fmt.Errorf("start timer begin: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(
		`UPDATE time_entries SET ended_at = ? WHERE ended_at IS NULL`, now,
	); err != nil {
		return TimeEntry{}, fmt.Errorf("start timer stop previous: %w", err)
	}
	res, err := tx.Exec(
		`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, NULL)`,
		taskID, now,
	)
	if err != nil {
		return TimeEntry{}, fmt.Errorf("start timer insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return TimeEntry{}, fmt.Errorf("start timer id: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return TimeEntry{}, fmt.Errorf("start timer commit: %w", err)
	}
	return TimeEntry{ID: id, TaskID: taskID, StartedAt: now}, nil
}

func (s *Store) StopTimer() error {
	_, err := s.db.Exec(
		`UPDATE time_entries SET ended_at = ? WHERE ended_at IS NULL`, s.now(),
	)
	if err != nil {
		return fmt.Errorf("stop timer: %w", err)
	}
	return nil
}

func (s *Store) RunningEntry() (*TimeEntry, error) {
	var e TimeEntry
	err := s.db.QueryRow(
		`SELECT id, task_id, started_at FROM time_entries
		 WHERE ended_at IS NULL ORDER BY started_at DESC LIMIT 1`,
	).Scan(&e.ID, &e.TaskID, &e.StartedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("running entry: %w", err)
	}
	return &e, nil
}

func (s *Store) TaskDuration(taskID int64) (time.Duration, error) {
	rows, err := s.db.Query(
		`SELECT started_at, ended_at FROM time_entries
		 WHERE task_id = ? AND ended_at IS NOT NULL`, taskID,
	)
	if err != nil {
		return 0, fmt.Errorf("task duration: %w", err)
	}
	defer rows.Close()
	var total time.Duration
	for rows.Next() {
		var start, end time.Time
		if err := rows.Scan(&start, &end); err != nil {
			return 0, fmt.Errorf("scan duration: %w", err)
		}
		total += end.Sub(start)
	}
	return total, rows.Err()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/store/ -run Timer -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/store/timelog.go internal/store/timelog_test.go
git commit -m "Add time tracking to store"
```

---

### Task 5: Daily summary

**Files:**
- Create: `internal/store/summary.go`
- Test: `internal/store/summary_test.go`

**Interfaces:**
- Consumes: `Store`, `Task` (Tasks 1,3,4).
- Produces:
  - ```go
    type TaskDuration struct {
        Task     Task
        Duration time.Duration
    }
    type DailySummary struct {
        Day       time.Time       // midnight UTC of the summarized day
        Completed []Task          // tasks with done_at within the day
        Times     []TaskDuration  // per-task closed-entry time started within the day
        Total     time.Duration   // sum of Times durations
    }
    ```
  - `func (s *Store) DailySummary(day time.Time) (DailySummary, error)` — `day` is any time within the target day; the method truncates to `[midnight, midnight+24h)` in UTC.

- [ ] **Step 1: Write the failing test**

Create `internal/store/summary_test.go`:
```go
package store

import (
	"testing"
	"time"
)

func TestDailySummary(t *testing.T) {
	s := newTestStore(t)
	s.now = advancingClock() // 9:00, 10:00, 11:00, 12:00 on 2026-07-09
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Report") // created_at 9:00
	s.StartTimer(tk.ID)                  // started 10:00
	s.StopTimer()                        // ended 11:00 => 1h
	s.SetTaskDone(tk.ID, true)           // done_at 12:00

	sum, err := s.DailySummary(time.Date(2026, 7, 9, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("DailySummary: %v", err)
	}
	if len(sum.Completed) != 1 || sum.Completed[0].ID != tk.ID {
		t.Fatalf("want 1 completed task, got %+v", sum.Completed)
	}
	if sum.Total != time.Hour {
		t.Fatalf("want total 1h, got %s", sum.Total)
	}
	if len(sum.Times) != 1 || sum.Times[0].Duration != time.Hour {
		t.Fatalf("want 1 timed task of 1h, got %+v", sum.Times)
	}
}

func TestDailySummaryExcludesOtherDays(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Yesterday")
	// Force done_at into the prior day.
	s.db.Exec(`UPDATE tasks SET done = 1, done_at = ? WHERE id = ?`,
		time.Date(2026, 7, 8, 10, 0, 0, 0, time.UTC), tk.ID)

	sum, err := s.DailySummary(time.Date(2026, 7, 9, 10, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("DailySummary: %v", err)
	}
	if len(sum.Completed) != 0 {
		t.Fatalf("want 0 completed today, got %d", len(sum.Completed))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/store/ -run Summary -v`
Expected: FAIL — `undefined: (*Store).DailySummary`.

- [ ] **Step 3: Write minimal implementation**

Create `internal/store/summary.go`:
```go
package store

import (
	"fmt"
	"time"
)

type TaskDuration struct {
	Task     Task
	Duration time.Duration
}

type DailySummary struct {
	Day       time.Time
	Completed []Task
	Times     []TaskDuration
	Total     time.Duration
}

func (s *Store) DailySummary(day time.Time) (DailySummary, error) {
	day = day.UTC()
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	sum := DailySummary{Day: start}

	// Completed tasks with done_at within the day.
	rows, err := s.db.Query(
		`SELECT id, project_id, title, notes, done, sort_order, created_at, done_at
		 FROM tasks
		 WHERE done = 1 AND done_at >= ? AND done_at < ?
		 ORDER BY done_at`, start, end,
	)
	if err != nil {
		return sum, fmt.Errorf("summary completed: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var t Task
		var doneAt time.Time
		if err := rows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt,
		); err != nil {
			return sum, fmt.Errorf("scan completed: %w", err)
		}
		t.DoneAt = &doneAt
		sum.Completed = append(sum.Completed, t)
	}
	if err := rows.Err(); err != nil {
		return sum, err
	}

	// Per-task time from closed entries started within the day.
	trows, err := s.db.Query(
		`SELECT t.id, t.project_id, t.title, t.notes, t.done, t.sort_order,
		        t.created_at, t.done_at,
		        SUM((julianday(e.ended_at) - julianday(e.started_at)) * 86400.0) AS secs
		 FROM time_entries e
		 JOIN tasks t ON t.id = e.task_id
		 WHERE e.ended_at IS NOT NULL AND e.started_at >= ? AND e.started_at < ?
		 GROUP BY t.id
		 ORDER BY secs DESC`, start, end,
	)
	if err != nil {
		return sum, fmt.Errorf("summary times: %w", err)
	}
	defer trows.Close()
	for trows.Next() {
		var t Task
		var doneAt *time.Time
		var secs float64
		if err := trows.Scan(
			&t.ID, &t.ProjectID, &t.Title, &t.Notes, &t.Done,
			&t.SortOrder, &t.CreatedAt, &doneAt, &secs,
		); err != nil {
			return sum, fmt.Errorf("scan times: %w", err)
		}
		t.DoneAt = doneAt
		d := time.Duration(secs * float64(time.Second))
		sum.Times = append(sum.Times, TaskDuration{Task: t, Duration: d})
		sum.Total += d
	}
	return sum, trows.Err()
}
```
Note: `done_at` here is scanned into `*time.Time` in the second query (nullable) but `time.Time` in the first (guaranteed non-null by the `done = 1 AND done_at >= ?` filter).

- [ ] **Step 4: Run tests to verify they pass, then the whole store package**

Run: `go test ./internal/store/ -v`
Expected: PASS for every store test.

- [ ] **Step 5: Commit**

```bash
git add internal/store/summary.go internal/store/summary_test.go
git commit -m "Add daily summary to store"
```

---

### Task 6: TUI styles + root model and screen routing

**Files:**
- Create: `internal/tui/styles.go`
- Create: `internal/tui/app.go`
- Test: `internal/tui/app_test.go`

**Interfaces:**
- Consumes: `store.Store`, `store.Project` (Tasks 1–5).
- Produces:
  - `type screen int` with `screenTasks`, `screenProjects`, `screenSummary`.
  - `type Model struct { ... }` implementing `tea.Model` (`Init`, `Update`, `View`).
  - `func New(s *store.Store) (Model, error)` — loads projects, selects the first as active, loads its tasks.
  - `func (m Model) activeProject() store.Project`
  - A `statusMsg string` field set by store errors, shown in the footer.
  - Screen-specific state added by later tasks lives in this struct; Tasks 7–10 extend `Update`/`View` via a `switch m.screen`.
  - `var ( titleStyle, selectedStyle, doneStyle, statusStyle, helpStyle lipgloss.Style )` in `styles.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/app_test.go`:
```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/liam-dev-c/what-was-next/internal/store"
)

func newModel(t *testing.T) Model {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("store.Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	m, err := New(s)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return m
}

func TestNewSelectsDefaultProject(t *testing.T) {
	m := newModel(t)
	if m.activeProject().Name != "Inbox" {
		t.Fatalf("want active project 'Inbox', got %q", m.activeProject().Name)
	}
	if m.screen != screenTasks {
		t.Fatalf("want initial screen screenTasks, got %v", m.screen)
	}
}

func TestQuitKey(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("want quit command on 'q', got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run TestNew -v`
Expected: FAIL — `undefined: New` / `undefined: Model`.

- [ ] **Step 3: Write styles**

Create `internal/tui/styles.go`:
```go
package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("212")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Bold(true)

	doneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Strikethrough(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("203"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			MarginTop(1)
)
```

- [ ] **Step 4: Write the root model**

Create `internal/tui/app.go`:
```go
// Package tui is the Bubble Tea terminal UI. It holds view state only and
// delegates every mutation to the store, reloading after each change.
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/liam-dev-c/what-was-next/internal/store"
)

type screen int

const (
	screenTasks screen = iota
	screenProjects
	screenSummary
)

// Model is the root Bubble Tea model. Screen-specific state is added by the
// task/project/timer/summary tasks; Update/View dispatch on m.screen.
type Model struct {
	store    *store.Store
	screen   screen
	projects []store.Project
	active   int // index into projects

	tasks    []store.Task
	cursor   int // selected task index on the tasks screen
	status   string

	// input state (task add/edit) — populated in Task 7
	editing  bool
	editID   int64 // 0 == adding a new task
	input    textInput

	// project switcher cursor — populated in Task 8
	projCursor int

	// summary snapshot — populated in Task 10
	summary store.DailySummary

	width  int
	height int
}

// textInput is a thin alias so app.go compiles before Task 7 wires the real
// bubbles/textinput; Task 7 replaces this with textinput.Model.
type textInput = struct{ Value string }

func New(s *store.Store) (Model, error) {
	m := Model{store: s, screen: screenTasks}
	if err := m.reloadProjects(); err != nil {
		return Model{}, err
	}
	if err := m.reloadTasks(); err != nil {
		return Model{}, err
	}
	return m, nil
}

func (m *Model) reloadProjects() error {
	projects, err := m.store.ListProjects()
	if err != nil {
		return fmt.Errorf("load projects: %w", err)
	}
	m.projects = projects
	if m.active >= len(m.projects) {
		m.active = 0
	}
	return nil
}

func (m *Model) reloadTasks() error {
	if len(m.projects) == 0 {
		m.tasks = nil
		return nil
	}
	tasks, err := m.store.ListTasks(m.activeProject().ID)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}
	m.tasks = tasks
	if m.cursor >= len(m.tasks) {
		m.cursor = max(0, len(m.tasks)-1)
	}
	return nil
}

func (m Model) activeProject() store.Project {
	if len(m.projects) == 0 {
		return store.Project{}
	}
	return m.projects[m.active]
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		// Global quit (only when not typing in an input; Task 7 guards this).
		if !m.editing && (msg.String() == "q" || msg.String() == "ctrl+c") {
			return m, tea.Quit
		}
		switch m.screen {
		case screenTasks:
			return m.updateTasks(msg)
		case screenProjects:
			return m.updateProjects(msg)
		case screenSummary:
			return m.updateSummary(msg)
		}
	}
	return m, nil
}

func (m Model) View() string {
	switch m.screen {
	case screenProjects:
		return m.viewProjects()
	case screenSummary:
		return m.viewSummary()
	default:
		return m.viewTasks()
	}
}
```

- [ ] **Step 5: Add temporary stubs so the package compiles**

The `switch` above references methods built in Tasks 7–10. Add a stub file `internal/tui/stubs.go` so Task 6 compiles and tests run. Each later task deletes its stub and adds the real method.
```go
package tui

import tea "github.com/charmbracelet/bubbletea"

// Temporary stubs — replaced by Tasks 7–10. Delete each as its task lands.
func (m Model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd)    { return m, nil }
func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) { return m, nil }
func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd)  { return m, nil }
func (m Model) viewTasks() string    { return "tasks" }
func (m Model) viewProjects() string { return "projects" }
func (m Model) viewSummary() string  { return "summary" }
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/tui/ -run 'TestNew|TestQuit' -v`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/tui/styles.go internal/tui/app.go internal/tui/stubs.go internal/tui/app_test.go
git commit -m "Add TUI root model, styles, and screen routing"
```

---

### Task 7: Task-list screen (navigate, add, edit, complete, delete, reorder, timer, screen switches)

**Files:**
- Modify: `internal/tui/app.go` (replace `textInput` alias with real textinput; add helpers)
- Create: `internal/tui/tasks.go`
- Delete from `internal/tui/stubs.go`: `updateTasks`, `viewTasks`
- Test: `internal/tui/tasks_test.go`

**Interfaces:**
- Consumes: `Model`, store task/timer methods.
- Produces: real `func (m Model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd)` and `func (m Model) viewTasks() string`. Introduces `m.input textinput.Model` (replacing the alias) and `func (m *Model) setStatus(err error)`.

- [ ] **Step 1: Replace the textInput alias with the real component**

In `internal/tui/app.go`, change the import block to include textinput and update the field:
```go
import (
	"fmt"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/liam-dev-c/what-was-next/internal/store"
)
```
Replace `input textInput` with `input textinput.Model` in the struct, and delete the `type textInput = struct{ Value string }` line.

Add a status helper at the bottom of `app.go`:
```go
func (m *Model) setStatus(err error) {
	if err != nil {
		m.status = err.Error()
	}
}
```

- [ ] **Step 2: Write the failing test**

Create `internal/tui/tasks_test.go`:
```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestAddTaskFlow(t *testing.T) {
	m := newModel(t)
	// Press 'a' to start adding.
	mi, _ := m.updateTasks(key('a'))
	m = mi.(Model)
	if !m.editing {
		t.Fatal("want editing mode after 'a'")
	}
	// Type "Hello".
	for _, r := range "Hello" {
		mi, _ = m.Update(key(r))
		m = mi.(Model)
	}
	// Enter to commit.
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(Model)
	if m.editing {
		t.Fatal("want editing off after Enter")
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Hello" {
		t.Fatalf("want 1 task 'Hello', got %+v", m.tasks)
	}
}

func TestToggleDone(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	mi, _ := m.updateTasks(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(Model)
	if !m.tasks[0].Done {
		t.Fatal("want task done after Enter toggle")
	}
}

func TestDeleteTask(t *testing.T) {
	m := newModel(t)
	m.store.CreateTask(m.activeProject().ID, "Task")
	m.reloadTasks()
	mi, _ := m.updateTasks(key('d'))
	m = mi.(Model)
	if len(m.tasks) != 0 {
		t.Fatalf("want 0 tasks after delete, got %d", len(m.tasks))
	}
}

func TestSwitchToProjectsAndSummary(t *testing.T) {
	m := newModel(t)
	mi, _ := m.updateTasks(key('p'))
	if mi.(Model).screen != screenProjects {
		t.Fatal("want screenProjects after 'p'")
	}
	mi, _ = m.updateTasks(key('s'))
	if mi.(Model).screen != screenSummary {
		t.Fatal("want screenSummary after 's'")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'TestAddTaskFlow' -v`
Expected: FAIL — duplicate/compile error until the stub is removed and `tasks.go` added. (Also compile error on `textinput` until Step 4.)

- [ ] **Step 4: Delete the tasks stubs**

In `internal/tui/stubs.go`, delete the `updateTasks` and `viewTasks` stub functions (keep the project/summary stubs for now).

- [ ] **Step 5: Write the tasks screen**

Create `internal/tui/tasks.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateTasks(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.updateTaskInput(msg)
	}
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
		m.beginEdit(0, "")
		return m, textinput.Blink
	case "e":
		if t, ok := m.selectedTask(); ok {
			m.beginEdit(t.ID, t.Title)
			return m, textinput.Blink
		}
	case "enter", " ":
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
		m.projCursor = m.active
		m.screen = screenProjects
	case "s":
		m.loadSummary()
		m.screen = screenSummary
	}
	return m, nil
}

func (m *Model) toggleTimer(taskID int64) {
	running, err := m.store.RunningEntry()
	if err != nil {
		m.setStatus(err)
		return
	}
	if running != nil && running.TaskID == taskID {
		m.setStatus(m.store.StopTimer())
		return
	}
	_, err = m.store.StartTimer(taskID)
	m.setStatus(err)
}

func (m *Model) beginEdit(id int64, initial string) {
	ti := textinput.New()
	ti.SetValue(initial)
	ti.Focus()
	ti.CursorEnd()
	m.input = ti
	m.editing = true
	m.editID = id
}

func (m Model) updateTaskInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		title := strings.TrimSpace(m.input.Value())
		if title != "" {
			if m.editID == 0 {
				_, err := m.store.CreateTask(m.activeProject().ID, title)
				m.setStatus(err)
			} else {
				m.setStatus(m.store.UpdateTask(m.editID, title, ""))
			}
			m.setStatus(m.reloadTasks())
		}
		m.editing = false
		return m, nil
	case tea.KeyEsc:
		m.editing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) selectedTask() (task, bool) {
	if m.cursor < 0 || m.cursor >= len(m.tasks) {
		return task{}, false
	}
	return m.tasks[m.cursor], true
}

// task is an alias for store.Task used above; declared here to keep imports local.
type task = storeTask

func (m Model) viewTasks() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("what was next — " + m.activeProject().Name))
	b.WriteString("\n")

	running, _ := m.store.RunningEntry()
	for i, t := range m.tasks {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		box := "[ ]"
		if t.Done {
			box = "[x]"
		}
		clock := "  "
		if running != nil && running.TaskID == t.ID {
			clock = "⏱ "
		}
		line := fmt.Sprintf("%s%s %s%s", cursor, box, clock, t.Title)
		switch {
		case i == m.cursor:
			line = selectedStyle.Render(line)
		case t.Done:
			line = doneStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if len(m.tasks) == 0 {
		b.WriteString(helpStyle.Render("No tasks yet — press 'a' to add one.\n"))
	}

	if m.editing {
		verb := "New task"
		if m.editID != 0 {
			verb = "Edit task"
		}
		b.WriteString("\n" + verb + ": " + m.input.View() + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	b.WriteString(helpStyle.Render(
		"\na add · e edit · enter done · d del · J/K move · t timer · p projects · s summary · q quit"))
	return b.String()
}
```
Add to `internal/tui/app.go` a type alias so `tasks.go` can name the store task without re-importing under a new name:
```go
type storeTask = store.Task
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: PASS (task flows + earlier root tests).

- [ ] **Step 7: Commit**

```bash
git add internal/tui/app.go internal/tui/tasks.go internal/tui/stubs.go internal/tui/tasks_test.go
git commit -m "Add task-list screen with add/edit/complete/delete/reorder/timer"
```

---

### Task 8: Project switcher screen

**Files:**
- Create: `internal/tui/projects.go`
- Delete from `internal/tui/stubs.go`: `updateProjects`, `viewProjects`
- Test: `internal/tui/projects_test.go`

**Interfaces:**
- Consumes: `Model`, store project methods.
- Produces: real `func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd)` and `func (m Model) viewProjects() string`. Reuses `m.input`/`m.editing` for project creation, distinguished by `m.screen == screenProjects`.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/projects_test.go`:
```go
package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSelectProjectReloadsTasks(t *testing.T) {
	m := newModel(t)
	p, _ := m.store.CreateProject("Work")
	m.store.CreateTask(p.ID, "Work task")
	m.reloadProjects()
	m.screen = screenProjects
	m.projCursor = 0

	// Move down to "Work" (index 1) and select it.
	mi, _ := m.updateProjects(key('j'))
	m = mi.(Model)
	mi, _ = m.updateProjects(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(Model)

	if m.screen != screenTasks {
		t.Fatal("want return to tasks after selecting project")
	}
	if m.activeProject().Name != "Work" {
		t.Fatalf("want active 'Work', got %q", m.activeProject().Name)
	}
	if len(m.tasks) != 1 || m.tasks[0].Title != "Work task" {
		t.Fatalf("want Work's tasks loaded, got %+v", m.tasks)
	}
}

func TestAddProjectFlow(t *testing.T) {
	m := newModel(t)
	m.screen = screenProjects
	mi, _ := m.updateProjects(key('a'))
	m = mi.(Model)
	for _, r := range "Side" {
		mi, _ = m.Update(key(r))
		m = mi.(Model)
	}
	mi, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = mi.(Model)
	names := make([]string, len(m.projects))
	for i, p := range m.projects {
		names[i] = p.Name
	}
	found := false
	for _, n := range names {
		if n == "Side" {
			found = true
		}
	}
	if !found {
		t.Fatalf("want a 'Side' project, got %v", names)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run Project -v`
Expected: FAIL — stub returns unchanged model.

- [ ] **Step 3: Delete the project stubs**

In `internal/tui/stubs.go`, delete `updateProjects` and `viewProjects`.

- [ ] **Step 4: Write the projects screen**

Create `internal/tui/projects.go`:
```go
package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) updateProjects(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.editing {
		return m.updateProjectInput(msg)
	}
	switch msg.String() {
	case "j", "down":
		if m.projCursor < len(m.projects)-1 {
			m.projCursor++
		}
	case "k", "up":
		if m.projCursor > 0 {
			m.projCursor--
		}
	case "a":
		ti := textinput.New()
		ti.Focus()
		m.input = ti
		m.editing = true
		m.editID = 0
		return m, textinput.Blink
	case "enter", " ":
		m.active = m.projCursor
		m.cursor = 0
		m.setStatus(m.reloadTasks())
		m.screen = screenTasks
	case "esc", "p":
		m.screen = screenTasks
	}
	return m, nil
}

func (m Model) updateProjectInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		name := strings.TrimSpace(m.input.Value())
		if name != "" {
			_, err := m.store.CreateProject(name)
			m.setStatus(err)
			m.setStatus(m.reloadProjects())
		}
		m.editing = false
		return m, nil
	case tea.KeyEsc:
		m.editing = false
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) viewProjects() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Projects"))
	b.WriteString("\n")
	for i, p := range m.projects {
		cursor := "  "
		if i == m.projCursor {
			cursor = "> "
		}
		marker := "  "
		if i == m.active {
			marker = "* "
		}
		line := cursor + marker + p.Name
		if i == m.projCursor {
			line = selectedStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}
	if m.editing {
		b.WriteString("\nNew project: " + m.input.View() + "\n")
	}
	if m.status != "" {
		b.WriteString("\n" + statusStyle.Render(m.status))
	}
	b.WriteString(helpStyle.Render(
		"\nj/k move · enter select · a add · esc back"))
	return b.String()
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/projects.go internal/tui/stubs.go internal/tui/projects_test.go
git commit -m "Add project switcher screen"
```

---

### Task 9: Live-ticking timer display

**Files:**
- Modify: `internal/tui/app.go` (add tick message + command, wire into Init/Update)
- Modify: `internal/tui/tasks.go` (show live elapsed for the running task)
- Test: `internal/tui/timer_test.go`

**Interfaces:**
- Consumes: `Model`, `store.RunningEntry`.
- Produces:
  - `type tickMsg struct{}`
  - `func tickCmd() tea.Cmd` — `tea.Tick(time.Second, ...)`.
  - `func (m Model) elapsedFor(taskID int64) (time.Duration, bool)` — accumulated closed time plus live time for a running task, from the store's clock via `time.Now`.
  - `Init` returns `tickCmd()`; `Update` handles `tickMsg` by returning `m, tickCmd()`.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/timer_test.go`:
```go
package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTickKeepsTicking(t *testing.T) {
	m := newModel(t)
	_, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Fatal("want a follow-up tick command")
	}
}

func TestElapsedForClosedTime(t *testing.T) {
	m := newModel(t)
	tk, _ := m.store.CreateTask(m.activeProject().ID, "Timed")
	m.reloadTasks()
	// Start and stop across a known gap using the store's injectable clock.
	// (RunningEntry-based live time is exercised manually; here we assert the
	// closed-time path returns a non-negative duration and an ok flag.)
	m.store.StartTimer(tk.ID)
	m.store.StopTimer()
	d, ok := m.elapsedFor(tk.ID)
	if !ok {
		t.Fatal("want ok for a task with time entries")
	}
	if d < 0 {
		t.Fatalf("want non-negative duration, got %s", d)
	}
	_ = time.Second
	_ = tea.KeyMsg{}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run 'Tick|Elapsed' -v`
Expected: FAIL — `undefined: tickMsg`.

- [ ] **Step 3: Add tick plumbing**

In `internal/tui/app.go`, add the import for `time` and, after the type declarations, add:
```go
type tickMsg struct{}

func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return tickMsg{} })
}

// elapsedFor returns total tracked time for a task, including the live segment
// if its timer is currently running.
func (m Model) elapsedFor(taskID int64) (time.Duration, bool) {
	closed, err := m.store.TaskDuration(taskID)
	if err != nil {
		return 0, false
	}
	total := closed
	running, err := m.store.RunningEntry()
	if err == nil && running != nil && running.TaskID == taskID {
		total += time.Since(running.StartedAt)
	}
	if total == 0 && closed == 0 {
		// Distinguish "no time at all" from "exactly zero": ok only if entries exist.
		return 0, running != nil && running.TaskID == taskID
	}
	return total, true
}
```
Change `Init` to start ticking:
```go
func (m Model) Init() tea.Cmd { return tickCmd() }
```
In `Update`, add a case at the top of the `switch msg := msg.(type)`:
```go
	case tickMsg:
		return m, tickCmd()
```

- [ ] **Step 4: Show live elapsed in the task list**

In `internal/tui/tasks.go`, inside `viewTasks`, replace the line-building block so a running task shows its elapsed time. Replace:
```go
		line := fmt.Sprintf("%s%s %s%s", cursor, box, clock, t.Title)
```
with:
```go
		suffix := ""
		if d, ok := m.elapsedFor(t.ID); ok {
			suffix = "  (" + fmtDuration(d) + ")"
		}
		line := fmt.Sprintf("%s%s %s%s%s", cursor, box, clock, t.Title, suffix)
```
And add a helper at the bottom of `tasks.go`:
```go
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	mnt := d / time.Minute
	d -= mnt * time.Minute
	s := d / time.Second
	if h > 0 {
		return fmt.Sprintf("%dh%02dm", h, mnt)
	}
	return fmt.Sprintf("%dm%02ds", mnt, s)
}
```
Add `"time"` to the imports of `tasks.go`.

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/app.go internal/tui/tasks.go internal/tui/timer_test.go
git commit -m "Add live-ticking timer display to task list"
```

---

### Task 10: Daily summary screen

**Files:**
- Create: `internal/tui/summary.go`
- Delete from `internal/tui/stubs.go`: `updateSummary`, `viewSummary` (and delete the now-empty `stubs.go`)
- Test: `internal/tui/summary_test.go`

**Interfaces:**
- Consumes: `Model`, `store.DailySummary`.
- Produces:
  - `func (m *Model) loadSummary()` — calls `m.store.DailySummary(time.Now())`, stores it in `m.summary`, sets status on error.
  - real `func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd)` — `esc`/`s` returns to tasks.
  - `func (m Model) viewSummary() string`.

- [ ] **Step 1: Write the failing test**

Create `internal/tui/summary_test.go`:
```go
package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSummaryLoadsAndRenders(t *testing.T) {
	m := newModel(t)
	tk, _ := m.store.CreateTask(m.activeProject().ID, "Wrote report")
	m.store.SetTaskDone(tk.ID, true)
	m.reloadTasks()

	m.loadSummary()
	out := m.viewSummary()
	if !strings.Contains(out, "Wrote report") {
		t.Fatalf("summary should list completed task, got:\n%s", out)
	}
}

func TestSummaryEscReturns(t *testing.T) {
	m := newModel(t)
	m.screen = screenSummary
	mi, _ := m.updateSummary(tea.KeyMsg{Type: tea.KeyEsc})
	if mi.(Model).screen != screenTasks {
		t.Fatal("want return to tasks on esc")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/tui/ -run Summary -v`
Expected: FAIL — stub `viewSummary` returns `"summary"`.

- [ ] **Step 3: Delete the summary stubs**

Delete `updateSummary` and `viewSummary` from `internal/tui/stubs.go`. If the file is now empty (only `package tui` and the import remain), delete the file entirely:
```bash
git rm internal/tui/stubs.go
```
(Only if empty. If other stubs remain, just edit it.)

- [ ] **Step 4: Write the summary screen**

Create `internal/tui/summary.go`:
```go
package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) loadSummary() {
	sum, err := m.store.DailySummary(time.Now())
	if err != nil {
		m.setStatus(err)
		return
	}
	m.summary = sum
}

func (m Model) updateSummary(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "s", "q":
		m.screen = screenTasks
	}
	return m, nil
}

func (m Model) viewSummary() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Today — " + m.summary.Day.Format("Mon 2 Jan 2006")))
	b.WriteString("\n")

	b.WriteString(selectedStyle.Render("Completed"))
	b.WriteString("\n")
	if len(m.summary.Completed) == 0 {
		b.WriteString(helpStyle.Render("Nothing completed yet today.\n"))
	}
	for _, t := range m.summary.Completed {
		b.WriteString("  [x] " + t.Title + "\n")
	}

	b.WriteString("\n" + selectedStyle.Render("Time tracked") + "\n")
	if len(m.summary.Times) == 0 {
		b.WriteString(helpStyle.Render("No time tracked today.\n"))
	}
	for _, td := range m.summary.Times {
		b.WriteString(fmt.Sprintf("  %-30s %s\n", td.Task.Title, fmtDuration(td.Duration)))
	}
	b.WriteString(fmt.Sprintf("\n  %-30s %s\n", "TOTAL", fmtDuration(m.summary.Total)))

	b.WriteString(helpStyle.Render("\nesc back · q quit"))
	return b.String()
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/tui/ -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/tui/summary.go
git rm --cached internal/tui/stubs.go 2>/dev/null || true
git commit -am "Add daily summary screen"
```

---

### Task 11: main entry point + wiring + manual run

**Files:**
- Create: `main.go`
- Modify: `README.md`
- Test: manual (build + run) — no unit test; this task wires already-tested units.

**Interfaces:**
- Consumes: `store.Open`, `tui.New`, `tea.NewProgram`.
- Produces: the `main` package and runnable binary.

- [ ] **Step 1: Write main.go**

Create `main.go`:
```go
// Command what-was-next is a terminal task manager and time tracker.
package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/liam-dev-c/what-was-next/internal/store"
	"github.com/liam-dev-c/what-was-next/internal/tui"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "what-was-next:", err)
		os.Exit(1)
	}
}

func run() error {
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
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
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

- [ ] **Step 2: Build and vet**

Run:
```bash
go build ./...
go vet ./...
go test ./...
```
Expected: build succeeds, vet clean, all tests pass.

- [ ] **Step 3: Manual smoke test**

Run: `go run .`
Verify by hand:
- Task list shows for project "Inbox".
- `a` → type a title → Enter adds it.
- `enter` toggles done (strikethrough + `[x]`).
- `t` starts the timer (⏱ marker + live-updating elapsed); `t` again stops it.
- `p` opens projects; `a` adds one; `enter` switches to it (task list changes).
- `s` shows today's summary with the completed task and tracked time.
- `q` quits.
Press `q` to exit.

- [ ] **Step 4: Update README**

Replace `README.md` contents:
```markdown
# what-was-next

A simple terminal task manager and time tracker — a small take on Super Productivity.

## Features

- Task list per project (add, edit, complete, delete, reorder)
- Per-task time tracking with a live timer
- Projects to group tasks
- Daily summary of completed tasks and time tracked

## Install

```bash
go install github.com/liam-dev-c/what-was-next@latest
```

## Usage

Run `what-was-next`. Data is stored at `~/.config/what-was-next/what-was-next.db`
(honoring `XDG_CONFIG_HOME`).

### Keys

| Key | Action |
|-----|--------|
| `a` | add task/project |
| `e` | edit task |
| `enter` / `space` | toggle done / select |
| `d` | delete task |
| `J` / `K` | move task down / up |
| `t` | start/stop timer on task |
| `p` | projects |
| `s` | daily summary |
| `esc` | back |
| `q` | quit |
```

- [ ] **Step 5: Commit**

```bash
git add main.go README.md
git commit -m "Add main entry point and usage docs"
```

---

## Self-Review Notes

- **Spec coverage:** Task CRUD → Task 3/7; Time tracking → Task 4/7/9; Projects → Task 2/8; Daily summary → Task 5/10; SQLite storage + location → Task 1/11; one-timer invariant → Task 4 (enforced in a transaction). All spec sections map to a task.
- **Type consistency:** store method names/signatures declared in Task interfaces are reused verbatim by TUI tasks (`StartTimer`, `RunningEntry`, `DailySummary`, `MoveTask(id, delta)`). `store.Task` is aliased once as `storeTask`/`task` in the TUI.
- **Clock injection** (Task 1) makes duration/summary tests deterministic without `Date.now`-style flakiness. Live TUI elapsed uses real `time.Since` and is verified manually (Task 11).
