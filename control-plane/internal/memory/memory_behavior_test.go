package memory

import (
	"testing"

	"control-plane/pkg/api"
)

func TestMergePersistTags_onlyExplicitTags(t *testing.T) {
	got := mergePersistTags(CreateRequest{
		Kind:      api.MemoryKindDecision,
		Statement: "x",
		Tags:      nil,
	})
	if len(got) != 0 {
		t.Fatalf("expected no auto tags, got %v", got)
	}
	got2 := mergePersistTags(CreateRequest{
		Tags: []string{"  a ", "a", "b"},
	})
	if len(got2) != 2 || got2[0] != "a" || got2[1] != "b" {
		t.Fatalf("got %v", got2)
	}
}

func TestDedupKey_isSharedConstant(t *testing.T) {
	if DedupKey() != "shared" {
		t.Fatalf("unexpected dedup key: %q", DedupKey())
	}
}
