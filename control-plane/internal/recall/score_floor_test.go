package recall

import (
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestFillGroupedViews_appliesScoreFloorNoPad(t *testing.T) {
	b := &RecallBundle{}
	scored := []ScoredMemory{
		{Object: memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "on-topic"}, Score: 3.5, Components: &ScoreComponentBreakdown{RelevanceScore: 0.8}},
		{Object: memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "off-topic loot"}, Score: 0.04, Components: &ScoreComponentBreakdown{RelevanceScore: 0}},
		{Object: memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindState, Statement: "other junk"}, Score: 0.04, Components: &ScoreComponentBreakdown{RelevanceScore: 0}},
	}
	fillGroupedViews(b, scored, nil, true, 5, "continuity", "", "pluribus controlplane pg_dump", memoryItemEnrichment{})
	if len(b.Constraints) != 1 || b.Constraints[0].Statement != "on-topic" {
		t.Fatalf("constraints %+v", b.Constraints)
	}
	if len(b.Continuity) != 0 {
		t.Fatalf("continuity must not pad with off-topic: %+v", b.Continuity)
	}
}

func TestScoreFloorFromTop_matchesBenchmark(t *testing.T) {
	if ScoreFloorFromTop(0.05) != 0 {
		t.Fatal("weak top has no floor")
	}
	if got := ScoreFloorFromTop(1.0); got < 0.27 || got > 0.29 {
		t.Fatalf("floor for top=1.0 got %v", got)
	}
}

func TestFilterScoredByFloor_usesRelevanceNotAuthority(t *testing.T) {
	on := ScoredMemory{Object: memory.MemoryObject{Statement: "on-topic"}, Score: 0.4, Components: &ScoreComponentBreakdown{RelevanceScore: 0.8}}
	off := ScoredMemory{Object: memory.MemoryObject{Statement: "lootz"}, Score: 2.0, Components: &ScoreComponentBreakdown{RelevanceScore: 0}}
	got := filterScoredByFloor([]ScoredMemory{off, on}, "pluribus recall")
	if len(got) != 1 || got[0].Object.Statement != "on-topic" {
		t.Fatalf("got %+v", got)
	}
	kept := filterScoredByFloor([]ScoredMemory{off, on}, "")
	if len(kept) != 2 {
		t.Fatalf("empty query must not drop rows, got %d", len(kept))
	}
}
