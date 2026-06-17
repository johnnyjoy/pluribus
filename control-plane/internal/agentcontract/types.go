package agentcontract

import "control-plane/internal/recall"

// ContractEvaluation is the deterministic agent-facing memory contract check output.
// It is intentionally simple: no LLMs, no subjective scoring.
type ContractEvaluation struct {
	ContractPassed         bool                         `json:"contract_passed"`
	MissingRequiredFields  []string                    `json:"missing_required_fields"`
	UnsafeOmissions        []string                    `json:"unsafe_omissions"`
	Warnings                []string                    `json:"warnings"`
	MemoryContractScores   map[string]float64          `json:"memory_contract_scores"`
	BundleContractScore    float64                     `json:"bundle_contract_score"`
}

type UseDisciplineDecision struct {
	// Decision is one of: use, historical_only, ignore, unsafe.
	Decision string `json:"decision"`
	// Reason is deterministic and machine-readable.
	Reason string `json:"reason"`
}

// Contract fields required by Phase 11F.
// These are the defect codes used by the deterministic contract evaluator.
const (
	DefectMissingMemoryID                       = "missing_memory_id"
	DefectMissingStatement                      = "missing_statement"
	DefectMissingSchemaType                    = "missing_schema_type"
	DefectMissingLifecycleRole                 = "missing_lifecycle_role"
	DefectMissingStatus                         = "missing_status"
	DefectMissingScopeForGuidance             = "missing_scope_for_guidance"
	DefectMissingNegativeScopeForUniversalRule = "missing_negative_scope_for_universal_rule"
	DefectMissingUseInstructionForGuidance    = "missing_use_instruction_for_guidance"
	DefectMissingMisuseWarningForHistorical  = "missing_misuse_warning_for_historical"
	DefectMissingProvenanceSummary            = "missing_provenance_summary"
	DefectMissingSourceType                   = "missing_source_type"
	DefectMissingAuthorityBasis               = "missing_authority_basis"
	DefectMissingQualityScore                 = "missing_quality_score"
	DefectMissingUtilityScore                 = "missing_utility_score"
	DefectMissingQualityState                  = "missing_quality_state"
	DefectMissingSupersessionMarker          = "missing_supersession_marker"
	DefectMissingRefutationMarker            = "missing_refutation_marker"
	DefectHistoricalReturnedWithoutHistoricalLabel = "historical_returned_without_historical_label"
	DefectSupersededReturnedAsCurrent        = "superseded_returned_as_current"
	DefectRefutedReturnedAsCurrent            = "refuted_returned_as_current"
	DefectWrongScopeWithoutWarning           = "wrong_scope_without_warning"
	DefectFlattenedTextOnlyMemory            = "flattened_text_only_memory"
	DefectMCPTextOnlyWithoutJSON             = "mcp_text_only_without_json"
	DefectNegativeScopeWithoutSuppression    = "negative_scope_without_suppression"
	DefectMCPRESTContractMismatch           = "mcp_rest_contract_mismatch"
)

// InterfaceType is the recall surface being validated.
type InterfaceType string

const (
	InterfaceREST InterfaceType = "rest"
	InterfaceMCP  InterfaceType = "mcp"
)

// EvaluateBundleContract validates an agent-facing contract completeness check
// for one recall bundle returned by either primary interface.
func EvaluateBundleContract(b *recall.RecallBundle, expectedRecallMode recall.RecallMode, includeMCPFlattenedText bool) ContractEvaluation {
	ev := ContractEvaluation{
		ContractPassed:      true,
		MissingRequiredFields:  nil,
		UnsafeOmissions:    nil,
		Warnings:           nil,
		MemoryContractScores: map[string]float64{},
		BundleContractScore: 1.0,
	}

	if b == nil {
		ev.ContractPassed = false
		ev.UnsafeOmissions = append(ev.UnsafeOmissions, DefectMissingMemoryID)
		ev.BundleContractScore = 0.0
		return ev
	}

	all := collectMemoryItems(b)
	if includeMCPFlattenedText {
		ev.ContractPassed = false
		ev.UnsafeOmissions = append(ev.UnsafeOmissions, DefectMCPTextOnlyWithoutJSON)
		ev.UnsafeOmissions = append(ev.UnsafeOmissions, DefectFlattenedTextOnlyMemory)
		ev.BundleContractScore = 0.0
		for _, it := range all {
			ev.MemoryContractScores[it.ID] = 0.0
		}
		return ev
	}

	missingAgg := map[string]struct{}{}
	unsafeAgg := map[string]struct{}{}

	var total float64
	var count int

	for _, it := range all {
		memMissing := []string{}
		memUnsafe := []string{}

		if it.ID == "" {
			memMissing = append(memMissing, DefectMissingMemoryID)
		}
		if it.Statement == "" {
			memMissing = append(memMissing, DefectMissingStatement)
		}
		if it.SchemaType == "" {
			memMissing = append(memMissing, DefectMissingSchemaType)
		}
		if it.LifecycleRole == "" {
			memMissing = append(memMissing, DefectMissingLifecycleRole)
		}
		if it.Status == "" {
			memMissing = append(memMissing, DefectMissingStatus)
		}
		// Contract requires lifecycle-aware guidance fields.
		switch it.LifecycleRole {
		case recall.LifecycleCurrentGuidance:
			if it.Scope == "" {
				memUnsafe = append(memUnsafe, DefectMissingScopeForGuidance)
			}
			if len(it.NegativeScope) == 0 {
				memUnsafe = append(memUnsafe, DefectMissingNegativeScopeForUniversalRule)
			}
			if it.UseInstruction == "" {
				memUnsafe = append(memUnsafe, DefectMissingUseInstructionForGuidance)
			}
		case recall.LifecycleHistoricalContext:
			if it.MisuseWarning == "" {
				memUnsafe = append(memUnsafe, DefectMissingMisuseWarningForHistorical)
			}
		}

		// Hostile lifecycle-mode checks:
		// In current recall mode, historical/superseded/refuted items must not be exposed as
		// if they were current guidance.
		if expectedRecallMode == recall.RecallModeCurrent {
			switch it.LifecycleRole {
			case recall.LifecycleHistoricalContext:
				memUnsafe = append(memUnsafe, DefectHistoricalReturnedWithoutHistoricalLabel)
			case recall.LifecycleSupersededContext:
				memUnsafe = append(memUnsafe, DefectSupersededReturnedAsCurrent)
			case recall.LifecycleRefutedContext:
				memUnsafe = append(memUnsafe, DefectRefutedReturnedAsCurrent)
			}
		}

		if it.SourceType == "" && it.AuthorityBasis == "" {
			memMissing = append(memMissing, DefectMissingProvenanceSummary)
		} else if it.SourceType == "" {
			memMissing = append(memMissing, DefectMissingSourceType)
		} else if it.AuthorityBasis == "" {
			memMissing = append(memMissing, DefectMissingAuthorityBasis)
		}

		if it.UtilityScore == nil {
			memMissing = append(memMissing, DefectMissingUtilityScore)
		}
		if it.QualityState == "" {
			memMissing = append(memMissing, DefectMissingQualityState)
		}
		if it.QualityScore == nil {
			memMissing = append(memMissing, DefectMissingQualityScore)
		}

		// Supersession/refutation markers exist via lifecycle_role plus superseded_by marker.
		switch it.LifecycleRole {
		case recall.LifecycleSupersededContext:
			if it.SupersededBy == "" {
				memUnsafe = append(memUnsafe, DefectMissingSupersessionMarker)
			}
		case recall.LifecycleRefutedContext:
			// refuted_context itself is the marker
			// (defect code is emitted only when lifecycle_role is missing or ambiguous).
			// Nothing else required here beyond lifecycle_role presence.
		}

		// Deduplicate within memory.
		memMissing = uniqueStrings(memMissing)
		memUnsafe = uniqueStrings(memUnsafe)

		for _, x := range memMissing {
			missingAgg[x] = struct{}{}
		}
		for _, x := range memUnsafe {
			unsafeAgg[x] = struct{}{}
		}

		score := 1.0
		if len(memMissing) > 0 || len(memUnsafe) > 0 {
			score = 0.0
		}
		ev.MemoryContractScores[it.ID] = score
		total += score
		count++
	}

	if len(missingAgg) > 0 || len(unsafeAgg) > 0 {
		ev.ContractPassed = false
	}
	ev.MissingRequiredFields = keysOf(missingAgg)
	ev.UnsafeOmissions = keysOf(unsafeAgg)
	if count > 0 {
		ev.BundleContractScore = total / float64(count)
	}
	return ev
}

func collectMemoryItems(b *recall.RecallBundle) []recall.MemoryItem {
	var out []recall.MemoryItem
	for _, s := range [][]recall.MemoryItem{
		b.GoverningConstraints,
		b.Decisions,
		b.KnownFailures,
		b.ApplicablePatterns,
		b.Continuity,
		b.Constraints,
		b.Experience,
	} {
		out = append(out, s...)
	}
	return out
}

func uniqueStrings(xs []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x == "" {
			continue
		}
		if _, ok := seen[x]; ok {
			continue
		}
		seen[x] = struct{}{}
		out = append(out, x)
	}
	return out
}

func keysOf(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// stable ordering isn't required for correctness but is helpful in tests.
	sortStrings(out)
	return out
}

func sortStrings(xs []string) {
	// small local sort to avoid importing sort in this file
	for i := 0; i < len(xs); i++ {
		for j := i + 1; j < len(xs); j++ {
			if xs[j] < xs[i] {
				xs[i], xs[j] = xs[j], xs[i]
			}
		}
	}
}

