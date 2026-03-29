package recall

import (
	"encoding/json"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// Dominance sprint: same-authority ordering must favor binding-relevant signals (pattern severity,
// salience, symbols) over weak lexical noise.

func TestDominance_strongPatternBeatsWeakVariantSameAuthority(t *testing.T) {
	now := time.Now()
	req := ScoreRequest{SituationQuery: "payment webhook retry timeout billing"}
	w := DefaultRankingWeights()
	w.PatternPriority = 0.2

	weak := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 8,
		Statement: "generic timeout handling for any service",
		UpdatedAt: now,
		Payload:   []byte(`{"polarity":"positive","impact":{"severity":"low"}}`),
	}
	strong := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 8,
		Statement: "payment webhook retry with idempotency keys for billing",
		UpdatedAt: now,
		Payload:   []byte(`{"polarity":"negative","impact":{"severity":"high"},"symbols":["billing.Webhook"]}`),
	}

	out := ScoreAndSortWithReason([]memory.MemoryObject{weak, strong}, req, w, 0)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].Object.ID != strong.ID {
		t.Fatalf("expected high-severity pattern first, got %v before %v", out[0].Object.Statement, out[1].Object.Statement)
	}
}

func TestDominance_crossAgentSalienceWinsAuthorityTie(t *testing.T) {
	now := time.Now()
	req := ScoreRequest{}
	w := DefaultRankingWeights()
	w.CrossAgentSalience = 0.12

	localOnly := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 8,
		Statement: "Prefer local cache for session tokens",
		UpdatedAt: now,
	}
	shared := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindDecision, Authority: 8,
		Statement: "Prefer shared redis for session tokens",
		UpdatedAt: now,
		Payload:   []byte(`{"salience":{"distinct_agents":4}}`),
	}

	out := ScoreAndSortWithReason([]memory.MemoryObject{localOnly, shared}, req, w, 0)
	if out[0].Object.ID != shared.ID {
		t.Fatalf("expected cross-agent memory first, got %q before %q", out[0].Object.Statement, out[1].Object.Statement)
	}
}

func TestDominance_crossContextSalienceWinsAuthorityTie(t *testing.T) {
	now := time.Now()
	req := ScoreRequest{}
	w := DefaultRankingWeights()
	w.CrossContextSalience = 0.12

	narrow := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 7,
		Statement: "Single-tenant deploy checklist",
		UpdatedAt: now,
	}
	broad := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 7,
		Statement: "Multi-region rollout checklist",
		UpdatedAt: now,
		Payload:   []byte(`{"salience":{"distinct_contexts":5}}`),
	}

	out := ScoreAndSortWithReason([]memory.MemoryObject{narrow, broad}, req, w, 0)
	if out[0].Object.ID != broad.ID {
		t.Fatalf("expected cross-context pattern first")
	}
}

func TestDominance_failureSeverityBeatsPlainFailureSameAuthority(t *testing.T) {
	now := time.Now()
	req := ScoreRequest{Tags: []string{"prod"}, SituationQuery: "deploy rollback production"}
	w := DefaultRankingWeights()
	w.FailureSeverity = 0.4

	plain := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 8,
		Statement: "flaky test sometimes failed",
		UpdatedAt:   now,
		Tags:        []string{"prod"},
	}
	severe := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindFailure, Authority: 8,
		Statement: "production outage after bad migration",
		UpdatedAt:   now,
		Tags:        []string{"prod"},
	}

	out := ScoreAndSortWithReason([]memory.MemoryObject{plain, severe}, req, w, 0)
	if out[0].Object.ID != severe.ID {
		t.Fatalf("expected severity-weighted failure first")
	}
}

func TestDominance_symbolOverlapBeatsPureLexicalTie(t *testing.T) {
	now := time.Now()
	req := ScoreRequest{
		SituationQuery: "billing webhook handler timeout",
		Symbols:        []string{"pkg/billing.Handler"},
	}
	w := DefaultRankingWeights()

	noise := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 8,
		Statement: "billing webhook handler timeout without symbols payload",
		UpdatedAt:   now,
		Payload:     []byte(`{}`),
	}
	targeted := memory.MemoryObject{
		ID: uuid.New(), Kind: api.MemoryKindPattern, Authority: 8,
		Statement: "handler stability",
		UpdatedAt:   now,
	}
	var p memory.PatternPayload
	p.Symbols = []string{"pkg/billing.Handler"}
	p.Polarity = "positive"
	p.Impact.Severity = "low"
	pl, _ := json.Marshal(p)
	targeted.Payload = pl

	out := ScoreAndSortWithReason([]memory.MemoryObject{noise, targeted}, req, w, 0)
	if out[0].Object.ID != targeted.ID {
		t.Fatalf("expected symbol-overlap pattern first: got %q", out[0].Object.Statement)
	}
}
