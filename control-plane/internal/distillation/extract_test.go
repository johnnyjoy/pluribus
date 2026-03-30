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
