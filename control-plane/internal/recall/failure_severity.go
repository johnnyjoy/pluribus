package recall

import (
	"strings"
)

// High-impact keywords for failure statements (documented heuristic v1). Scorer-only; does not mutate DB authority.
var failureSeverityKeywords = []string{
	"production", "data loss", "outage", "corruption", "financial",
	"breach", "pii", "security", "catastrophic", "customer data",
}

// FailureSeverityScore returns a bounded [0,1] score from statement keywords (no ML).
func FailureSeverityScore(statement string) float64 {
	s := strings.ToLower(strings.TrimSpace(statement))
	if s == "" {
		return 0
	}
	var hit float64
	for _, kw := range failureSeverityKeywords {
		if strings.Contains(s, kw) {
			hit += 0.12
		}
	}
	if hit > 1 {
		return 1
	}
	return hit
}
