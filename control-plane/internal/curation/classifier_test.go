package curation

import (
	"strings"
	"testing"

	"control-plane/pkg/api"
)

func TestClassify_structuredAnswers(t *testing.T) {
	req := DigestRequest{
		WorkSummary: "Enough summary text for validation.",
		CurationAnswers: &DigestCurationAnswers{
			Decision:   "Prefer feature flags",
			Constraint: "No direct prod writes",
		},
	}
	ds := Classify(req, 10)
	if len(ds) < 2 {
		t.Fatalf("expected at least 2 drafts, got %d", len(ds))
	}
	if ds[0].kind != api.MemoryKindDecision || ds[0].statement != "Prefer feature flags" {
		t.Errorf("first draft: %+v", ds[0])
	}
	if ds[1].kind != api.MemoryKindConstraint {
		t.Errorf("second kind: %+v", ds[1])
	}
}

func TestClassify_workSummaryFallback(t *testing.T) {
	long := "This is a long enough work summary for fallback when there are no structured answers at all."
	req := DigestRequest{WorkSummary: long}
	ds := Classify(req, 5)
	if len(ds) != 1 {
		t.Fatalf("expected 1 fallback draft, got %d (%+v)", len(ds), ds)
	}
	if ds[0].kind != api.MemoryKindState {
		t.Errorf("kind=%s want state", ds[0].kind)
	}
}

func TestClassify_neverAgain_emitsFailureAndConstraint(t *testing.T) {
	req := DigestRequest{
		WorkSummary: "Enough summary text for validation.",
		CurationAnswers: &DigestCurationAnswers{
			NeverAgain: "Never skip tests in release",
		},
	}
	ds := Classify(req, 10)
	var hasFail, hasCon bool
	for _, d := range ds {
		if d.kind == api.MemoryKindFailure {
			hasFail = true
		}
		if d.kind == api.MemoryKindConstraint && strings.Contains(d.reason, "never again") {
			hasCon = true
		}
	}
	if !hasFail || !hasCon {
		t.Fatalf("want failure+constraint from never_again, got %+v", ds)
	}
}

func TestClassify_respectsMax(t *testing.T) {
	req := DigestRequest{
		WorkSummary: "Enough summary text for validation.",
		CurationAnswers: &DigestCurationAnswers{
			Decision:   "D1",
			Constraint: "C1",
			Failure:    "F1",
		},
	}
	ds := Classify(req, 2)
	if len(ds) != 2 {
		t.Fatalf("expected cap 2, got %d %+v", len(ds), ds)
	}
}
