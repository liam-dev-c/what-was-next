package store

import (
	"path/filepath"
	"testing"
)

// TestDataVersionTracksExternalWrites verifies the contract the TUI relies on:
// data_version is unchanged by writes on the querying connection but changes
// when another connection commits. This is what lets the TUI poll for MCP-side
// edits without reloading on its own mutations.
func TestDataVersionTracksExternalWrites(t *testing.T) {
	path := filepath.Join(t.TempDir(), "dv.db")
	self, err := Open(path)
	if err != nil {
		t.Fatalf("Open self: %v", err)
	}
	defer self.Close()
	other, err := Open(path)
	if err != nil {
		t.Fatalf("Open other: %v", err)
	}
	defer other.Close()

	v0, err := self.DataVersion()
	if err != nil {
		t.Fatalf("DataVersion: %v", err)
	}

	// A write on our own connection must not move data_version.
	if _, err := self.CreateProject("local"); err != nil {
		t.Fatalf("CreateProject self: %v", err)
	}
	vSelf, err := self.DataVersion()
	if err != nil {
		t.Fatalf("DataVersion: %v", err)
	}
	if vSelf != v0 {
		t.Fatalf("data_version changed after own write: %d -> %d", v0, vSelf)
	}

	// A write from another connection must move it.
	if _, err := other.CreateProject("external"); err != nil {
		t.Fatalf("CreateProject other: %v", err)
	}
	vExt, err := self.DataVersion()
	if err != nil {
		t.Fatalf("DataVersion: %v", err)
	}
	if vExt == vSelf {
		t.Fatalf("data_version unchanged after external write: still %d", vExt)
	}
}
