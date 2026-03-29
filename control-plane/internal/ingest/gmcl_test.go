package ingest

import (
	"testing"
)

// GMCL directive tests (T1–T5): behavior is covered by service_test + this package.
// T1/T2 (cross-batch duplicate / variation): reinforce + global unify paths — see TestService_IngestCognition_reinforcesWhenHashExistsInDB
// and similar-unify tests; full two-transaction sqlmock would duplicate harness noise.
// T3: TestService_IngestCognition_conflictReportedWithoutReject + expectContradictionInsert.
// T4: determinism — same normalization + ordered global unify + hash tie-breaks; see minLexHash / ORDER BY in repo.
// T5: multi-hop — UnifySimilarWithinBatch + GlobalUnifyFromDB chain via merge_actions + lineage meta.

func TestGMCL_MinLexHash(t *testing.T) {
	t.Parallel()
	if minLexHash("b", "a") != "a" {
		t.Fatal(minLexHash("b", "a"))
	}
	if minLexHash("a", "a") != "a" {
		t.Fatal()
	}
}

func TestGMCL_OppositePredicateNorm(t *testing.T) {
	t.Parallel()
	o, ok := OppositePredicateNorm("supports")
	if !ok || o != "does_not_support" {
		t.Fatalf("got %q %v", o, ok)
	}
	_, ok = OppositePredicateNorm("uses")
	if ok {
		t.Fatal("uses should have no registered opposite")
	}
}

func TestGMCL_CollectSimilarCandidates_skipsSameHash(t *testing.T) {
	t.Parallel()
	a := CanonicalFactRow{
		SubjectNorm: "s", PredicateNorm: "p", ObjectNorm: "o",
		NormalizedHash: "hash1",
	}
	b := CanonicalFactRow{
		SubjectNorm: "s", PredicateNorm: "p", ObjectNorm: "o",
		NormalizedHash: "hash1",
	}
	row := &CanonicalFactRow{
		SubjectNorm: "s", PredicateNorm: "p", ObjectNorm: "o",
		NormalizedHash: "hash1",
	}
	rows := []CanonicalFactRow{a, b}
	db := []canonicalTriple{{PredicateNorm: "p", ObjectNorm: "o", NormalizedHash: "hash1"}}
	got := collectSimilarCandidates(row, 1, rows, db, DefaultSimilarJaccardMin)
	if len(got) != 0 {
		t.Fatalf("same hash should be skipped: %#v", got)
	}
}
