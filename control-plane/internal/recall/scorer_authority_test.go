package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// TestScoreAndSortWithReason_authorityDominates proves a lower-authority row with newer UpdatedAt
// does not sort above a higher-authority row when both are the same kind (RC1 authority dominance).
func TestScoreAndSortWithReason_authorityDominates(t *testing.T) {
	old := time.Date(2019, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	idWeak := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	idStrong := uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
	objs := []memory.MemoryObject{
		{ID: idWeak, Kind: api.MemoryKindDecision,
			Statement: "weak newer", Authority: 3, UpdatedAt: newer, Status: api.StatusActive},
		{ID: idStrong, Kind: api.MemoryKindDecision,
			Statement: "strong older", Authority: 9, UpdatedAt: old, Status: api.StatusActive},
	}
	w := DefaultRankingWeights()
	out := ScoreAndSortWithReason(objs, ScoreRequest{Tags: []string{}}, w, 0)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Object.ID != idStrong {
		t.Fatalf("want higher authority first, got %v before %v", out[0].Object.Authority, out[1].Object.Authority)
	}
}
