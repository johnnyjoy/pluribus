package curation

import (
	"strings"
	"testing"

	"control-plane/pkg/api"
)

func TestClassifyPromotionReadiness_lowSupport(t *testing.T) {
	p := &ProposalPayloadV1{
		Kind:                api.MemoryKindFailure,
		Statement:           "retry without idempotency causes duplicate rows in production",
		DistillSupportCount: 1,
	}
	r, msg := ClassifyPromotionReadiness(p, 0.4)
	if r != ReadinessNotReady {
		t.Fatalf("got %s %s", r, msg)
	}
}

func TestClassifyPromotionReadiness_highConfidence(t *testing.T) {
	p := &ProposalPayloadV1{
		Kind:                api.MemoryKindFailure,
		Statement:           "retry without idempotency causes duplicate rows in production",
		DistillSupportCount: 5,
	}
	r, msg := ClassifyPromotionReadiness(p, 0.8)
	if r != ReadinessHighConfidence {
		t.Fatalf("got %s", r)
	}
	if !strings.Contains(msg, "5") {
		t.Fatalf("expected support in reason: %s", msg)
	}
}
