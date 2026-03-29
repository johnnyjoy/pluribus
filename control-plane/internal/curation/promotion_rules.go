package curation

import (
	"strings"

	"control-plane/pkg/api"
)

// hasImperativeGuardrailLanguage detects short imperative / prohibition phrasing (deterministic; no ML).
func hasImperativeGuardrailLanguage(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	ls := strings.ToLower(strings.TrimSpace(s))
	if strings.Contains(ls, "forbidden") {
		return true
	}
	t := " " + ls + " "
	return strings.Contains(t, " never ") ||
		strings.Contains(t, "must not") ||
		strings.Contains(t, "mustn't") ||
		strings.Contains(t, " do not ") ||
		strings.Contains(t, " don't ") ||
		strings.Contains(t, " always ") ||
		strings.Contains(t, " shall not ") ||
		strings.Contains(t, " will not ")
}

// hasSeveritySignal flags operational-risk language suitable for one-shot constraint promotion.
func hasSeveritySignal(s string) bool {
	if strings.TrimSpace(s) == "" {
		return false
	}
	ld := strings.ToLower(s)
	return strings.Contains(ld, "production") ||
		strings.Contains(ld, "release") ||
		strings.Contains(ld, "corruption") ||
		strings.Contains(ld, "data loss") ||
		strings.Contains(ld, "duplicate charge") ||
		strings.Contains(ld, "outage") ||
		strings.Contains(ld, "p0") ||
		strings.Contains(ld, "p1") ||
		strings.Contains(ld, "security breach") ||
		strings.Contains(ld, "incident")
}

// shapeConstraintStatement normalizes text toward a short rule-shaped statement.
func shapeConstraintStatement(source string) string {
	s := strings.TrimSpace(source)
	if s == "" {
		return s
	}
	ls := strings.ToLower(s)
	if strings.HasPrefix(ls, "never ") || strings.HasPrefix(ls, "do not ") || strings.HasPrefix(ls, "don't ") ||
		strings.HasPrefix(ls, "must not ") || strings.HasPrefix(ls, "always ") {
		return s
	}
	if strings.Contains(ls, "never ") {
		idx := strings.Index(ls, "never ")
		return strings.TrimSpace(s[idx:])
	}
	return s
}

// constraintFromDecision derives a forbidding constraint from a narrow set of decision templates.
func constraintFromDecision(decision string) (string, bool) {
	d := strings.TrimSpace(decision)
	if d == "" {
		return "", false
	}
	ld := strings.ToLower(d)
	pg := strings.Contains(ld, "postgres") || strings.Contains(ld, "postgresql")
	if !pg {
		return "", false
	}
	if !(strings.Contains(ld, "durable") || strings.Contains(ld, "store") || strings.Contains(ld, "storage")) {
		return "", false
	}
	if !(strings.Contains(ld, "must") || strings.Contains(ld, "only") || strings.Contains(ld, "required") || strings.Contains(ld, "shall")) {
		return "", false
	}
	return "Do not use SQLite for durable storage.", true
}

// draftPriority returns sort order for truncation (lower = keep first).
func draftPriority(k api.MemoryKind) int {
	switch k {
	case api.MemoryKindConstraint:
		return 0
	case api.MemoryKindDecision:
		return 1
	case api.MemoryKindFailure:
		return 2
	case api.MemoryKindPattern:
		return 3
	case api.MemoryKindState:
		return 4
	default:
		return 5
	}
}
