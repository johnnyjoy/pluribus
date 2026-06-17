package formationquality

import (
	"strings"
)

// Evaluator applies deterministic formation quality rules.
type Evaluator struct{}

// NewEvaluator returns a default evaluator.
func NewEvaluator() *Evaluator {
	return &Evaluator{}
}

// Evaluate runs schema-specific quality checks.
func (e *Evaluator) Evaluate(in Input) Result {
	res := Result{
		SchemaType:             inferSchema(in),
		SuggestedStatus:        "active",
		SuggestedApplicability: strings.TrimSpace(in.Applicability),
		SafeForActiveRecall:    true,
	}
	if res.SuggestedApplicability == "" {
		res.SuggestedApplicability = "advisory"
	}

	if isVagueStatement(in.Statement) {
		res.Defects = append(res.Defects, Defect{
			Code: DefectVagueStatement, Severity: SeverityHard,
			Message: "Statement is too vague for durable memory.",
		})
	}

	if strings.TrimSpace(in.SchemaType) == "" && IsDirectLikePath(in.Path) {
		res.Warnings = append(res.Warnings, Warning{
			Code: DefectMissingSchemaType, Severity: SeveritySoft,
			Message: "schema_type not explicit; inferred from kind/statement.",
		})
	} else if strings.TrimSpace(in.SchemaType) != "" && !knownSchema(SchemaType(in.SchemaType)) {
		res.Defects = append(res.Defects, Defect{
			Code: DefectUnknownSchemaType, Severity: SeverityHard,
			Message: "Unknown schema_type.",
		})
	}

	evalScope(in, &res)
	evalCues(in, &res)
	evalProvenance(in, &res)
	evalSchemaRules(in, res.SchemaType, &res)
	evalLifecycle(in, res.SchemaType, &res)
	evalUseInstructions(in, res.SchemaType, &res)

	res.QualityScore = scoreResult(res)
	res.Decision = decide(res, in)
	res.Passed = res.Decision == DecisionAcceptActive || res.Decision == DecisionAcceptPending
	res.SafeForActiveRecall = res.Decision == DecisionAcceptActive && wantsActive(in)

	if res.Decision == DecisionAcceptPending || res.Decision == DecisionNeedsCuration {
		res.SuggestedStatus = "pending"
		res.SafeForActiveRecall = false
	}
	if res.Decision == DecisionRejectGarbage || res.Decision == DecisionRejectDangerous {
		res.SuggestedStatus = "rejected"
		res.SafeForActiveRecall = false
		res.Passed = false
	}
	return res
}

func evalScope(in Input, res *Result) {
	hasScope := strings.TrimSpace(in.Scope) != "" || hasScopeTag(in.Tags)
	universal := isUniversalRule(in.Statement)

	if universal && !hasScope && len(in.NegativeScope) == 0 {
		sev := SeveritySoft
		if in.Applicability == "governing" || in.Kind == "constraint" {
			sev = SeverityHard
		}
		appendDefect(res, DefectOvergeneralizedStatement, sev,
			"Universal always/never rule requires scope or exception.")
	}

	switch inferSchema(in) {
	case SchemaConstraint, SchemaCurrentGuidance:
		if !hasScope && universal {
			res.Defects = append(res.Defects, Defect{
				Code: DefectConstraintWithoutScope, Severity: SeverityHard,
				Message: "Constraint requires scope or explicit exception.",
			})
		}
	case SchemaPreference:
		if !hasScope {
			res.Defects = append(res.Defects, Defect{
				Code: DefectMissingScope, Severity: SeverityHard,
				Message: "Preference requires owner/scope.",
			})
		}
		if in.Applicability == "governing" || in.Authority >= 8 {
			res.Defects = append(res.Defects, Defect{
				Code: DefectPreferenceAsRuleRisk, Severity: SeverityHard,
				Message: "Preference cannot be governing or high authority without curation.",
			})
		}
	}

	if inferSchema(in) == SchemaConstraint && len(in.NegativeScope) == 0 && universal {
		res.Warnings = append(res.Warnings, Warning{
			Code: DefectMissingNegativeScope, Severity: SeveritySoft,
			Message: "No negative scope provided; memory may overgeneralize.",
		})
	}
}

func hasScopeTag(tags []string) bool {
	for _, t := range tags {
		lower := strings.ToLower(t)
		if strings.HasPrefix(lower, "scope:") || strings.HasPrefix(lower, "project:") ||
			strings.HasPrefix(lower, "repo:") || strings.HasPrefix(lower, "domain:") {
			return true
		}
	}
	return false
}

func evalCues(in Input, res *Result) {
	_, misleading, underCued := EvaluateCues(in)
	if misleading {
		res.Defects = append(res.Defects, Defect{
			Code: DefectMisleadingCues, Severity: SeverityHard,
			Message: "Retrieval cues are generic or misleading.",
		})
	}
	if underCued && wantsActive(in) && requiresRichEncoding(in) {
		if IsDirectLikePath(in.Path) {
			appendDefect(res, DefectMissingRetrievalCues, SeverityHard,
				"Active high-signal memory requires retrieval cues.")
		} else {
			res.Warnings = append(res.Warnings, Warning{
				Code: DefectMissingRetrievalCues, Severity: SeveritySoft,
				Message: "Under-cued memory should be pending or curated.",
			})
		}
	}
}

func requiresRichEncoding(in Input) bool {
	if in.Applicability == "governing" || in.Authority >= 6 {
		return true
	}
	switch inferSchema(in) {
	case SchemaConstraint, SchemaCurrentGuidance, SchemaProcedure:
		return true
	default:
		return false
	}
}

func appendDefect(res *Result, code DefectCode, sev Severity, msg string) {
	res.Defects = append(res.Defects, Defect{Code: code, Severity: sev, Message: msg})
}

func evalProvenance(in Input, res *Result) {
	highRisk := in.Applicability == "governing" || in.Kind == "constraint" || in.Authority >= 8
	if in.SourceType == "agent_inferred" && in.Authority >= 6 {
		res.Defects = append(res.Defects, Defect{
			Code: DefectAgentInferredHighAuthority, Severity: SeverityHard,
			Message: "Agent-inferred memory cannot be high authority.",
		})
	}
	if highRisk && wantsActive(in) && in.ProvenanceFields < 2 {
		res.Defects = append(res.Defects, Defect{
			Code: DefectMissingProvenance, Severity: SeverityHard,
			Message: "High-risk active memory requires provenance.",
		})
	}
	if highRisk && strings.TrimSpace(in.AuthorityBasis) == "" && wantsActive(in) {
		res.Defects = append(res.Defects, Defect{
			Code: DefectMissingAuthorityBasis, Severity: SeverityHard,
			Message: "High-risk memory requires authority_basis.",
		})
	}
}

func evalSchemaRules(in Input, schema SchemaType, res *Result) {
	stmt := in.Statement
	switch schema {
	case SchemaDecision:
		if !hasDecisionReason(in) {
			res.Defects = append(res.Defects, Defect{
				Code: DefectDecisionWithoutReason, Severity: SeverityHard,
				Message: "Decision requires reason or rationale.",
			})
		}
	case SchemaFailureWarning, SchemaLesson:
		if !hasFailureCause(stmt) {
			res.Defects = append(res.Defects, Defect{
				Code: DefectFailureWithoutCause, Severity: SeverityHard,
				Message: "Failure/warning requires cause or condition.",
			})
		}
	case SchemaProcedure:
		if !hasProcedureSteps(stmt) {
			res.Defects = append(res.Defects, Defect{
				Code: DefectProcedureWithoutSteps, Severity: SeverityHard,
				Message: "Procedure requires ordered steps.",
			})
		}
	case SchemaHistoricalEvent:
		if !hasTemporalBasis(in) {
			res.Defects = append(res.Defects, Defect{
				Code: DefectMissingTemporalBasis, Severity: SeverityHard,
				Message: "Historical event requires temporal basis.",
			})
		}
	}
}

func evalLifecycle(in Input, schema SchemaType, res *Result) {
	role := strings.TrimSpace(in.LifecycleRole)
	if schema == SchemaRefutedGuidance && wantsActive(in) {
		res.Defects = append(res.Defects, Defect{
			Code: DefectRefutedAsActive, Severity: SeverityHard,
			Message: "Refuted guidance cannot be active.",
		})
	}
	if schema == SchemaSupersededGuidance && wantsActive(in) {
		res.Defects = append(res.Defects, Defect{
			Code: DefectSupersededAsActive, Severity: SeverityHard,
			Message: "Superseded guidance cannot be active current guidance.",
		})
	}
	if schema == SchemaHistoricalEvent && wantsActive(in) && role != "historical_context" {
		res.Defects = append(res.Defects, Defect{
			Code: DefectHistoricalAsCurrentRisk, Severity: SeverityHard,
			Message: "Historical event must not be active current guidance.",
		})
	}
	if schema == SchemaHistoricalEvent && wantsActive(in) && role == "" {
		res.Defects = append(res.Defects, Defect{
			Code: DefectMissingLifecycleRole, Severity: SeverityHard,
			Message: "Historical memory requires lifecycle_role.",
		})
	}
	if in.Applicability == "governing" && IsDirectLikePath(in.Path) && in.Authority >= 8 && wantsActive(in) && in.ProvenanceFields < 2 {
		res.Defects = append(res.Defects, Defect{
			Code: DefectUnsafeDirectGoverning, Severity: SeverityHard,
			Message: "Unsafe direct governing memory without provenance.",
		})
	}
}

func evalUseInstructions(in Input, schema SchemaType, res *Result) {
	needs := (schema == SchemaCurrentGuidance || schema == SchemaConstraint) &&
		(in.Applicability == "governing" || in.Authority >= 6)
	if schema == SchemaPreference || schema == SchemaHistoricalEvent {
		needs = wantsActive(in)
	}
	if needs && strings.TrimSpace(in.UseInstruction) == "" && wantsActive(in) {
		res.Defects = append(res.Defects, Defect{
			Code: DefectMissingUseInstruction, Severity: SeverityHard,
			Message: "Guidance memory requires use_instruction.",
		})
	}
	if schema == SchemaHistoricalEvent && strings.TrimSpace(in.MisuseWarning) == "" {
		res.Warnings = append(res.Warnings, Warning{
			Code: DefectHistoricalAsCurrentRisk, Severity: SeveritySoft,
			Message: "Historical memory should include misuse_warning.",
		})
	}
}

// HasHardDefects reports whether formation quality found blocking hard defects.
func HasHardDefects(r Result) bool {
	for _, d := range r.Defects {
		if d.Severity == SeverityHard {
			return true
		}
	}
	return false
}

func wantsActive(in Input) bool {
	s := strings.ToLower(strings.TrimSpace(in.Status))
	return s == "" || s == "active"
}

func decide(res Result, in Input) Decision {
	for _, d := range res.Defects {
		if d.Severity != SeverityHard {
			continue
		}
		switch d.Code {
		case DefectRefutedAsActive, DefectSupersededAsActive:
			return DecisionRejectDangerous
		}
	}
	for _, d := range res.Defects {
		if d.Severity != SeverityHard {
			continue
		}
		switch d.Code {
		case DefectVagueStatement, DefectMisleadingCues:
			return DecisionRejectGarbage
		case DefectRefutedAsActive, DefectSupersededAsActive:
			return DecisionRejectDangerous
		default:
			if wantsActive(in) {
				return DecisionNeedsCuration
			}
			return DecisionAcceptPending
		}
	}
	if len(res.Warnings) > 0 && IsDirectLikePath(in.Path) && wantsActive(in) {
		return DecisionAcceptPending
	}
	return DecisionAcceptActive
}

func scoreResult(res Result) float64 {
	score := 1.0
	for _, d := range res.Defects {
		switch d.Severity {
		case SeverityHard:
			score -= 0.35
		case SeveritySoft:
			score -= 0.15
		}
	}
	for range res.Warnings {
		score -= 0.05
	}
	if score < 0 {
		return 0
	}
	if score > 1 {
		return 1
	}
	return score
}
