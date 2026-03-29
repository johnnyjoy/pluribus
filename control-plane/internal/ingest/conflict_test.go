package ingest

import (
	"testing"

	"github.com/google/uuid"
)

func TestDetectConflictsAmongRows_samePredicateDivergentObject(t *testing.T) {
	t.Parallel()
	ing := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	rows := []CanonicalFactRow{
		{IngestionID: ing, SubjectNorm: "app", PredicateNorm: "uses", ObjectNorm: "postgres", NormalizedHash: "a", SourceIndex: 0},
		{IngestionID: ing, SubjectNorm: "app", PredicateNorm: "uses", ObjectNorm: "mysql", NormalizedHash: "b", SourceIndex: 1},
	}
	cs := DetectConflictsAmongRows(rows, 0.35)
	if len(cs) != 1 {
		t.Fatalf("conflicts: %v", cs)
	}
	if cs[0]["code"] != "same_predicate_divergent_object" {
		t.Fatalf("code: %v", cs[0]["code"])
	}
}

func TestDetectConflictsAmongRows_oppositePredicate(t *testing.T) {
	t.Parallel()
	ing := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	rows := []CanonicalFactRow{
		{IngestionID: ing, SubjectNorm: "api", PredicateNorm: "is", ObjectNorm: "ready", NormalizedHash: "a", SourceIndex: 0},
		{IngestionID: ing, SubjectNorm: "api", PredicateNorm: "is_not", ObjectNorm: "ready", NormalizedHash: "b", SourceIndex: 1},
	}
	cs := DetectConflictsAmongRows(rows, 0.35)
	if len(cs) != 1 || cs[0]["code"] != "opposite_predicate" {
		t.Fatalf("got %v", cs)
	}
}
