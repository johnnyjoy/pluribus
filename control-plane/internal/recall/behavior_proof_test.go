package recall

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestProof_ContinuityFailurePatternAndValidation(t *testing.T) {
	lessonPayload := json.RawMessage(`{"polarity":"negative","experience":"skipping tests caused regressions","decision":"always run tests","outcome":"rollback","impact":{"severity":"high"},"directive":"never skip tests","symbols":["pkg.Foo"]}`)
	memories := []memory.MemoryObject{
		{ID: uuid.New(), Kind: api.MemoryKindState, Statement: "Implement migration runner", Authority: 8, UpdatedAt: time.Now()},
		{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "Use transactional postgres migrations", Authority: 9, UpdatedAt: time.Now()},
		{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "Skipping tests broke deploy", Authority: 9, UpdatedAt: time.Now()},
		{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "Use staged rollout with retries", Authority: 7, UpdatedAt: time.Now()},
		{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "Never skip tests", Authority: 8, UpdatedAt: time.Now(), Payload: lessonPayload},
	}
	c := &Compiler{
		Memory:  &behaviorMemoryStub{objs: memories},
		Ranking: &RankingWeights{Authority: 1, Recency: 0.5, TagMatch: 1, FailureOverlap: 0.5, SymbolOverlap: 0.5},
	}
	bundle, err := c.Compile(context.Background(), CompileRequest{
		MaxPerKind: 3,
		Mode:       "continuity",
		Symbols:    []string{"pkg.Foo"},
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	pretty, _ := json.MarshalIndent(map[string]any{
		"continuity":  bundle.Continuity,
		"constraints": bundle.Constraints,
		"experience":  bundle.Experience,
	}, "", "  ")
	t.Logf("recall output:\n%s", string(pretty))

	// Candidate that would repeat a known failure and contradict decision.
	badProposal := "Replace transactional postgres migrations with sqlite and skip tests"
	badValidation := validateBehavior(bundle, badProposal, nil)
	t.Logf("validation (bad proposal): %+v", badValidation)
	if badValidation.OK() {
		t.Fatalf("expected bad proposal to fail validation")
	}

	// Candidate that follows remembered pattern/decision/constraints.
	goodProposal := "Continue transactional postgres migrations, keep tests, and use staged rollout with retries"
	goodValidation := validateBehavior(bundle, goodProposal, nil)
	t.Logf("validation (good proposal): %+v", goodValidation)
	if !goodValidation.OK() {
		t.Fatalf("expected good proposal to pass validation")
	}
}
