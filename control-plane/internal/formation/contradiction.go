package formation

import (
	"regexp"
	"strings"
)

var notAfterMust = regexp.MustCompile(`(?i)\bmust\s+not\b`)

// ContradictsStatement returns true when newStmt is a likely negation flip of existingStmt.
// Lightweight heuristic for Phase 5 — not full semantic contradiction.
func ContradictsStatement(newStmt, existingStmt string) bool {
	a := normalizeContradictionText(newStmt)
	b := normalizeContradictionText(existingStmt)
	if a == "" || b == "" || a == b {
		return false
	}
	if isNegationPair(a, b) {
		return true
	}
	// High overlap with must/must not flip
	if tokenJaccard(a, b) >= 0.65 {
		aNot := notAfterMust.MatchString(a)
		bNot := notAfterMust.MatchString(b)
		if aNot != bNot {
			return true
		}
	}
	return false
}

func normalizeContradictionText(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.Join(strings.Fields(s), " ")
	return s
}

func isNegationPair(a, b string) bool {
	if strings.Contains(a, " must not ") && strings.Contains(b, " must ") && !strings.Contains(b, " must not ") {
		return stripNot(a) == stripNot(b)
	}
	if strings.Contains(b, " must not ") && strings.Contains(a, " must ") && !strings.Contains(a, " must not ") {
		return stripNot(a) == stripNot(b)
	}
	return false
}

func stripNot(s string) string {
	s = strings.ReplaceAll(s, " must not ", " must ")
	s = strings.ReplaceAll(s, "must not ", "must ")
	return s
}

func tokenJaccard(a, b string) float64 {
	ta := tokenSet(a)
	tb := tokenSet(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	inter := 0
	for k := range ta {
		if tb[k] {
			inter++
		}
	}
	union := len(ta) + len(tb) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func tokenSet(s string) map[string]bool {
	out := make(map[string]bool)
	for _, w := range strings.Fields(s) {
		w = strings.Trim(w, ".,;:!?\"'")
		if len(w) > 1 {
			out[w] = true
		}
	}
	return out
}
