package recall

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type behaviorMemoryStub struct {
	objs []memory.MemoryObject
}

func (s *behaviorMemoryStub) Search(_ context.Context, _ memory.SearchRequest) ([]memory.MemoryObject, error) {
	return append([]memory.MemoryObject(nil), s.objs...), nil
}

func (s *behaviorMemoryStub) SearchMemories(_ context.Context, _ memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	return append([]memory.MemoryObject(nil), s.objs...), nil
}

func TestValidateBehavior_flagsConstraintFailureAndConflict(t *testing.T) {
	b := &RecallBundle{
		Continuity: []MemoryItem{
			{Kind: "state", Statement: "ship migration with tests"},
			{Kind: "decision", Statement: "Use postgres migrations and keep tests green"},
		},
		Constraints: []MemoryItem{
			{Kind: "constraint", Statement: "never skip tests"},
			{Kind: "failure", Statement: "deleting migration files broke deploy"},
		},
	}
	v := validateBehavior(b, "Replace use postgres migrations and keep tests green; skip tests by deleting migration files instead", nil)
	if v.OK() {
		t.Fatalf("expected behavior validation to fail")
	}
	if len(v.ConstraintViolations) == 0 || len(v.RepeatedFailures) == 0 || len(v.DecisionConflicts) == 0 {
		t.Fatalf("expected all validation buckets populated: %+v", v)
	}
	if len(v.Actions) < 3 {
		t.Fatalf("expected actions for each hit, got %+v", v.Actions)
	}
}

func TestValidateBehavior_failureDefaultReject_softFailureReviseRestoresRevise(t *testing.T) {
	b := &RecallBundle{
		Constraints: []MemoryItem{
			{Kind: "failure", Statement: "deleting migration files broke deploy"},
		},
	}
	prop := "Deleting migration files broke our staging pipeline similar to the known failure"
	v := validateBehavior(b, prop, nil)
	if v.OK() {
		t.Fatal("expected overlap with failure")
	}
	var sawReject bool
	for _, a := range v.Actions {
		if a.Category == "repeated_failure" && a.Action == "reject" {
			sawReject = true
		}
	}
	if !sawReject {
		t.Fatalf("expected default reject on failure overlap, actions=%+v", v.Actions)
	}

	v2 := validateBehavior(b, prop, &BehaviorValidationConfig{SoftFailureRevise: true})
	var sawRevise bool
	for _, a := range v2.Actions {
		if a.Category == "repeated_failure" && a.Action == "revise" {
			sawRevise = true
		}
	}
	if !sawRevise {
		t.Fatalf("expected revise when soft_failure_revise, actions=%+v", v2.Actions)
	}
}

func TestValidateBehavior_postgresDecisionVsSQLiteProposal(t *testing.T) {
	b := &RecallBundle{
		Continuity: []MemoryItem{
			{Kind: "decision", Statement: "Postgres is the only durable store required for this service"},
		},
		Constraints: []MemoryItem{},
	}
	v := validateBehavior(b, "use sqlite instead of postgres for durable store data", nil)
	if v.OK() {
		t.Fatal("expected decision conflict")
	}
	if len(v.DecisionConflicts) == 0 {
		t.Fatalf("expected decision conflict: %+v", v)
	}
}

func TestCompile_continuityIncludesStateInUnrankedMode(t *testing.T) {
	st := memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindState, Statement: "Implement retry logic", Authority: 6, UpdatedAt: time.Now()}
	dec := memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindDecision, Statement: "Use exponential backoff", Authority: 7, UpdatedAt: time.Now()}
	fail := memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindFailure, Statement: "fixed interval caused thundering herd", Authority: 8, UpdatedAt: time.Now()}
	pat := memory.MemoryObject{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "jitter improves resilience", Authority: 7, UpdatedAt: time.Now()}

	c := &Compiler{
		Memory:  &behaviorMemoryStub{objs: []memory.MemoryObject{st, dec, fail, pat}},
		Ranking: nil, // force raw/fallback grouping path
	}
	b, err := c.Compile(context.Background(), CompileRequest{MaxPerKind: 3})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	var hasState, hasDecision bool
	for _, it := range b.Continuity {
		if it.Kind == string(api.MemoryKindState) {
			hasState = true
		}
		if it.Kind == string(api.MemoryKindDecision) {
			hasDecision = true
		}
	}
	if !hasState || !hasDecision {
		t.Fatalf("continuity must include state and decision, got: %+v", b.Continuity)
	}
	if len(b.Constraints) == 0 || b.Constraints[0].Kind != string(api.MemoryKindFailure) {
		t.Fatalf("constraints must include failures, got: %+v", b.Constraints)
	}
	if len(b.Experience) == 0 || b.Experience[0].Kind != string(api.MemoryKindPattern) {
		t.Fatalf("experience must include patterns, got: %+v", b.Experience)
	}
}
