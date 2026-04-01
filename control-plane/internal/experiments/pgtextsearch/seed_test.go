package pgtextsearch

import "testing"

func TestBuildSeedRows_MinCount(t *testing.T) {
	rows := BuildSeedRows()
	if len(rows) < 50 {
		t.Fatalf("expected at least 50 seed rows, got %d", len(rows))
	}
}

func TestBuildSeedRows_HasEvalTag(t *testing.T) {
	rows := BuildSeedRows()
	for i, r := range rows {
		has := false
		for _, x := range r.Tags {
			if x == EvalTag {
				has = true
				break
			}
		}
		if !has {
			t.Fatalf("row %d missing eval tag", i)
		}
	}
}
