package agentcontract

import "control-plane/internal/recall"

// DecideUseDiscipline applies deterministic Phase 11F "memory use discipline" rules
// to a recalled MemoryItem.
//
// This is not a judge. It is a ruleset that explains when the contract indicates
// a memory is safe (or unsafe) to guide current action.
func DecideUseDiscipline(it recall.MemoryItem, taskTags []string) UseDisciplineDecision {
	// Refuted/superseded/archived are never "current guidance".
	switch it.LifecycleRole {
	case recall.LifecycleHistoricalContext:
		return UseDisciplineDecision{Decision: "historical_only", Reason: "historical_context_not_current_guidance"}
	case recall.LifecycleSupersededContext:
		return UseDisciplineDecision{Decision: "ignore", Reason: "superseded_guidance_not_current"}
	case recall.LifecycleArchivedContext:
		return UseDisciplineDecision{Decision: "ignore", Reason: "archived_guidance_not_current"}
	case recall.LifecycleRefutedContext:
		return UseDisciplineDecision{Decision: "ignore", Reason: "refuted_guidance_not_current"}
	case recall.LifecycleCurrentGuidance:
		// fallthrough to current guidance checks
	default:
		return UseDisciplineDecision{Decision: "ignore", Reason: "unknown_lifecycle_role"}
	}

	// Current guidance must be explicitly safe for active recall.
	if it.QualityState != "accept_active" {
		return UseDisciplineDecision{Decision: "ignore", Reason: "quality_state_not_accept_active"}
	}

	// Quality defects/missing instructions must suppress guidance.
	if it.UseInstruction == "" {
		return UseDisciplineDecision{Decision: "ignore", Reason: "missing_use_instruction"}
	}

	// Missing negative scope is treated as contract-unsafe for guidance (universal rule).
	if len(it.NegativeScope) == 0 {
		return UseDisciplineDecision{Decision: "ignore", Reason: "missing_negative_scope_for_universal_rule"}
	}

	// Negative scope suppresses use.
	for _, ns := range it.NegativeScope {
		for _, tt := range taskTags {
			if ns == tt {
				return UseDisciplineDecision{Decision: "ignore", Reason: "negative_scope_hit"}
			}
		}
	}

	// Scope must match for guidance.
	if it.Scope == "" {
		return UseDisciplineDecision{Decision: "unsafe", Reason: "missing_scope_for_guidance"}
	}
	scopeMatch := false
	for _, tt := range taskTags {
		if tt == it.Scope {
			scopeMatch = true
			break
		}
	}
	if !scopeMatch {
		return UseDisciplineDecision{Decision: "ignore", Reason: "scope_mismatch_wrong_domain"}
	}

	return UseDisciplineDecision{Decision: "use", Reason: "contract_fields_present_and_safe"}
}

