package enforcement

import (
	"testing"

	"control-plane/internal/memory"
	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestNormativeConflict_PostgresVsSQLite(t *testing.T) {
	m := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindConstraint,
		Authority: 9,
		Statement: "All durable project data must use Postgres; SQLite is not permitted.",
	}
	pc := memorynorm.StatementCanonical("We will migrate the service to use SQLite for simplicity.")
	ok, detail := normativeConflict(m, pc, normalizeIntent("datastore"))
	if !ok {
		t.Fatalf("expected conflict, detail=%q", detail)
	}
	if detail == "" {
		t.Fatal("expected detail")
	}
}

func TestNormativeConflict_AllowUnrelated(t *testing.T) {
	m := memory.MemoryObject{
		Kind:      api.MemoryKindDecision,
		Authority: 8,
		Statement: "All durable project data must use Postgres.",
	}
	pc := memorynorm.StatementCanonical("We will add a new metrics dashboard.")
	ok, _ := normativeConflict(m, pc, normalizeIntent(""))
	if ok {
		t.Fatal("expected no conflict")
	}
}

func TestNormativeConflict_paraphrasesSameDecision(t *testing.T) {
	m := memory.MemoryObject{
		Kind:               api.MemoryKindConstraint,
		Authority:          9,
		Statement:          "All durable project data MUST use Postgres; SQLite is not permitted.",
		StatementCanonical: memorynorm.StatementCanonical("All durable project data MUST use Postgres; SQLite is not permitted."),
	}
	proposals := []string{
		"  We will migrate the service to use SQLite for simplicity.  ",
		"We will migrate the service to use sqlite for simplicity",
	}
	var wantOK bool
	var wantDetail string
	for i, raw := range proposals {
		pc := memorynorm.StatementCanonical(raw)
		ok, detail := normativeConflict(m, pc, normalizeIntent("datastore"))
		if i == 0 {
			wantOK, wantDetail = ok, detail
			if !ok {
				t.Fatalf("first proposal should conflict, detail=%q", detail)
			}
			continue
		}
		if ok != wantOK || detail != wantDetail {
			t.Fatalf("paraphrase %d: ok=%v want %v detail=%q want %q", i, ok, wantOK, detail, wantDetail)
		}
	}
}

func TestEvaluateAll_paraphraseProposalsSameHitCount(t *testing.T) {
	m := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindConstraint,
		Authority: 9,
		Statement: "All durable project data must use Postgres; SQLite is not permitted.",
	}
	m.StatementCanonical = memorynorm.StatementCanonical(m.Statement)
	proposals := []string{
		"We will migrate durable storage to SQLite.",
		"  we  will  migrate  durable  storage  to  sqlite.  ",
	}
	var n0 int
	for i, p := range proposals {
		hits := evaluateAll([]memory.MemoryObject{m}, p, "datastore", 0.25, 4)
		if i == 0 {
			n0 = len(hits)
			if n0 == 0 {
				t.Fatal("expected at least one hit")
			}
			continue
		}
		if len(hits) != n0 {
			t.Fatalf("proposal %d: len(hits)=%d want %d", i, len(hits), n0)
		}
	}
}

func TestEvaluateAll_naturalLanguageConstraintNotHeuristicHit(t *testing.T) {
	m := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindConstraint,
		Authority: 9,
		Statement: "All primary interface actions must use teal as the only accent color.",
	}
	m.StatementCanonical = memorynorm.StatementCanonical(m.Statement)
	hits := evaluateAll([]memory.MemoryObject{m},
		"We will ship primary actions as blue buttons for brand refresh.", "", 0.25, 4)
	if len(hits) != 0 {
		t.Fatalf("expected no rule-based hit for unmodelled NL constraint, got %d hits (first reason=%q)",
			len(hits), hits[0].ReasonCode)
	}
}

func TestEvaluateAll_intentNormalized(t *testing.T) {
	m := memory.MemoryObject{
		ID:        uuid.New(),
		Kind:      api.MemoryKindConstraint,
		Authority: 9,
		Statement: "All durable project data must use Postgres only.",
	}
	m.StatementCanonical = memorynorm.StatementCanonical(m.Statement)
	h1 := evaluateAll([]memory.MemoryObject{m}, "We will use SQLite.", "datastore", 0.25, 4)
	h2 := evaluateAll([]memory.MemoryObject{m}, "We will use SQLite.", "  Datastore  ", 0.25, 4)
	if len(h1) != len(h2) {
		t.Fatalf("len(h1)=%d len(h2)=%d", len(h1), len(h2))
	}
}

func TestWordOverlapRatio_paraphraseSame(t *testing.T) {
	m := memory.MemoryObject{
		Statement:          "The fluent builder API caused confusion in production",
		StatementCanonical: memorynorm.StatementCanonical("The fluent builder API caused confusion in production"),
	}
	memCanon := memoryStatementCanonical(m)
	p1 := memorynorm.StatementCanonical("We should refactor the fluent API for clarity")
	p2 := memorynorm.StatementCanonical("  we  SHOULD   refactor   the   fluent   API   for   clarity  ")
	r1 := wordOverlapRatio(wordSet(p1), wordSet(memCanon))
	r2 := wordOverlapRatio(wordSet(p2), wordSet(memCanon))
	if r1 != r2 {
		t.Fatalf("r1=%v r2=%v", r1, r2)
	}
}

func TestWordOverlapRatio_failure(t *testing.T) {
	p := "We should refactor the fluent API for clarity"
	s := "The fluent builder API caused confusion in production"
	r := wordOverlapRatio(wordSet(p), wordSet(s))
	if r < 0.2 {
		t.Fatalf("expected meaningful overlap, got %v", r)
	}
}

func TestWorstDecision(t *testing.T) {
	w := worstDecision([]EnforcementDecision{
		DecisionRequireReview,
		DecisionBlock,
		DecisionAllow,
	})
	if w != DecisionBlock {
		t.Fatalf("got %q", w)
	}
}
