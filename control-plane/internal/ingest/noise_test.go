package ingest

import (
	"testing"
)

func TestNoiseRejectReason_effectiveConfidence(t *testing.T) {
	t.Parallel()
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"Reviewed go.mod and verified dependency declarations for the service", "Second step with enough characters total"},
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c", Confidence: ptrFloat(0.01)}},
		Confidence:        0.99,
		ContextWindowHash: "h",
	}
	if s := NoiseRejectReason(req, 1.0, 0.15, 0); s == "" {
		t.Fatal("expected reject")
	}
	req.ExtractedFacts[0].Confidence = ptrFloat(0.2)
	if s := NoiseRejectReason(req, 1.0, 0.15, 0); s != "" {
		t.Fatalf("expected ok, got %q", s)
	}
}

func TestNoiseRejectReason_highTrustPassesMarginal(t *testing.T) {
	t.Parallel()
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"Reviewed go.mod and verified dependency declarations for the service", "Second step with enough characters total"},
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c", Confidence: ptrFloat(0.06)}},
		Confidence:        0.99,
		ContextWindowHash: "h",
	}
	if s := NoiseRejectReason(req, 1.0, 0.15, 0); s == "" {
		t.Fatal("expected reject at trust 1.0")
	}
	if s := NoiseRejectReason(req, 3.0, 0.15, 0); s != "" {
		t.Fatalf("expected ok at trust 3.0, got %q", s)
	}
}

func TestNoiseRejectReason_shortTrace(t *testing.T) {
	t.Parallel()
	req := CognitionRequest{
		TempContributorID: "c",
		Query:             "q",
		ReasoningTrace:    []string{"x", "y"},
		ExtractedFacts:    []ExtractedFact{{Subject: "a", Predicate: "b", Object: "c"}},
		Confidence:        0.9,
		ContextWindowHash: "h",
	}
	if s := NoiseRejectReason(req, 1.0, 0.15, 40); s == "" {
		t.Fatal("expected trace reject")
	}
}

func ptrFloat(f float64) *float64 { return &f }
