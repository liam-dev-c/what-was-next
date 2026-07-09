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
