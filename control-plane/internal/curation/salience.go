package curation

import (
	"strings"
)

// ScoreText computes a salience score (0–1) for raw text using simple heuristics:
// directives (always, never, must), canonical phrasing, failure signals, speculation.
// Used to decide whether to create a candidate and set should_review/should_promote.
func ScoreText(text string, _ *SalienceConfig) float64 {
	if text == "" {
		return 0
	}
	lower := strings.ToLower(text)
	score := 0.0
	// Directives / canonical: strong signals
	for _, s := range []string{"always", "never", "must", "shall", "do not", "cannot"} {
		if strings.Contains(lower, s) {
			score += 0.2
			break
		}
	}
	// Decision-like
	for _, s := range []string{"we decided", "we chose", "agreed", "canonical", "authority"} {
		if strings.Contains(lower, s) {
			score += 0.15
			break
		}
	}
	// Failure / lesson
	for _, s := range []string{"failed", "broke", "regression", "lesson", "avoid"} {
		if strings.Contains(lower, s) {
			score += 0.15
			break
		}
	}
	// Speculation / weak
	for _, s := range []string{"might", "could", "perhaps", "maybe", "discussed"} {
		if strings.Contains(lower, s) {
			score -= 0.1
			break
		}
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score
}
