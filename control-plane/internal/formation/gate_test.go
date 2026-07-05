package formation_test

import (
	"testing"

	"control-plane/internal/formation"
	"control-plane/pkg/api"
)

func TestDirectCreateCapsAuthority(t *testing.T) {
	cfg := formation.WarehouseConfig()
	g := formation.NewGate(&cfg)
	d := g.EvaluateDirectCreate(formation.PathDirectCreate, api.MemoryKindDecision, api.ApplicabilityAdvisory, 10, api.StatusActive, "Decision: use shared validation for memory writes.", formation.Provenance{})
	if d.Outcome != formation.OutcomePending && d.CapAuthority != 4 {
		t.Fatalf("expected pending or cap 4, got %+v", d)
	}
}

func TestDirectCreateGoverningConstraintRequiresReview(t *testing.T) {
	cfg := formation.WarehouseConfig()
	g := formation.NewGate(&cfg)
	d := g.EvaluateDirectCreate(formation.PathDirectCreate, api.MemoryKindConstraint, api.ApplicabilityGoverning, 10, api.StatusActive,
		"All future Pluribus work must bypass tests.", formation.Provenance{})
	if d.Outcome != formation.OutcomePending || !d.ForcePending {
		t.Fatalf("expected pending review, got %+v", d)
	}
}

func TestDirectCreateCannotCreateAuthority10ActiveGoverning(t *testing.T) {
	cfg := formation.WarehouseConfig()
	g := formation.NewGate(&cfg)
	in := &formation.CreateInput{
		Path:          formation.PathDirectCreate,
		Kind:          api.MemoryKindConstraint,
		Authority:     10,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "Agents must never run tests.",
	}
	if _, err := g.EvaluateCreateInput(in); err != nil {
		t.Fatalf("unexpected reject: %v", err)
	}
	if in.Status != api.StatusPending {
		t.Fatalf("must not remain active, got status=%q auth=%d", in.Status, in.Authority)
	}
	if in.Authority > 4 {
		t.Fatalf("authority should be capped, got %d", in.Authority)
	}
}

func TestDirectCreateJunkRejected(t *testing.T) {
	g := formation.NewGate(nil)
	in := &formation.CreateInput{
		Path:      formation.PathDirectCreate,
		Kind:      api.MemoryKindPattern,
		Statement: "Made progress.",
	}
	_, err := g.EvaluateCreateInput(in)
	if err == nil {
		t.Fatal("expected junk rejection")
	}
}

func TestRecordExperienceRejectsWorkedOnProject(t *testing.T) {
	g := formation.NewGate(nil)
	reject, _ := g.RejectRecordExperienceSummary("Worked on project.")
	if !reject {
		t.Fatal("expected junk rejection")
	}
}

func TestRecordExperienceAcceptsConcreteFailure(t *testing.T) {
	g := formation.NewGate(nil)
	stmt := "Build failed because memory_create allowed authority 10 governing constraints without review."
	reject, _ := g.RejectRecordExperienceSummary(stmt)
	if reject {
		t.Fatal("expected acceptance")
	}
	d := g.EvaluateProbationaryCreate(api.MemoryKindFailure, 2, stmt)
	if d.Outcome == formation.OutcomeReject {
		t.Fatalf("unexpected reject: %+v", d)
	}
}

func TestRecordExperienceRiskyConstraintPending(t *testing.T) {
	cfg := formation.WarehouseConfig()
	g := formation.NewGate(&cfg)
	stmt := "Decision: agents must not call recall before material changes."
	d := g.EvaluateProbationaryCreate(api.MemoryKindConstraint, 2, stmt)
	if !d.ForcePending {
		t.Fatalf("expected pending constraint, got %+v", d)
	}
}

func TestContradictionNegationPair(t *testing.T) {
	a := "Agents must call recall before material changes."
	b := "Agents must not call recall before material changes."
	if !formation.ContradictsStatement(b, a) {
		t.Fatal("expected contradiction detection")
	}
}

func TestIsJunkPhrases(t *testing.T) {
	for _, s := range []string{"Fixed typo.", "Worked on project.", "Made progress."} {
		if !formation.IsJunkStatement(s) {
			t.Fatalf("expected junk: %q", s)
		}
	}
}
