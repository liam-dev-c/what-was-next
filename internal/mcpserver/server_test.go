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
