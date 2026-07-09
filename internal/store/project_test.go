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
