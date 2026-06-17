package agentcontract

import (
	"testing"

	"control-plane/internal/recall"

	"control-plane/pkg/api"
)

func fptr(v float64) *float64 { return &v }

func bptr(v bool) *bool { return &v }

func TestEvaluateBundleContract_CurrentGuidanceCompletePasses(t *testing.T) {
	it := recall.MemoryItem{
		ID:             "mem_001",
		Kind:           "constraint",
		Statement:     "do X",
		Authority:     3,
		Status:        string(api.StatusActive),
		Applicability: api.ApplicabilityGoverning,
		LifecycleRole: recall.LifecycleCurrentGuidance,
		SchemaType:    "constraint",
		Scope:         "project:payments",
		NegativeScope: []string{"project:wrong"},
		UseInstruction: "if scope matches, do X",
		MisuseWarning:  "",
		SourceType:     "formationquality",
		AuthorityBasis: "provenance",
		QualityScore:   fptr(0.9),
		QualityState:   "accept_active",
		SafeForActiveRecall: bptr(true),
		UtilityScore:   fptr(0.7),
	}

	b := &recall.RecallBundle{
		GoverningConstraints: []recall.MemoryItem{it},
	}

	ev := EvaluateBundleContract(b, recall.RecallModeCurrent, false)
	if !ev.ContractPassed {
		t.Fatalf("expected contract_passed=true, got false: %+v", ev)
	}
	if ev.BundleContractScore < 0.99 {
		t.Fatalf("expected bundle score ~1, got %v", ev.BundleContractScore)
	}
}

func TestEvaluateBundleContract_HistoricalRequiresMisuseWarning(t *testing.T) {
	it := recall.MemoryItem{
		ID:             "mem_002",
		Kind:           "constraint",
		Statement:      "historical X",
		Status:         string(api.StatusActive),
		Applicability:  api.ApplicabilityAdvisory,
		LifecycleRole:  recall.LifecycleHistoricalContext,
		SchemaType:     "constraint",
		SourceType:     "formationquality",
		AuthorityBasis: "provenance",
		QualityScore:   fptr(0.7),
		QualityState:   "accept_active",
		SafeForActiveRecall: bptr(true),
		UtilityScore:   fptr(0.4),
		// MisuseWarning intentionally missing.
	}

	b := &recall.RecallBundle{
		Constraints: []recall.MemoryItem{it},
	}

	ev := EvaluateBundleContract(b, recall.RecallModeHistorical, false)
	if ev.ContractPassed {
		t.Fatalf("expected contract_passed=false")
	}
	found := false
	for _, x := range ev.UnsafeOmissions {
		if x == DefectMissingMisuseWarningForHistorical {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected misuse warning defect %q in unsafe omissions; got %+v", DefectMissingMisuseWarningForHistorical, ev.UnsafeOmissions)
	}
}

func TestEvaluateBundleContract_FlattensTextOnlyRejected(t *testing.T) {
	it := recall.MemoryItem{
		ID:             "mem_003",
		Kind:           "constraint",
		Statement:     "do X",
		Status:        string(api.StatusActive),
		Applicability: api.ApplicabilityGoverning,
		LifecycleRole: recall.LifecycleCurrentGuidance,
		SchemaType:    "constraint",
		Scope:         "project:payments",
		NegativeScope: []string{"project:wrong"},
		UseInstruction: "if scope matches, do X",
		SourceType:     "formationquality",
		AuthorityBasis: "provenance",
		QualityScore:   fptr(0.9),
		QualityState:   "accept_active",
		SafeForActiveRecall: bptr(true),
		UtilityScore:   fptr(0.7),
	}
	b := &recall.RecallBundle{
		GoverningConstraints: []recall.MemoryItem{it},
	}

	ev := EvaluateBundleContract(b, recall.RecallModeCurrent, true)
	if ev.ContractPassed {
		t.Fatalf("expected contract_passed=false for flattened text-only input")
	}
	found := false
	for _, x := range ev.UnsafeOmissions {
		if x == DefectFlattenedTextOnlyMemory {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected defect %q; got %+v", DefectFlattenedTextOnlyMemory, ev.UnsafeOmissions)
	}
}

