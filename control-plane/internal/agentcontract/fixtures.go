package agentcontract

import (
	"control-plane/internal/recall"
	"control-plane/pkg/api"
)

func floatPtr(v float64) *float64 { return &v }
func boolPtr(v bool) *bool         { return &v }

// CanonicalContractMemory returns a contract-complete MemoryItem for hostile parity/coverage tests.
func CanonicalContractMemory(id string) recall.MemoryItem {
	return recall.MemoryItem{
		ID:              id,
		Kind:            "constraint",
		Statement:       "Canonical contract memory for endpoint coverage.",
		Authority:       3,
		Applicability:   api.ApplicabilityGoverning,
		Status:          "active",
		LifecycleRole:   recall.LifecycleCurrentGuidance,
		SchemaType:      "constraint",
		Scope:           "project:parity",
		NegativeScope:   []string{"project:wrong"},
		UseInstruction:  "Apply only when project:parity matches.",
		MisuseWarning:   "",
		SourceType:      "formationquality",
		AuthorityBasis:  "authority_basis:formed_from_gate",
		UtilityScore:    floatPtr(0.75),
		QualityState:    "accept_active",
		QualityScore:    floatPtr(0.92),
		SafeForActiveRecall: boolPtr(true),
	}
}

// CanonicalRecallBundle returns a bundle with one contract-complete memory item.
func CanonicalRecallBundle() *recall.RecallBundle {
	it := CanonicalContractMemory("mem_canonical_001")
	return &recall.RecallBundle{
		GoverningConstraints: []recall.MemoryItem{it},
		RecallPreamble:         "Phase 11G endpoint coverage stub.",
	}
}

// CanonicalWakeupResponse returns wakeup with contract-complete governing memory.
func CanonicalWakeupResponse() *recall.WakeupResponse {
	it := CanonicalContractMemory("mem_wakeup_001")
	return &recall.WakeupResponse{
		Identity:        []recall.MemoryItem{},
		GoverningMemory: []recall.MemoryItem{it},
		RecallPreamble:  "Wakeup stub.",
	}
}

// CanonicalCompileMultiResponse returns compile-multi with contract-complete bundles.
func CanonicalCompileMultiResponse() *recall.CompileMultiResponse {
	b := CanonicalRecallBundle()
	return &recall.CompileMultiResponse{
		Bundles: []recall.VariantBundle{
			{Variant: "balanced", Bundle: *b},
		},
	}
}
