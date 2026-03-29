package ingest

import (
	"context"
	"testing"

	"control-plane/internal/memory"

	"github.com/google/uuid"
)

func TestPromoteTypeFromPredicate(t *testing.T) {
	if promoteTypeFromPredicate("must not access") != "constraint" {
		t.Fatal()
	}
	if promoteTypeFromPredicate("failed to compile") != "failure" {
		t.Fatal()
	}
	if promoteTypeFromPredicate("decide architecture") != "decision" {
		t.Fatal()
	}
	if promoteTypeFromPredicate("depends_on") != "pattern" {
		t.Fatal()
	}
}

func TestClampPromoteConfidence(t *testing.T) {
	if clampPromoteConfidence(-1) != 0 || clampPromoteConfidence(2) != 1 || clampPromoteConfidence(0.5) != 0.5 {
		t.Fatal()
	}
}

type recordingPromoter struct {
	reqs []memory.PromoteRequest
	err  error
}

func (r *recordingPromoter) Promote(ctx context.Context, req memory.PromoteRequest) (*memory.PromoteResponse, error) {
	r.reqs = append(r.reqs, req)
	if r.err != nil {
		return nil, r.err
	}
	return &memory.PromoteResponse{Promoted: true, ID: uuid.New().String()}, nil
}

func TestApplyPromotions_gated(t *testing.T) {
	ing := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	row := CanonicalFactRow{
		IngestionID: ing,
		SubjectNorm: "a", PredicateNorm: "b", ObjectNorm: "c",
		Confidence: 0.8, NormalizedHash: "abc",
	}
	rec := &recordingPromoter{}

	dbg := applyPromotions(context.Background(), rec, false, true, []CanonicalFactRow{row}, ing)
	if dbg.Attempted || len(rec.reqs) != 0 {
		t.Fatalf("auto_promote off should not call: %+v", dbg)
	}

	dbg = applyPromotions(context.Background(), rec, true, false, []CanonicalFactRow{row}, ing)
	if dbg.Attempted || len(rec.reqs) != 0 {
		t.Fatalf("client must propose: %+v", dbg)
	}

	dbg = applyPromotions(context.Background(), nil, true, true, []CanonicalFactRow{row}, ing)
	if dbg.Attempted {
		t.Fatal()
	}

	dbg = applyPromotions(context.Background(), rec, true, true, []CanonicalFactRow{row}, ing)
	if !dbg.Attempted || len(rec.reqs) != 1 {
		t.Fatalf("expected 1 promote, got %+v reqs=%d", dbg, len(rec.reqs))
	}
	if rec.reqs[0].Type != "pattern" {
		t.Fatalf("req: %+v", rec.reqs[0])
	}
}

func TestApplyCommitPromotions_operator(t *testing.T) {
	ing := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	row := CanonicalFactRow{
		IngestionID: ing,
		SubjectNorm: "x", PredicateNorm: "y", ObjectNorm: "z",
		Confidence: 0.9, NormalizedHash: "xyz",
	}
	rec := &recordingPromoter{}

	dbg := applyCommitPromotions(context.Background(), rec, true, []CanonicalFactRow{row}, ing)
	if !dbg.Attempted || dbg.Mode != "commit_operator" || len(rec.reqs) != 1 {
		t.Fatalf("%+v", dbg)
	}
}
