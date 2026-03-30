package distillation

import (
	"strings"

	"control-plane/pkg/api"
)

// draft is one proposed candidate kind + rule tag for the reason field.
type draft struct {
	kind   api.MemoryKind
	reason string
}

var (
	constraintHints = []string{"must not", "never ", "never.", "shall not", "prohibited", "required that", "do not ", "don't ", "cannot "}
	failureHints    = []string{"error", "rollback", "rolled back", "failed", "rejection", "rejected", "timeout", "outage", "exception", "blocked", "incident"}
	decisionHints   = []string{" chose ", " decided ", " picked ", " selected ", " instead of ", " went with ", " opted for "}
	patternHints    = []string{"worked", "this worked", " best practice", " pattern", "always use", "always "}
)

func extractDrafts(lower string) []draft {
	var out []draft
	if containsAny(lower, constraintHints) {
		out = append(out, draft{kind: api.MemoryKindConstraint, reason: "distilled:constraint_keywords"})
	}
	if containsAny(lower, failureHints) {
		out = append(out, draft{kind: api.MemoryKindFailure, reason: "distilled:failure_keywords"})
	}
	if containsAny(lower, decisionHints) {
		out = append(out, draft{kind: api.MemoryKindDecision, reason: "distilled:decision_keywords"})
	}
	if containsAny(lower, patternHints) {
		out = append(out, draft{kind: api.MemoryKindPattern, reason: "distilled:pattern_keywords"})
	}
	return out
}

func containsAny(s string, hints []string) bool {
	for _, h := range hints {
		if strings.Contains(s, strings.TrimSpace(h)) {
			return true
		}
	}
	return false
}

func normalizeLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func clipStatement(s string, max int) string {
	s = strings.TrimSpace(s)
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}
