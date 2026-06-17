package formationquality

import "strings"

func knownSchema(s SchemaType) bool {
	switch s {
	case SchemaConstraint, SchemaDecision, SchemaLesson, SchemaFailureWarning,
		SchemaPreference, SchemaFact, SchemaHistoricalEvent, SchemaProcedure,
		SchemaCurrentGuidance, SchemaSupersededGuidance, SchemaRefutedGuidance:
		return true
	default:
		return false
	}
}

func inferSchema(in Input) SchemaType {
	if s := SchemaType(strings.TrimSpace(in.SchemaType)); s != "" {
		return s
	}
	switch strings.ToLower(strings.TrimSpace(in.Kind)) {
	case "constraint":
		return SchemaConstraint
	case "decision":
		return SchemaDecision
	case "failure":
		return SchemaFailureWarning
	case "pattern":
		lower := strings.ToLower(in.Statement)
		if strings.Contains(lower, "preference") || strings.Contains(lower, "prefer ") {
			return SchemaPreference
		}
		if strings.Contains(lower, "procedure") || strings.Contains(lower, "step ") {
			return SchemaProcedure
		}
		if strings.Contains(lower, "warning") {
			return SchemaFailureWarning
		}
		return SchemaFact
	default:
		return SchemaFact
	}
}

func isUniversalRule(stmt string) bool {
	lower := strings.ToLower(stmt)
	return strings.Contains(lower, " always ") || strings.HasPrefix(lower, "always ") ||
		strings.Contains(lower, " never ") || strings.HasPrefix(lower, "never ") ||
		strings.Contains(lower, "all agents") || strings.Contains(lower, "all future")
}

func hasTemporalBasis(in Input) bool {
	if strings.TrimSpace(in.OccurredAt) != "" {
		return true
	}
	lower := strings.ToLower(in.Statement)
	return strings.Contains(lower, "202") || strings.Contains(lower, "incident") ||
		strings.Contains(lower, "on ") && strings.Contains(lower, "deploy")
}

func hasFailureCause(stmt string) bool {
	lower := strings.ToLower(stmt)
	return strings.Contains(lower, "because") || strings.Contains(lower, "caused") ||
		strings.Contains(lower, "failed") || strings.Contains(lower, "broke") ||
		strings.Contains(lower, "error")
}

func hasDecisionReason(in Input) bool {
	if strings.TrimSpace(in.Reason) != "" {
		return true
	}
	lower := strings.ToLower(in.Statement)
	return strings.Contains(lower, "because") || strings.HasPrefix(lower, "decision:") ||
		strings.Contains(lower, " chose ") || strings.Contains(lower, "selected ")
}

func hasProcedureSteps(stmt string) bool {
	lower := strings.ToLower(stmt)
	return strings.Contains(lower, "step ") || strings.Contains(lower, "first ") ||
		strings.Contains(lower, "then ") || strings.Contains(lower, "order ")
}

func isVagueStatement(stmt string) bool {
	s := strings.TrimSpace(strings.ToLower(stmt))
	if len(s) < 12 {
		return true
	}
	vague := []string{"made progress", "worked on", "important thing", "something", "stuff", "misc", "general note"}
	for _, v := range vague {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
}
