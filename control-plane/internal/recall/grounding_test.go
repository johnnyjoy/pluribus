package recall

import (
	"strings"
	"testing"

	"control-plane/pkg/api"
)

func TestPopulateAgentGrounding_fromGroupedSlices(t *testing.T) {
	b := &RecallBundle{
		Continuity: []MemoryItem{
			{ID: "1", Kind: string(api.MemoryKindDecision), Statement: "Use POST for creates."},
		},
		Constraints: []MemoryItem{
			{ID: "2", Kind: string(api.MemoryKindConstraint), Statement: "No duplicate query builders."},
		},
		Experience: []MemoryItem{
			{ID: "3", Kind: string(api.MemoryKindPattern), Statement: "Prefer integration tests."},
		},
	}
	populateAgentGrounding(b)
	if b.AgentGrounding == nil {
		t.Fatal("expected agent_grounding")
	}
	if !strings.Contains(b.AgentGrounding.Formatted, "Continuity:") || !strings.Contains(b.AgentGrounding.Formatted, "Use POST") {
		t.Fatalf("formatted: %q", b.AgentGrounding.Formatted)
	}
}

func TestPopulateAgentGrounding_fallbackBuckets(t *testing.T) {
	b := &RecallBundle{
		GoverningConstraints: []MemoryItem{
			{ID: "c1", Kind: string(api.MemoryKindConstraint), Statement: "Constraint text."},
		},
		ApplicablePatterns: []MemoryItem{
			{ID: "p1", Kind: string(api.MemoryKindPattern), Statement: "Pattern text."},
		},
	}
	populateAgentGrounding(b)
	if b.AgentGrounding == nil {
		t.Fatal("expected agent_grounding")
	}
	if !strings.Contains(b.AgentGrounding.Constraints, "Constraint text") {
		t.Fatalf("constraints: %q", b.AgentGrounding.Constraints)
	}
	if !strings.Contains(b.AgentGrounding.Experience, "Pattern text") {
		t.Fatalf("experience: %q", b.AgentGrounding.Experience)
	}
}
