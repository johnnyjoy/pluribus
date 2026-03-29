package recall

import (
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

func TestApplicabilityComponent_emptyRequestTags_noPenalty(t *testing.T) {
	obj := memory.MemoryObject{
		Tags:           []string{"recall", "cursor-verify"},
		Applicability:  api.ApplicabilityGoverning,
	}
	w := DefaultRIUWeights()
	got := applicabilityComponent(obj, nil, w)
	// tagPart=1, enumPart=1 → blend=1 → Applicability weight * 1
	want := w.Applicability * 1.0
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("applicabilityComponent(empty tags) = %v, want %v (full blend)", got, want)
	}
}

func TestApplicabilityComponent_withRequestTags_usesOverlap(t *testing.T) {
	obj := memory.MemoryObject{
		Tags:          []string{"recall"},
		Applicability: api.ApplicabilityGoverning,
	}
	w := DefaultRIUWeights()
	got := applicabilityComponent(obj, []string{"recall"}, w)
	want := w.Applicability * (0.55*1.0 + 0.45*1.0)
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("applicabilityComponent(tags=recall) = %v, want %v", got, want)
	}
}
