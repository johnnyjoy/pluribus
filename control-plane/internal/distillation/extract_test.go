package distillation

import (
	"testing"

	"control-plane/pkg/api"
)

func TestExtractDrafts_constraintAndFailure(t *testing.T) {
	d := extractDrafts("we had an error and rollback; must not skip tests")
	if len(d) < 2 {
		t.Fatalf("want at least constraint+failure, got %+v", d)
	}
	var hasC, hasF bool
	for _, x := range d {
		if x.kind == api.MemoryKindConstraint {
			hasC = true
		}
		if x.kind == api.MemoryKindFailure {
			hasF = true
		}
	}
	if !hasC || !hasF {
		t.Fatalf("got %+v", d)
	}
}

func TestExtractDrafts_decision(t *testing.T) {
	d := extractDrafts("we decided to use postgres instead of sqlite")
	if len(d) != 1 || d[0].kind != api.MemoryKindDecision {
		t.Fatalf("got %+v", d)
	}
}

func TestExtractDrafts_pattern(t *testing.T) {
	d := extractDrafts("this worked well in production")
	if len(d) != 1 || d[0].kind != api.MemoryKindPattern {
		t.Fatalf("got %+v", d)
	}
}

func TestExtractDrafts_empty(t *testing.T) {
	d := extractDrafts("lunch was fine")
	if len(d) != 0 {
		t.Fatalf("got %+v", d)
	}
}

func TestKindHintFromSummary_priority(t *testing.T) {
	if got, ok := KindHintFromSummary("must not deploy on friday; we had an error"); !ok || got != api.MemoryKindConstraint {
		t.Fatalf("constraint wins: got %v ok=%v", got, ok)
	}
	if got, ok := KindHintFromSummary("rollback and outage in prod"); !ok || got != api.MemoryKindFailure {
		t.Fatalf("got %v ok=%v", got, ok)
	}
	if got, ok := KindHintFromSummary("we decided to use redis"); !ok || got != api.MemoryKindDecision {
		t.Fatalf("got %v ok=%v", got, ok)
	}
	if got, ok := KindHintFromSummary("lunch was fine"); ok {
		t.Fatalf("no signal: expected !ok, got kind %v", got)
	}
}

func TestQualifyForProbationaryMemory_eventTag(t *testing.T) {
	k, r, ok, weak := QualifyForProbationaryMemory("short", []string{"mcp:event:failure"})
	if !ok || weak || k != api.MemoryKindFailure || r == "" {
		t.Fatalf("want failure from event tag, got kind=%v reason=%q ok=%v weak=%v", k, r, ok, weak)
	}
}

func TestQualifyForProbationaryMemory_plausibleWeak(t *testing.T) {
	s := "During this sprint we measured latency and compared baseline against candidate builds; results were mixed but insightful takeaway for next iteration."
	k, r, ok, weak := QualifyForProbationaryMemory(s, nil)
	if !ok || !weak || k != api.MemoryKindPattern {
		t.Fatalf("want plausible weak pattern, got kind=%v ok=%v weak=%v reason=%q", k, ok, weak, r)
	}
}

func TestQualifyForProbationaryMemory_garbageRepeat(t *testing.T) {
	_, _, ok, _ := QualifyForProbationaryMemory("zzzzzzzzzzzzzz", nil)
	if ok {
		t.Fatal("expected garbage reject")
	}
}

func TestQualifyForProbationaryMemory_experimentEventWeak(t *testing.T) {
	k, r, ok, weak := QualifyForProbationaryMemory("enough text here for min length boundary testing bench", []string{"mcp:event:benchmark"})
	if !ok || !weak || k != api.MemoryKindPattern || r == "" {
		t.Fatalf("got kind=%v ok=%v weak=%v reason=%q", k, ok, weak, r)
	}
}
