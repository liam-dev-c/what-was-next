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
