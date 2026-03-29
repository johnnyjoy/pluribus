package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestScore_semanticSimilarityTerm(t *testing.T) {
	id := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	req := ScoreRequest{
		SemanticSimilarity: map[uuid.UUID]float64{id: 0.9},
	}
	w := DefaultRankingWeights()
	w.SemanticSimilarity = 0.2
	now := time.Now()
	obj := memory.MemoryObject{
		ID:        id,
		Kind:      api.MemoryKindPattern,
		Authority: 3,
		UpdatedAt: now,
	}
	s := scoreBase(obj, req, w, 10, now)
	wOff := DefaultRankingWeights()
	wOff.SemanticSimilarity = 0
	s0 := scoreBase(obj, req, wOff, 10, now)
	if s <= s0 {
		t.Fatalf("semantic term should increase score: %v vs %v", s, s0)
	}
}

func TestSemantic_contributionCappedAtAuthorityTerm(t *testing.T) {
	w := DefaultRankingWeights()
	w.SemanticSimilarity = 2.0 // would exceed raw authority contribution without cap
	authNorm := 0.5
	authContrib := w.Authority * authNorm
	raw := w.SemanticSimilarity * 1.0
	if raw <= authContrib {
		t.Fatalf("test setup: raw semantic should exceed auth contrib for cap to matter")
	}
	got := semanticScoreTerm(w, authNorm, 1.0)
	if got != authContrib {
		t.Fatalf("semanticScoreTerm = %v, want capped %v", got, authContrib)
	}
}

func TestSemantic_doesNotOutrankHighAuthority(t *testing.T) {
	lowAuth := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	highAuth := uuid.MustParse("b0000000-0000-0000-0000-000000000002")
	req := ScoreRequest{
		SemanticSimilarity: map[uuid.UUID]float64{
			lowAuth:  0.99,
			highAuth: 0.1,
		},
	}
	w := DefaultRankingWeights()
	w.SemanticSimilarity = 0.15
	w.Authority = 1.0
	now := time.Now()
	low := memory.MemoryObject{ID: lowAuth, Kind: api.MemoryKindDecision, Authority: 2, UpdatedAt: now}
	high := memory.MemoryObject{ID: highAuth, Kind: api.MemoryKindDecision, Authority: 9, UpdatedAt: now}
	// Stable sort: authority first — high authority row should sort before low even if semantic loves the low row.
	sm := []ScoredMemory{
		{Object: low, Score: scoreBase(low, req, w, 10, now), Reason: "x"},
		{Object: high, Score: scoreBase(high, req, w, 10, now), Reason: "y"},
	}
	sortScoredMemoriesStable(sm)
	if sm[0].Object.ID != highAuth {
		t.Fatalf("expected higher-authority memory first: got %v before %v", sm[0].Object.ID, sm[1].Object.ID)
	}
}
