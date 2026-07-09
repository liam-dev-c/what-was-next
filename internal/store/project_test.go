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

func TestDeleteProjectCascades(t *testing.T) {
	s := newTestStore(t)
	p, _ := s.CreateProject("Doomed")
	// Insert a task and a time entry directly (task CRUD arrives in a later task).
	res, err := s.db.Exec(
		`INSERT INTO tasks (project_id, title, notes, done, sort_order, created_at)
		 VALUES (?, 'child', '', 0, 1, ?)`, p.ID, s.now())
	if err != nil {
		t.Fatalf("insert task: %v", err)
	}
	taskID, _ := res.LastInsertId()
	if _, err := s.db.Exec(
		`INSERT INTO time_entries (task_id, started_at, ended_at) VALUES (?, ?, NULL)`,
		taskID, s.now()); err != nil {
		t.Fatalf("insert time entry: %v", err)
	}

	if err := s.DeleteProject(p.ID); err != nil {
		t.Fatalf("DeleteProject: %v", err)
	}

	var tasks, entries int
	s.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE project_id = ?`, p.ID).Scan(&tasks)
	s.db.QueryRow(`SELECT COUNT(*) FROM time_entries WHERE task_id = ?`, taskID).Scan(&entries)
	if tasks != 0 {
		t.Errorf("want 0 tasks after cascade delete, got %d", tasks)
	}
	if entries != 0 {
		t.Errorf("want 0 time entries after cascade delete, got %d", entries)
	}
}
