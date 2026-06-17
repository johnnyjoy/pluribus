package recall

import (
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

func TestGenericAgentNoisePenalty(t *testing.T) {
	obj := memory.MemoryObject{
		ID:        [16]byte{1},
		Kind:      api.MemoryKindPattern,
		Authority: 4,
		Statement: "Agent workflows should recall context before edits and record experience afterward in multi-step tasks.",
		Tags:      []string{"agent", "workflow", "loop", "recall"},
		UpdatedAt: time.Now(),
	}
	req := ScoreRequest{SituationQuery: "agent interface integration binding recall"}
	out := ScoreAndSortWithReason([]memory.MemoryObject{obj}, req, DefaultRankingWeights(), 0)
	if out[0].Components.GenericTermPenalty < 0.5 {
		t.Fatalf("expected technical generic penalty, got gtp=%v score=%v", out[0].Components.GenericTermPenalty, out[0].Score)
	}
	if out[0].Score > 0.1 {
		t.Fatalf("expected low score after penalty, got %v", out[0].Score)
	}
}
