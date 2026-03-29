package tooling

import (
	"path/filepath"
	"testing"
)

func TestAllowlist_CheckCommand_emptyAllowsAll(t *testing.T) {
	a := NewAllowlist(nil, nil)
	if !a.CheckCommand("git") {
		t.Error("empty allowlist should allow git")
	}
	if !a.CheckCommand("anything") {
		t.Error("empty allowlist should allow any command")
	}
}

func TestAllowlist_CheckCommand_restrictsWhenSet(t *testing.T) {
	a := NewAllowlist([]string{"git", "rg", "go"}, nil)
	if !a.CheckCommand("git") || !a.CheckCommand("go") || !a.CheckCommand("rg") {
		t.Error("allowed commands should pass")
	}
	if a.CheckCommand("curl") || a.CheckCommand("bash") {
		t.Error("disallowed commands should fail")
	}
	if !a.CheckCommand("GO") {
		t.Error("command check should be case-insensitive")
	}
}

func TestAllowlist_CheckDir_emptyAllowsAll(t *testing.T) {
	a := NewAllowlist(nil, nil)
	if !a.CheckDir("/any/path") {
		t.Error("empty allowlist should allow any dir")
	}
}

func TestAllowlist_CheckDir_restrictsWhenSet(t *testing.T) {
	allowed := t.TempDir()
	a := NewAllowlist(nil, []string{allowed})
	if !a.CheckDir(allowed) {
		t.Error("allowed dir should pass")
	}
	sub := filepath.Join(allowed, "sub")
	if !a.CheckDir(sub) {
		t.Error("subdir of allowed should pass")
	}
	if a.CheckDir("/tmp/other") {
		t.Error("dir outside allowed should fail")
	}
}

func TestAllowlist_CheckDir_rejectsOutside(t *testing.T) {
	allowed := t.TempDir()
	a := NewAllowlist(nil, []string{allowed})
	// Path that is a sibling (e.g. allowed is /tmp/xyz, request is /tmp/xyz_evil)
	if a.CheckDir(allowed+"_evil") {
		t.Error("sibling path should not be allowed")
	}
}
