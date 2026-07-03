package mcp

import (
	"slices"
	"testing"
)

func TestTagsFromRepoRoot(t *testing.T) {
	got := TagsFromRepoRoot("/projects/pluribus")
	want := []string{"project:pluribus", "repo:pluribus"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
	if TagsFromRepoRoot("") != nil {
		t.Fatal("empty path should yield nil")
	}
}

func TestAppendRepoRootTags_dedupes(t *testing.T) {
	got := appendRepoRootTags([]string{"project:pluribus", "deploy"}, "/projects/pluribus")
	if len(got) != 3 {
		t.Fatalf("got %v", got)
	}
}
