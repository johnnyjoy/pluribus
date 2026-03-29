package curation

import "testing"

func TestHasImperativeGuardrailLanguage(t *testing.T) {
	if !hasImperativeGuardrailLanguage("Never skip tests in the release path") {
		t.Fatal("expected imperative")
	}
	if hasImperativeGuardrailLanguage("We should consider tests") {
		t.Fatal("expected non-imperative")
	}
}

func TestConstraintFromDecision_postgresDurable(t *testing.T) {
	s, ok := constraintFromDecision("Postgres is the only durable store for control-plane data; SQLite is not acceptable.")
	if !ok || s == "" {
		t.Fatalf("expected constraint, got ok=%v s=%q", ok, s)
	}
}

func TestShapeConstraintStatement(t *testing.T) {
	if got := shapeConstraintStatement("  Never foo  "); got != "Never foo" {
		t.Fatalf("got %q", got)
	}
}
