package formation

import (
	"testing"

	"control-plane/pkg/api"
)

func TestProofLikeConfig_keepsHighAuthorityGoverningActive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DirectCreate.MaxClientAuthority = 10
	cfg.DirectCreate.HighRiskAuthorityThreshold = 11
	cfg.DirectCreate.GoverningDefaultStatus = "active"
	cfg.DirectCreate.RequireProvenanceForGoverning = false
	cfg.DirectCreate.RequireAdminForGoverning = false
	cfg.DirectCreate.RequireProvenanceAuthorityGE = 11
	cfg.DirectCreate.AllowActiveHighRiskGoverning = true
	cfg.DirectCreate.AllowActiveWithQualityWarnings = true
	cfg.DirectCreate.AllowActiveDespiteQualityReview = true

	g := NewGate(&cfg)
	in := &CreateInput{
		Path:          PathMemoriesCreate,
		Kind:          api.MemoryKindDecision,
		Authority:     9,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "PROOF SIGNAL HUB UNIQUE test marker for formation gate",
	}
	d := g.EvaluateDirectCreate(in.Path, in.Kind, in.Applicability, in.Authority, in.Status, in.Statement, Provenance{})
	if d.ForcePending {
		t.Fatalf("EvaluateDirectCreate ForcePending=true reason=%q outcome=%s", d.Reason, d.Outcome)
	}
	if _, err := g.EvaluateCreateInput(in); err != nil {
		t.Fatalf("EvaluateCreateInput: %v", err)
	}
	if in.Status != api.StatusActive {
		if g.Config().DirectCreate.AllowActiveWithQualityWarnings {
			t.Fatalf("status=%q want active (quality=%v)", in.Status, in)
		}
		t.Fatalf("status=%q want active", in.Status)
	}
}

func TestProofLikeConfig_keepsPromotedGoverningConstraintActive(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DirectCreate.MaxClientAuthority = 10
	cfg.DirectCreate.HighRiskAuthorityThreshold = 11
	cfg.DirectCreate.GoverningDefaultStatus = "active"
	cfg.DirectCreate.RequireProvenanceForGoverning = false
	cfg.DirectCreate.RequireAdminForGoverning = false
	cfg.DirectCreate.RequireProvenanceAuthorityGE = 11
	cfg.DirectCreate.AllowActiveHighRiskGoverning = true
	cfg.DirectCreate.AllowActivePromotedGoverning = true
	cfg.DirectCreate.AllowActiveDespiteQualityReview = true

	g := NewGate(&cfg)
	in := &CreateInput{
		Path:          PathPromote,
		Kind:          api.MemoryKindConstraint,
		Authority:     3,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "EPISODIC PROOF MARKER never use SQLite for durable project database storage",
	}
	if _, err := g.EvaluateCreateInput(in); err != nil {
		t.Fatalf("EvaluateCreateInput: %v", err)
	}
	if in.Status != api.StatusActive {
		t.Fatalf("status=%q want active after promote path with proof bypass", in.Status)
	}
}
