package ingest

import (
	"testing"
)

func TestValidate_rejectsMissingTempContributorID(t *testing.T) {
	req := CognitionRequest{
		Query:             "q",
		ReasoningTrace:    []string{"x"},
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c"}},
		Confidence:        1,
		ContextWindowHash: "h",
	}
	if s := Validate(req, DefaultLimits(), true); s != "temp_contributor_id: required" {
		t.Fatalf("got %q", s)
	}
}

func TestValidate_acceptsMinimalValid(t *testing.T) {
	req := CognitionRequest{
		TempContributorID: "client-a",
		Query:             "why",
		ReasoningTrace:    []string{"step one"},
		ExtractedFacts:    []ExtractedFact{{Subject: "svc", Predicate: "uses", Object: "postgres"}},
		Confidence:        0.5,
		ContextWindowHash: "abc123",
	}
	if s := Validate(req, DefaultLimits(), true); s != "" {
		t.Fatalf("expected ok, got %q", s)
	}
}

func TestValidate_rejectsMissingTrace(t *testing.T) {
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    nil,
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c"}},
		Confidence:        1,
		ContextWindowHash: "h",
	}
	if s := Validate(req, DefaultLimits(), true); s != "reasoning_trace: at least one step required" {
		t.Fatalf("got %q", s)
	}
}

func TestValidate_rejectsEmptyFacts(t *testing.T) {
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"x"},
		ExtractedFacts:    []ExtractedFact{},
		Confidence:        1,
		ContextWindowHash: "h",
	}
	if s := Validate(req, DefaultLimits(), true); s != "extracted_facts: at least one fact required" {
		t.Fatalf("got %q", s)
	}
}

func TestValidate_rejectsMissingContextHashWhenRequired(t *testing.T) {
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"x"},
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c"}},
		Confidence:        1,
		ContextWindowHash: "",
	}
	if s := Validate(req, DefaultLimits(), true); s != "context_window_hash: required" {
		t.Fatalf("got %q", s)
	}
	if s := Validate(req, DefaultLimits(), false); s != "" {
		t.Fatalf("optional hash should pass, got %q", s)
	}
}

func TestValidate_rejectsFactConfidenceOutOfRange(t *testing.T) {
	bad := 1.1
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"x"},
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c", Confidence: &bad}},
		Confidence:        1,
		ContextWindowHash: "h",
	}
	if s := Validate(req, DefaultLimits(), true); s != "extracted_facts[0].confidence: must be between 0 and 1" {
		t.Fatalf("got %q", s)
	}
}
