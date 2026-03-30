package curation

import (
	"strings"
	"testing"

	"control-plane/pkg/api"
)

func TestGroupTags_entityAndDomain(t *testing.T) {
	g := groupTags([]string{"entity:redis", "distilled-from-advisory", "entity:redis", "ops"})
	if len(g.Entities) != 1 || g.Entities[0] != "redis" {
		t.Fatalf("entities: %+v", g.Entities)
	}
	if len(g.Domain) != 2 {
		t.Fatalf("domain: %+v", g.Domain)
	}
}

func TestBuildExplanation_containsKindAndSupport(t *testing.T) {
	p := &ProposalPayloadV1{
		Kind:              api.MemoryKindFailure,
		Statement:         "Retry without idempotency causes duplicate rows.",
		Tags:              []string{"entity:db"},
		DistillSupportCount: 3,
	}
	g := groupTags(p.Tags)
	ex := buildExplanation(p, "", 0.72, g)
	if !strings.Contains(ex, "failure") || !strings.Contains(ex, "3 supporting") {
		t.Fatalf("explanation: %q", ex)
	}
	if !strings.Contains(ex, "db") {
		t.Fatalf("want entity name: %q", ex)
	}
}

func TestComputeSignalStrength_tiers(t *testing.T) {
	p := &ProposalPayloadV1{Kind: api.MemoryKindFailure, DistillSupportCount: 1}
	s, _ := computeSignalStrength(p, 0.3)
	if s != "low" {
		t.Fatalf("got %s", s)
	}
	p.DistillSupportCount = 2
	s, _ = computeSignalStrength(p, 0.3)
	if s != "moderate" {
		t.Fatalf("got %s", s)
	}
	p.DistillSupportCount = 4
	s, _ = computeSignalStrength(p, 0.3)
	if s != "strong" {
		t.Fatalf("got %s", s)
	}
	s, _ = computeSignalStrength(&ProposalPayloadV1{Kind: api.MemoryKindFailure, DistillSupportCount: 1}, 0.9)
	if s != "strong" {
		t.Fatalf("high salience: got %s", s)
	}
}

func TestBuildPromotionPreview_constraintGoverning(t *testing.T) {
	p := &ProposalPayloadV1{
		Kind:              api.MemoryKindConstraint,
		Statement:         "Never skip health checks.",
		Tags:              []string{"entity:api"},
		ProposedAuthority: 0,
	}
	prev := buildPromotionPreview(p, &PromotionDigestConfig{RequireReview: true})
	if prev == nil {
		t.Fatal("nil preview")
	}
	if prev.Applicability != "governing" {
		t.Fatalf("got %s", prev.Applicability)
	}
	if prev.ProposedAuthority != 8 {
		t.Fatalf("authority %d", prev.ProposedAuthority)
	}
	if !strings.Contains(prev.MemoryStatusNote, "pending") {
		t.Fatalf("note: %s", prev.MemoryStatusNote)
	}
}
