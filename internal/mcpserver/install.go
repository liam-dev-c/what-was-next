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
