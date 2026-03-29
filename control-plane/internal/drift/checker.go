package drift

import (
	"strings"
	"unicode"
)

// Check runs string-based drift detection: for each constraint and failure statement,
// if the proposal contains the statement (case-insensitive substring), it is a violation.
// If failureFuzzyThreshold > 0, also adds failure_pattern issues when word overlap >= threshold (Task 76).
func Check(proposal string, constraintStatements, failureStatements []string, failureFuzzyThreshold float64) []DriftIssue {
	var issues []DriftIssue
	lower := strings.ToLower(proposal)
	for _, s := range constraintStatements {
		if s != "" && strings.Contains(lower, strings.ToLower(s)) {
			issues = append(issues, DriftIssue{Code: "constraint", Statement: s})
		}
	}
	substringMatched := make(map[string]bool)
	for _, s := range failureStatements {
		if s != "" && strings.Contains(lower, strings.ToLower(s)) {
			issues = append(issues, DriftIssue{Code: "failure", Statement: s})
			substringMatched[s] = true
		}
	}
	if failureFuzzyThreshold > 0 {
		proposalWords := wordSet(proposal)
		for _, s := range failureStatements {
			if s == "" || substringMatched[s] {
				continue
			}
			overlap := wordOverlapRatio(proposalWords, wordSet(s))
			if overlap >= failureFuzzyThreshold {
				issues = append(issues, DriftIssue{Code: "failure_pattern", Statement: s, Score: overlap})
			}
		}
	}
	return issues
}

func wordSet(text string) map[string]struct{} {
	set := make(map[string]struct{})
	f := func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsNumber(r) }
	for _, w := range strings.FieldsFunc(strings.ToLower(text), f) {
		if len(w) > 0 {
			set[w] = struct{}{}
		}
	}
	return set
}

// wordOverlapRatio returns |A ∩ B| / |B| (fraction of statement words present in proposal).
func wordOverlapRatio(proposalWords map[string]struct{}, statementWords map[string]struct{}) float64 {
	if len(statementWords) == 0 {
		return 0
	}
	var overlap int
	for w := range statementWords {
		if _, ok := proposalWords[w]; ok {
			overlap++
		}
	}
	return float64(overlap) / float64(len(statementWords))
}
