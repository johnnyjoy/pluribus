package formation_test

import (
	"testing"

	"control-plane/internal/formation"
	"control-plane/pkg/api"
)

func TestHiveConfig_governingConstraintActiveAtCappedAuthority(t *testing.T) {
	cfg := formation.HiveConfig()
	g := formation.NewGate(&cfg)
	in := &formation.CreateInput{
		Path:          formation.PathDirectCreate,
		Kind:          api.MemoryKindConstraint,
		Authority:     9,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "All durable project data must use Postgres; SQLite is not permitted.",
	}
	d, err := g.EvaluateCreateInput(in)
	if err != nil {
		t.Fatalf("unexpected reject: %v", err)
	}
	if d.Outcome != formation.OutcomeAllow {
		t.Fatalf("expected allow, got %+v", d)
	}
	if in.Status != api.StatusActive {
		t.Fatalf("expected active status, got %q", in.Status)
	}
	if in.Authority != 4 {
		t.Fatalf("expected authority capped to 4, got %d", in.Authority)
	}
}

func TestHiveConfig_materializeDecisionActive(t *testing.T) {
	cfg := formation.HiveConfig()
	g := formation.NewGate(&cfg)
	in := &formation.CreateInput{
		Path:      formation.PathPromote,
		Kind:      api.MemoryKindDecision,
		Authority: 7,
		Status:    api.StatusActive,
		Statement: "We will use feature flags for rollout of the new API surface.",
	}
	d, err := g.EvaluateCreateInput(in)
	if err != nil {
		t.Fatalf("unexpected reject: %v", err)
	}
	if d.Outcome != formation.OutcomeAllow {
		t.Fatalf("expected allow, got %+v", d)
	}
	if in.Status != api.StatusActive {
		t.Fatalf("expected active, got %q", in.Status)
	}
	if in.Authority != 4 {
		t.Fatalf("expected authority cap 4, got %d", in.Authority)
	}
}

func TestDefaultConfig_hiveActiveIngest(t *testing.T) {
	g := formation.NewGate(nil)
	in := &formation.CreateInput{
		Path:          formation.PathDirectCreate,
		Kind:          api.MemoryKindConstraint,
		Authority:     9,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "All durable project data must use Postgres; SQLite is not permitted.",
	}
	if _, err := g.EvaluateCreateInput(in); err != nil {
		t.Fatalf("unexpected reject: %v", err)
	}
	if in.Status != api.StatusActive {
		t.Fatalf("hive default should stay active, got %q", in.Status)
	}
	if in.Authority != 4 {
		t.Fatalf("expected authority capped to 4, got %d", in.Authority)
	}
}

func TestWarehouseConfig_stillPendingForHighAuthorityGoverning(t *testing.T) {
	cfg := formation.WarehouseConfig()
	g := formation.NewGate(&cfg)
	in := &formation.CreateInput{
		Path:          formation.PathDirectCreate,
		Kind:          api.MemoryKindConstraint,
		Authority:     9,
		Applicability: api.ApplicabilityGoverning,
		Status:        api.StatusActive,
		Statement:     "All durable project data must use Postgres; SQLite is not permitted.",
	}
	if _, err := g.EvaluateCreateInput(in); err != nil {
		t.Fatalf("unexpected reject: %v", err)
	}
	if in.Status != api.StatusPending {
		t.Fatalf("warehouse mode should stay pending, got %q", in.Status)
	}
}
