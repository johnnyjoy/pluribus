package parity

import (
	"testing"

	"control-plane/internal/recall"
	"control-plane/pkg/api"
)

func TestCompareMemoryItemsParityPass(t *testing.T) {
	util := 0.7
	q := 0.9
	it := recall.MemoryItem{
		ID: "m1", Statement: "s", SchemaType: "constraint",
		LifecycleRole: recall.LifecycleCurrentGuidance, Status: "active",
		Applicability: api.ApplicabilityGoverning, Scope: "p",
		NegativeScope: []string{"n"}, UseInstruction: "u",
		SourceType: "formationquality", AuthorityBasis: "ab", Authority: 3,
		UtilityScore: &util, QualityState: "accept_active", QualityScore: &q,
	}
	r := CompareMemoryItems([]recall.MemoryItem{it}, []recall.MemoryItem{it})
	if !r.ParityPassed || FieldMismatchCount(r) != 0 {
		t.Fatalf("expected parity pass, got %+v", r)
	}
}

func TestCompareMemoryItemsDetectsMismatch(t *testing.T) {
	util := 0.7
	q := 0.9
	rest := recall.MemoryItem{ID: "m1", Statement: "a", SchemaType: "constraint", LifecycleRole: recall.LifecycleCurrentGuidance, Status: "active", Scope: "p", NegativeScope: []string{"n"}, UseInstruction: "u", SourceType: "s", AuthorityBasis: "ab", Authority: 3, UtilityScore: &util, QualityState: "accept_active", QualityScore: &q}
	mcp := rest
	mcp.Statement = "b"
	r := CompareMemoryItems([]recall.MemoryItem{rest}, []recall.MemoryItem{mcp})
	if r.ParityPassed {
		t.Fatal("expected mismatch")
	}
}
