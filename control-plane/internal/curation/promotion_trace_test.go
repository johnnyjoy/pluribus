package curation

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestBuildMaterializePayload_includesEvolution(t *testing.T) {
	cid := uuid.MustParse("c0000000-0000-0000-0000-000000000099")
	p := &ProposalPayloadV1{
		DistillSupportCount: 2,
		PluribusEvolution: &PluribusEvolutionV1{
			InvalidatedBy: "a0000000-0000-0000-0000-000000000001",
			Contradicts:   []string{"b0000000-0000-0000-0000-000000000002"},
		},
	}
	raw := buildMaterializePayload(cid, p)
	if raw == nil {
		t.Fatal("nil payload")
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(*raw, &m); err != nil {
		t.Fatal(err)
	}
	if _, ok := m["pluribus_promotion"]; !ok {
		t.Fatal("missing pluribus_promotion")
	}
	var evo map[string]any
	if err := json.Unmarshal(m["pluribus_evolution"], &evo); err != nil {
		t.Fatal(err)
	}
	if evo["invalidated_by"] != "a0000000-0000-0000-0000-000000000001" {
		t.Fatalf("invalidated_by: %+v", evo)
	}
	if !strings.Contains(string(*raw), "contradicts") {
		t.Fatal("expected contradicts in JSON")
	}
}
