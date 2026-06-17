package recall

import (
	"context"
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestCompiler_Compile_skipExperienceHydration_skipsExperienceLister(t *testing.T) {
	synthID := uuid.New()
	fakeMem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindPattern, Statement: "from-search", Authority: 5, Applicability: api.ApplicabilityGoverning},
		},
	}
	exp := &fakeExperienceLister{
		objs: []memory.MemoryObject{
			{ID: synthID, Kind: api.MemoryKindPattern, Statement: "from-jsonl", Authority: 9, Applicability: api.ApplicabilityGoverning},
		},
	}
	c := &Compiler{Memory: fakeMem, Experiences: exp}
	req := CompileRequest{MaxPerKind: 5, SkipExperienceHydration: true}
	b, err := c.Compile(context.Background(), req)
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(b.ApplicablePatterns) != 1 {
		t.Fatalf("patterns: got %d want 1", len(b.ApplicablePatterns))
	}
	if b.ApplicablePatterns[0].ID == synthID.String() {
		t.Fatal("expected experience-derived memory to be skipped when SkipExperienceHydration is true")
	}
}

func TestBuildWakeupResponse_identityStateOnly_andGoverningFilter(t *testing.T) {
	b := &RecallBundle{
		Continuity: []MemoryItem{
			{ID: "s1", Kind: "state", Statement: "mission", Authority: 8, Applicability: api.ApplicabilityGoverning},
			{ID: "d1", Kind: "decision", Statement: "choose X", Authority: 9, Applicability: api.ApplicabilityGoverning},
		},
		GoverningConstraints: []MemoryItem{
			{ID: "c1", Kind: "constraint", Statement: "must", Authority: 7, Applicability: api.ApplicabilityGoverning},
		},
		ApplicablePatterns: []MemoryItem{
			{ID: "p1", Kind: "pattern", Statement: "adv", Authority: 10, Applicability: api.ApplicabilityAdvisory},
			{ID: "p2", Kind: "pattern", Statement: "gov", Authority: 3, Applicability: api.ApplicabilityGoverning},
		},
		RecallPreamble: "Context informed by prior experience.",
	}
	out := BuildWakeupResponse(b, 4, 10)
	if len(out.Identity) != 1 || out.Identity[0].ID != "s1" {
		t.Fatalf("identity: %+v", out.Identity)
	}
	var sawP2 bool
	for _, it := range out.GoverningMemory {
		if it.ID == "p1" {
			t.Fatal("advisory pattern must not appear in governing_memory")
		}
		if it.ID == "p2" {
			sawP2 = true
		}
	}
	if !sawP2 {
		t.Fatal("expected governing pattern in governing_memory")
	}
	if out.RecallPreamble != b.RecallPreamble {
		t.Fatalf("preamble: %q", out.RecallPreamble)
	}
}

func TestBuildWakeupResponse_maxGoverningTotal(t *testing.T) {
	b := &RecallBundle{
		GoverningConstraints: []MemoryItem{
			{ID: "c1", Kind: "constraint", Statement: "a", Authority: 1, Applicability: api.ApplicabilityGoverning},
			{ID: "c2", Kind: "constraint", Statement: "b", Authority: 2, Applicability: api.ApplicabilityGoverning},
		},
		Decisions: []MemoryItem{
			{ID: "d1", Kind: "decision", Statement: "c", Authority: 3, Applicability: api.ApplicabilityGoverning},
		},
	}
	out := BuildWakeupResponse(b, 2, 2)
	if len(out.GoverningMemory) != 2 {
		t.Fatalf("governing len %d want 2", len(out.GoverningMemory))
	}
}

func TestBuildWakeupResponse_emptyApplicabilityTreatedAsGoverning(t *testing.T) {
	b := &RecallBundle{
		GoverningConstraints: []MemoryItem{
			{ID: "c1", Kind: "constraint", Statement: "legacy", Authority: 5, Applicability: ""},
		},
	}
	out := BuildWakeupResponse(b, 2, 5)
	if len(out.GoverningMemory) != 1 {
		t.Fatalf("governing: %+v", out.GoverningMemory)
	}
}

func TestService_Wakeup_noReinforcementAndUsesCompiler(t *testing.T) {
	// Service.Compile reinforces; Wakeup must not call Compile (cache/reinforce path).
	fakeMem := &fakeMemorySearcher{
		objs: []memory.MemoryObject{
			{ID: uuid.New(), Kind: api.MemoryKindState, Statement: "role", Authority: 6, Applicability: api.ApplicabilityGoverning},
			{ID: uuid.New(), Kind: api.MemoryKindConstraint, Statement: "c", Authority: 5, Applicability: api.ApplicabilityGoverning},
		},
	}
	compiler := &Compiler{Memory: fakeMem}
	svc := &Service{Compiler: compiler}
	out, err := svc.Wakeup(context.Background(), WakeupRequest{})
	if err != nil {
		t.Fatalf("Wakeup: %v", err)
	}
	if len(out.Identity) != 1 || out.Identity[0].Statement != "role" {
		t.Fatalf("identity: %+v", out.Identity)
	}
	if len(out.GoverningMemory) < 1 {
		t.Fatalf("governing: %+v", out.GoverningMemory)
	}
}
