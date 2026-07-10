package mcpserver

import (
	"reflect"
	"testing"
)

func TestInstallArgs(t *testing.T) {
	got := installArgs("/usr/local/bin/what-was-next", "user")
	want := []string{
		"mcp", "add", "--scope", "user",
		"what-was-next", "--", "/usr/local/bin/what-was-next", "mcp",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("installArgs = %v, want %v", got, want)
	}
}

func TestInstallRejectsBadScope(t *testing.T) {
	if err := Install("bogus"); err == nil {
		t.Fatal("Install(\"bogus\"): expected error, got nil")
	}
}
