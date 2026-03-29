package ingest

import (
	"testing"

	"github.com/google/uuid"
)

func TestUnifySimilarWithinBatch(t *testing.T) {
	t.Parallel()
	ing := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"t"},
		ExtractedFacts: []ExtractedFact{
			{Subject: "app", Predicate: "depends_on", Object: "alpha beta gamma"},
			{Subject: "app", Predicate: "depends_on", Object: "alpha beta gamma delta"},
		},
		Confidence:        0.5,
		ContextWindowHash: "h",
	}
	rows, _ := BuildCanonicalRows(ing, req)
	if len(rows) != 2 {
		t.Fatalf("rows %d", len(rows))
	}
	var merge []map[string]interface{}
	UnifySimilarWithinBatch(rows, DefaultSimilarJaccardMin, &merge, nil)
	if rows[0].ObjectNorm == rows[1].ObjectNorm && rows[0].ObjectNorm == "alpha beta gamma" {
		// second merged into first wording
	} else {
		t.Fatalf("after unify: %#v %#v", rows[0].ObjectNorm, rows[1].ObjectNorm)
	}
	if rows[0].NormalizedHash != rows[1].NormalizedHash {
		t.Fatal("hashes should match after unify")
	}
	if len(merge) != 1 || merge[0]["action"] != "similar_unify" {
		t.Fatalf("merge %v", merge)
	}
}
