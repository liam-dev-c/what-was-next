package main

import "testing"

func TestCommand(t *testing.T) {
	cases := []struct {
		args []string
		want cmd
	}{
		{nil, cmdTUI},
		{[]string{}, cmdTUI},
		{[]string{"mcp"}, cmdMCPServe},
		{[]string{"mcp", "install"}, cmdMCPInstall},
		{[]string{"mcp", "install", "--scope", "project"}, cmdMCPInstall},
		{[]string{"help"}, cmdHelp},
		{[]string{"-h"}, cmdHelp},
		{[]string{"--help"}, cmdHelp},
		{[]string{"nonsense"}, cmdTUI},
	}
	for _, c := range cases {
		if got := command(c.args); got != c.want {
			t.Errorf("command(%v) = %d, want %d", c.args, got, c.want)
		}
	}
}

func TestScopeFlag(t *testing.T) {
	if got := scopeFlag([]string{"mcp", "install"}); got != "user" {
		t.Errorf("default scope = %q, want \"user\"", got)
	}
	if got := scopeFlag([]string{"mcp", "install", "--scope", "project"}); got != "project" {
		t.Errorf("scope = %q, want \"project\"", got)
	}
}
