package runmulti

import (
	"strings"
	"testing"
)

func TestBuildContext(t *testing.T) {
	b := &RecallBundleMirror{
		GoverningConstraints: []MemoryItemMirror{{Statement: "Do not skip tests"}},
		Decisions:            []MemoryItemMirror{{Statement: "Use feature flags"}},
	}
	out := BuildContext(b, "balanced", "Implement the handler.")
	if !strings.Contains(out, "Do not skip tests") || !strings.Contains(out, "feature flags") {
		t.Fatalf("unexpected context: %s", out)
	}
	if !strings.Contains(out, "[VARIANT]") || !strings.Contains(out, "balanced") {
		t.Fatalf("missing variant: %s", out)
	}
}

func TestBuildContext_nil_bundle(t *testing.T) {
	out := BuildContext(nil, "", "Hello")
	if !strings.Contains(out, "Hello") {
		t.Fatalf("BuildContext(nil, '', 'Hello') = %q", out)
	}
}
