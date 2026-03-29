package merge

import (
	"strings"
	"testing"

	"control-plane/internal/runmulti"
)

func TestExtractSegments_paragraphs(t *testing.T) {
	runs := []runmulti.RunResult{{
		Variant: "a",
		Output:  "First paragraph here with enough chars.\n\nSecond paragraph also long enough.",
	}}
	segs := ExtractSegments(runs)
	if len(segs) != 2 {
		t.Fatalf("want 2 segments, got %d", len(segs))
	}
	if !strings.Contains(segs[0].Text, "First paragraph") || !strings.Contains(segs[1].Text, "Second paragraph") {
		t.Errorf("unexpected segments: %#v", segs)
	}
}

func TestExtractSegments_bullets(t *testing.T) {
	runs := []runmulti.RunResult{{
		Variant: "b",
		Output:  "- First bullet line with enough text here\n- Second bullet line with enough text here",
	}}
	segs := ExtractSegments(runs)
	if len(segs) != 2 {
		t.Fatalf("want 2 bullet segments, got %d: %#v", len(segs), segs)
	}
}

func TestExtractSegments_skipsShort(t *testing.T) {
	runs := []runmulti.RunResult{{
		Variant: "c",
		Output:  "short",
	}}
	if len(ExtractSegments(runs)) != 0 {
		t.Fatal("expected no segments")
	}
}

func TestExtractSegments_order(t *testing.T) {
	runs := []runmulti.RunResult{
		{Variant: "v1", Output: "Alpha paragraph with sufficient length here."},
		{Variant: "v2", Output: "Beta paragraph with sufficient length here."},
	}
	segs := ExtractSegments(runs)
	if len(segs) != 2 || segs[0].Variant != "v1" || segs[1].Variant != "v2" {
		t.Fatalf("order wrong: %#v", segs)
	}
}
