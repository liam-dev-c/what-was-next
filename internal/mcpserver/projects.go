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

// TEMP stub — replaced by tasks.go in Task 3.
func addTaskTools(srv *mcp.Server, s *store.Store) {}
