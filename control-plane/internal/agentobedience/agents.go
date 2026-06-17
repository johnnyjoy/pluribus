package agentobedience

import (
	"fmt"
	"sort"

	"control-plane/internal/agentcontract"
	"control-plane/internal/recall"
	"control-plane/pkg/api"
)

func buildMemoryItem(cm CaseMemory) recall.MemoryItem {
	return recall.MemoryItem{
		ID:             cm.MemoryID,
		Kind:           "constraint",
		Statement:      cm.Statement,
		Authority:      3,
		Applicability:  api.ApplicabilityGoverning,
		Status:         cm.Status,
		LifecycleRole:  cm.LifecycleRole,
		SchemaType:     cm.SchemaType,
		Scope:          cm.Scope,
		NegativeScope:  cm.NegativeScope,
		UseInstruction: cm.UseInstruction,
		MisuseWarning:  cm.MisuseWarning,
		SourceType:     cm.SourceType,
		AuthorityBasis: cm.AuthorityBasis,
		UtilityScore:   cm.UtilityScore,
		QualityState:   cm.QualityState,
		QualityScore:   cm.QualityScore,
		SupersededBy:   cm.SupersededBy,
	}
}

// BundleFromCase builds a RecallBundle from fixture memories.
func BundleFromCase(c ObedienceCase) *recall.RecallBundle {
	b := &recall.RecallBundle{}
	for _, cm := range c.InputMemories {
		it := buildMemoryItem(cm)
		switch cm.LifecycleRole {
		case recall.LifecycleHistoricalContext, recall.LifecycleRefutedContext, recall.LifecycleSupersededContext:
			b.KnownFailures = append(b.KnownFailures, it)
		default:
			b.GoverningConstraints = append(b.GoverningConstraints, it)
		}
	}
	return b
}

func memoryFactsMap(c ObedienceCase) map[string][]string {
	out := map[string][]string{}
	for _, cm := range c.InputMemories {
		if len(cm.OutputFacts) > 0 {
			out[cm.MemoryID] = append([]string(nil), cm.OutputFacts...)
		}
	}
	return out
}

func hasBehavior(behaviors []string, b string) bool {
	for _, x := range behaviors {
		if x == b {
			return true
		}
	}
	return false
}

// RunScriptedAgent executes a deterministic scripted agent (not an LLM).
func RunScriptedAgent(c ObedienceCase, bundle *recall.RecallBundle, iface string) MemoryUseTelemetry {
	items := agentcontract.CollectMemoryItems(bundle)
	recalled := make([]string, 0, len(items))
	for _, it := range items {
		recalled = append(recalled, it.ID)
	}
	sort.Strings(recalled)

	tel := MemoryUseTelemetry{
		RunID:             fmt.Sprintf("run_%s_%s", c.TaskID, c.ID),
		SessionID:         "sess_obedience_proof",
		TaskID:            c.TaskID,
		Interface:         iface,
		AgentKind:         c.AgentMode,
		RecallRequestID:   fmt.Sprintf("recall_%s", c.TaskID),
		RecallBundleID:    fmt.Sprintf("bundle_%s", c.ID),
		OutputID:          fmt.Sprintf("out_%s", c.ID),
		RecalledMemoryIDs: recalled,
		TelemetryComplete: true,
	}
	factMap := memoryFactsMap(c)

	switch c.AgentMode {
	case AgentObedient:
		runObedient(c, items, factMap, &tel)
	case AgentSloppy, AgentBroken:
		runObedient(c, items, factMap, &tel)
		applyViolationBehaviors(c, items, factMap, &tel)
	default:
		runObedient(c, items, factMap, &tel)
	}

	if hasBehavior(c.ViolationBehaviors, BehaviorOmitTelemetry) {
		tel.TelemetryComplete = false
		tel.MemoryDecisions = nil
	}

	tel.RubricResult = evaluateRubric(tel.FinalOutput.Facts, c.ExpectedOutputFacts, c.ForbiddenOutputFacts)
	return tel
}

func runObedient(c ObedienceCase, items []recall.MemoryItem, factMap map[string][]string, tel *MemoryUseTelemetry) {
	var facts []string
	var citations []string

	for _, it := range items {
		disc := agentcontract.DecideUseDiscipline(it, c.TaskTags)
		dec := MemoryDecision{
			MemoryID: it.ID,
			Reason:   disc.Reason,
		}

		switch disc.Decision {
		case "use":
			dec.Decision = "used"
			dec.ContractFieldsCited = contractFieldsForUse(it)
			if ff, ok := factMap[it.ID]; ok {
				dec.OutputFactsSupported = append([]string(nil), ff...)
				facts = append(facts, ff...)
			}
			tel.UsedMemoryIDs = append(tel.UsedMemoryIDs, it.ID)
			citations = append(citations, it.ID)
		case "historical_only":
			dec.Decision = "historical_only"
			dec.ContractFieldsCited = []string{"lifecycle_role", "misuse_warning"}
			tel.HistoricalOnlyMemoryIDs = append(tel.HistoricalOnlyMemoryIDs, it.ID)
		default:
			dec.Decision = "ignored"
			dec.ContractFieldsCited = []string{"lifecycle_role", "scope", "use_instruction"}
			tel.IgnoredMemoryIDs = append(tel.IgnoredMemoryIDs, it.ID)
		}
		tel.MemoryDecisions = append(tel.MemoryDecisions, dec)
	}

	tel.FinalOutput = FinalOutput{
		Facts:           uniqueSorted(facts),
		Actions:         []string{"proceed_with_task"},
		MemoryCitations: uniqueSorted(citations),
	}
	sortIDs(&tel.UsedMemoryIDs)
	sortIDs(&tel.IgnoredMemoryIDs)
	sortIDs(&tel.HistoricalOnlyMemoryIDs)
}

func applyViolationBehaviors(c ObedienceCase, items []recall.MemoryItem, factMap map[string][]string, tel *MemoryUseTelemetry) {
	byID := map[string]recall.MemoryItem{}
	for _, it := range items {
		byID[it.ID] = it
	}

	if hasBehavior(c.ViolationBehaviors, BehaviorHistoricalAsCurrent) {
		for _, it := range items {
			if it.LifecycleRole == recall.LifecycleHistoricalContext {
				forceUse(tel, it.ID, factMap[it.ID], []string{"historical_as_current"})
			}
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorIgnoreNegativeScope) {
		for _, it := range items {
			for _, ns := range it.NegativeScope {
				for _, tt := range c.TaskTags {
					if ns == tt {
						forceUse(tel, it.ID, factMap[it.ID], []string{"negative_scope_ignored"})
					}
				}
			}
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorUseWrongScope) {
		for _, it := range items {
			if it.Scope != "" && !tagMatch(c.TaskTags, it.Scope) {
				forceUse(tel, it.ID, factMap[it.ID], []string{"wrong_scope"})
			}
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorIgnoreRequiredConstraint) && c.RequiredConstraintMemoryID != "" {
		removeFromUsed(tel, c.RequiredConstraintMemoryID)
		tel.IgnoredMemoryIDs = append(tel.IgnoredMemoryIDs, c.RequiredConstraintMemoryID)
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorUsePreferenceOutsideScope) {
		for _, it := range items {
			if it.SchemaType == "preference" && !tagMatch(c.TaskTags, it.Scope) {
				forceUse(tel, it.ID, factMap[it.ID], []string{"preference_outside_scope"})
			}
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorUseRefuted) {
		for _, it := range items {
			if it.LifecycleRole == recall.LifecycleRefutedContext {
				forceUse(tel, it.ID, factMap[it.ID], []string{"refuted_used"})
			}
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorUseSuperseded) {
		for _, it := range items {
			if it.LifecycleRole == recall.LifecycleSupersededContext {
				forceUse(tel, it.ID, factMap[it.ID], []string{"superseded_used"})
			}
		}
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorUseUnrecalled) {
		tel.UsedMemoryIDs = append(tel.UsedMemoryIDs, "mem_unrecalled_fake")
		tel.FinalOutput.Facts = append(tel.FinalOutput.Facts, "fact_from_unrecalled_memory")
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorCiteUnrecalled) {
		tel.FinalOutput.MemoryCitations = append(tel.FinalOutput.MemoryCitations, "mem_unrecalled_fake")
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorInventUnsupportedOutput) {
		tel.FinalOutput.Facts = append(tel.FinalOutput.Facts, "invented_unsupported_fact_xyz")
	}
	if hasBehavior(c.ViolationBehaviors, BehaviorClaimHelpWithoutImprovement) {
		tel.FinalOutput.Facts = append(tel.FinalOutput.Facts, "memory_helped_but_no_real_facts")
	}

	_ = byID
	tel.FinalOutput.Facts = uniqueSorted(tel.FinalOutput.Facts)
	tel.FinalOutput.MemoryCitations = uniqueSorted(tel.FinalOutput.MemoryCitations)
	sortIDs(&tel.UsedMemoryIDs)
	sortIDs(&tel.IgnoredMemoryIDs)
}

func forceUse(tel *MemoryUseTelemetry, id string, facts []string, violation []string) {
	removeFromIgnored(tel, id)
	removeFromHistorical(tel, id)
	if !contains(tel.UsedMemoryIDs, id) {
		tel.UsedMemoryIDs = append(tel.UsedMemoryIDs, id)
	}
	tel.FinalOutput.Facts = append(tel.FinalOutput.Facts, facts...)
	if !contains(tel.FinalOutput.MemoryCitations, id) {
		tel.FinalOutput.MemoryCitations = append(tel.FinalOutput.MemoryCitations, id)
	}
	for i := range tel.MemoryDecisions {
		if tel.MemoryDecisions[i].MemoryID == id {
			tel.MemoryDecisions[i].Decision = "misused"
			tel.MemoryDecisions[i].ViolationCodes = violation
		}
	}
	if !contains(tel.MisusedMemoryIDs, id) {
		tel.MisusedMemoryIDs = append(tel.MisusedMemoryIDs, id)
	}
}

func contractFieldsForUse(it recall.MemoryItem) []string {
	fields := []string{"lifecycle_role", "scope", "use_instruction", "quality_state"}
	if it.SchemaType != "" {
		fields = append(fields, "schema_type")
	}
	if len(it.NegativeScope) > 0 {
		fields = append(fields, "negative_scope")
	}
	return fields
}

// EvaluateRubric checks output facts against expected/forbidden lists deterministically.
func EvaluateRubric(facts, expected, forbidden []string) RubricResult {
	return evaluateRubric(facts, expected, forbidden)
}

func evaluateRubric(facts, expected, forbidden []string) RubricResult {
	r := RubricResult{Passed: true}
	set := map[string]struct{}{}
	for _, f := range facts {
		set[f] = struct{}{}
	}
	for _, e := range expected {
		if _, ok := set[e]; ok {
			r.ExpectedFactsPresent = append(r.ExpectedFactsPresent, e)
		} else {
			r.Passed = false
			r.Violations = append(r.Violations, ViolationUnsupportedOutputClaim+":missing:"+e)
		}
	}
	for _, f := range forbidden {
		if _, ok := set[f]; ok {
			r.Passed = false
			r.Violations = append(r.Violations, ViolationUnsupportedOutputClaim+":forbidden:"+f)
			r.ForbiddenFactsAbsent = append(r.ForbiddenFactsAbsent, f)
		}
	}
	return r
}

func tagMatch(tags []string, scope string) bool {
	for _, t := range tags {
		if t == scope {
			return true
		}
	}
	return false
}

func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}

func removeFromUsed(tel *MemoryUseTelemetry, id string) {
	tel.UsedMemoryIDs = filterOut(tel.UsedMemoryIDs, id)
}

func removeFromIgnored(tel *MemoryUseTelemetry, id string) {
	tel.IgnoredMemoryIDs = filterOut(tel.IgnoredMemoryIDs, id)
}

func removeFromHistorical(tel *MemoryUseTelemetry, id string) {
	tel.HistoricalOnlyMemoryIDs = filterOut(tel.HistoricalOnlyMemoryIDs, id)
}

func filterOut(xs []string, id string) []string {
	out := make([]string, 0, len(xs))
	for _, x := range xs {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

func uniqueSorted(in []string) []string {
	set := map[string]struct{}{}
	for _, s := range in {
		if s != "" {
			set[s] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func sortIDs(xs *[]string) {
	sort.Strings(*xs)
}
