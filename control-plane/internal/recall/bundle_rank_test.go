package recall

import (
	"testing"
)

func TestFlattenBundleByScore_dedupesAndSorts(t *testing.T) {
	b := &RecallBundle{
		GoverningConstraints: []MemoryItem{{ID: "a", Justification: &JustificationMeta{Score: 2}}},
		ApplicablePatterns:   []MemoryItem{{ID: "a", Justification: &JustificationMeta{Score: 1}}, {ID: "b", Justification: &JustificationMeta{Score: 3}}},
	}
	flat := FlattenBundleByScore(b)
	if len(flat) != 2 {
		t.Fatalf("len=%d want 2", len(flat))
	}
	if flat[0].ID != "b" || flat[1].ID != "a" {
		t.Fatalf("order wrong: %+v", flat)
	}
}
