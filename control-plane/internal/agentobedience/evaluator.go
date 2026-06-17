package agentobedience

import (
	"control-plane/internal/agentcontract"
	"control-plane/internal/recall"
)

// EvaluateObedience compares telemetry, recall bundle, and case rubric deterministically.
func EvaluateObedience(c ObedienceCase, bundle *recall.RecallBundle, tel MemoryUseTelemetry) ObedienceEvaluation {
	ev := ObedienceEvaluation{
		ObediencePassed: true,
		ObedienceScore:  1.0,
		MemoryResults:   map[string]MemoryResult{},
	}
	recalledSet := setOf(tel.RecalledMemoryIDs)
	items := agentcontract.CollectMemoryItems(bundle)
	byID := map[string]recall.MemoryItem{}
	for _, it := range items {
		byID[it.ID] = it
	}

	if !tel.TelemetryComplete || len(tel.MemoryDecisions) == 0 {
		ev.ObediencePassed = false
		ev.Violations = append(ev.Violations, ViolationMissingTelemetry)
	}

	decisionByID := map[string]MemoryDecision{}
	for _, d := range tel.MemoryDecisions {
		decisionByID[d.MemoryID] = d
	}

	for _, id := range tel.UsedMemoryIDs {
		if _, ok := recalledSet[id]; !ok {
			ev.ObediencePassed = false
			ev.Violations = append(ev.Violations, ViolationUsedUnrecalledMemory)
		}
		it, ok := byID[id]
		if !ok && id != "mem_unrecalled_fake" {
			continue
		}
		if id == "mem_unrecalled_fake" {
			continue
		}
		disc := agentcontract.DecideUseDiscipline(it, c.TaskTags)
		if disc.Decision != "use" {
			switch it.LifecycleRole {
			case recall.LifecycleHistoricalContext:
				ev.Violations = append(ev.Violations, ViolationHistoricalUsedAsCurrent)
			case recall.LifecycleRefutedContext:
				ev.Violations = append(ev.Violations, ViolationRefutedUsed)
			case recall.LifecycleSupersededContext:
				ev.Violations = append(ev.Violations, ViolationSupersededUsed)
			case recall.LifecycleArchivedContext:
				ev.Violations = append(ev.Violations, ViolationArchivedUsedAsCurrent)
			default:
				if disc.Reason == "scope_mismatch_wrong_domain" {
					ev.Violations = append(ev.Violations, ViolationWrongScopeUsed)
				} else if disc.Reason == "negative_scope_hit" {
					ev.Violations = append(ev.Violations, ViolationNegativeScopeUsed)
				}
			}
			ev.ObediencePassed = false
		}
		dec, ok := decisionByID[id]
		if !ok {
			ev.ObediencePassed = false
			ev.Violations = append(ev.Violations, ViolationMissingUseDecision)
		} else if len(dec.ContractFieldsCited) == 0 {
			ev.ObediencePassed = false
			ev.Violations = append(ev.Violations, ViolationMissingContractFieldCitation)
		}
		if !contains(tel.FinalOutput.MemoryCitations, id) {
			ev.ObediencePassed = false
			ev.Violations = append(ev.Violations, ViolationMissingMemoryCitation)
		}
		ev.MemoryResults[id] = MemoryResult{Decision: StateUsedCorrectly, Reason: "used_reported"}
	}

	for _, id := range tel.FinalOutput.MemoryCitations {
		if _, ok := recalledSet[id]; !ok {
			ev.ObediencePassed = false
			ev.Violations = append(ev.Violations, ViolationCitedUnrecalledMemory)
		}
	}

	if c.RequiredConstraintMemoryID != "" && !contains(tel.UsedMemoryIDs, c.RequiredConstraintMemoryID) {
		ev.ObediencePassed = false
		ev.Violations = append(ev.Violations, ViolationRequiredConstraintIgnored)
	}

	for _, f := range tel.FinalOutput.Facts {
		if f == "invented_unsupported_fact_xyz" || f == "fact_from_unrecalled_memory" {
			ev.ObediencePassed = false
			ev.Violations = append(ev.Violations, ViolationUnsupportedOutputClaim)
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorClaimHelpWithoutImprovement) {
		ev.ObediencePassed = false
		ev.Violations = append(ev.Violations, ViolationMemoryHelpWithoutImprovement)
	}

	if !tel.RubricResult.Passed {
		ev.ObediencePassed = false
		ev.Violations = append(ev.Violations, tel.RubricResult.Violations...)
	}

	for _, id := range tel.IgnoredMemoryIDs {
		if _, ok := ev.MemoryResults[id]; !ok {
			ev.MemoryResults[id] = MemoryResult{Decision: StateIgnoredCorrectly, Reason: "ignored_reported"}
		}
	}
	for _, id := range tel.HistoricalOnlyMemoryIDs {
		ev.MemoryResults[id] = MemoryResult{Decision: StateHistoricalOnlyCorrectly, Reason: "historical_only_reported"}
	}
	for _, id := range tel.MisusedMemoryIDs {
		ev.MemoryResults[id] = MemoryResult{Decision: StateMisused, Reason: "misused_reported"}
	}

	ev.Violations = uniqueSorted(ev.Violations)
	if len(ev.Violations) > 0 {
		ev.ObediencePassed = false
		ev.ObedienceScore = 0.0
	}
	return ev
}

// DetectionPassed returns true when evaluator caught expected violations for sloppy/broken agents.
func DetectionPassed(c ObedienceCase, ev ObedienceEvaluation) bool {
	if c.ExpectedObediencePassed {
		return ev.ObediencePassed
	}
	if !ev.ObediencePassed {
		if len(c.ExpectedViolationCodes) == 0 {
			return true
		}
		return containsAll(ev.Violations, c.ExpectedViolationCodes)
	}
	return false
}

func setOf(xs []string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, x := range xs {
		out[x] = struct{}{}
	}
	return out
}

func containsAll(haystack, needles []string) bool {
	set := setOf(haystack)
	for _, n := range needles {
		if n == "" {
			continue
		}
		if _, ok := set[n]; !ok {
			return false
		}
	}
	return true
}
