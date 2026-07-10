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
