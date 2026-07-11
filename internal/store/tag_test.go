package store

import (
	"errors"
	"reflect"
	"testing"
)

func tagsOf(t *testing.T, s *Store, pid, taskID int64) []string {
	t.Helper()
	tasks, err := s.ListTasks(pid)
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	for _, tk := range tasks {
		if tk.ID == taskID {
			return tk.Tags
		}
	}
	t.Fatalf("task %d not found", taskID)
	return nil
}

func TestSetTaskTagsRoundTrip(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Tagged")

	if err := s.SetTaskTags(tk.ID, []string{"urgent", "backend"}); err != nil {
		t.Fatalf("SetTaskTags: %v", err)
	}
	if got := tagsOf(t, s, pid, tk.ID); !reflect.DeepEqual(got, []string{"backend", "urgent"}) {
		t.Fatalf("tags = %v, want [backend urgent] (sorted)", got)
	}
}

func TestSetTaskTagsNormalizes(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Messy")

	// Whitespace, blanks, and case-duplicates collapse.
	if err := s.SetTaskTags(tk.ID, []string{"  API ", "", "api", "api"}); err != nil {
		t.Fatalf("SetTaskTags: %v", err)
	}
	got := tagsOf(t, s, pid, tk.ID)
	if len(got) != 1 || got[0] != "API" {
		t.Fatalf("tags = %v, want [API] (first spelling, deduped)", got)
	}
}

func TestSetTaskTagsReplacesAndClears(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Replace me")

	if err := s.SetTaskTags(tk.ID, []string{"a", "b"}); err != nil {
		t.Fatalf("SetTaskTags: %v", err)
	}
	if err := s.SetTaskTags(tk.ID, []string{"c"}); err != nil {
		t.Fatalf("SetTaskTags replace: %v", err)
	}
	if got := tagsOf(t, s, pid, tk.ID); !reflect.DeepEqual(got, []string{"c"}) {
		t.Fatalf("after replace tags = %v, want [c]", got)
	}
	// Empty slice clears all tags.
	if err := s.SetTaskTags(tk.ID, nil); err != nil {
		t.Fatalf("SetTaskTags clear: %v", err)
	}
	if got := tagsOf(t, s, pid, tk.ID); got != nil {
		t.Fatalf("after clear tags = %v, want nil", got)
	}
}

func TestSetTaskTagsNotFound(t *testing.T) {
	s := newTestStore(t)
	if err := s.SetTaskTags(99999, []string{"x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("SetTaskTags on missing id: got %v, want ErrNotFound", err)
	}
}

func TestListTagsInUseOnly(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	a, _ := s.CreateTask(pid, "A")
	b, _ := s.CreateTask(pid, "B")
	s.SetTaskTags(a.ID, []string{"shared", "only-a"})
	s.SetTaskTags(b.ID, []string{"shared"})

	tags, err := s.ListTags()
	if err != nil {
		t.Fatalf("ListTags: %v", err)
	}
	if !reflect.DeepEqual(tags, []string{"only-a", "shared"}) {
		t.Fatalf("ListTags = %v, want [only-a shared]", tags)
	}

	// Removing a tag from its last task drops it from ListTags.
	s.SetTaskTags(a.ID, []string{"shared"})
	tags, _ = s.ListTags()
	if !reflect.DeepEqual(tags, []string{"shared"}) {
		t.Fatalf("ListTags after removal = %v, want [shared]", tags)
	}
}

func TestTagsDeletedWithTask(t *testing.T) {
	s := newTestStore(t)
	pid := projectID(t, s)
	tk, _ := s.CreateTask(pid, "Doomed")
	s.SetTaskTags(tk.ID, []string{"gone"})
	if err := s.DeleteTask(tk.ID); err != nil {
		t.Fatalf("DeleteTask: %v", err)
	}
	// The join row should cascade away, leaving the tag unused.
	tags, _ := s.ListTags()
	if len(tags) != 0 {
		t.Fatalf("ListTags after task delete = %v, want empty", tags)
	}
}
