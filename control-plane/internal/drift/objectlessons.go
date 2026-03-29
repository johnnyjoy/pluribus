package drift

import (
	"encoding/json"
	"strings"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

// severityWeight returns a numeric weight for violation vs warning split (>= 4 → violation).
func severityWeight(severity string) float64 {
	switch strings.ToLower(strings.TrimSpace(severity)) {
	case "catastrophic":
		return 6.0
	case "high":
		return 4.0
	case "medium":
		return 2.5
	case "low":
		return 1.0
	default:
		return 1.0
	}
}

// containsAny returns true if proposal (lowercased) contains any of the tags as a substring.
func containsAny(proposal string, tags []string) bool {
	lower := strings.ToLower(proposal)
	for _, t := range tags {
		if t != "" && strings.Contains(lower, strings.ToLower(t)) {
			return true
		}
	}
	return false
}

// NegativePatternMatches returns drift issues when proposal overlaps negative pattern payloads
// (directive, experience, or tags). Score is set to severity weight; caller uses Score >= 4 for violations.
func NegativePatternMatches(proposal string, objects []memory.MemoryObject) []DriftIssue {
	var out []DriftIssue
	lower := strings.ToLower(proposal)

	for _, o := range objects {
		if o.Kind != api.MemoryKindPattern {
			continue
		}
		if len(o.Payload) == 0 {
			continue
		}
		var p memory.PatternPayload
		if err := json.Unmarshal(o.Payload, &p); err != nil {
			continue
		}
		polarity := strings.ToLower(strings.TrimSpace(p.Polarity))
		if polarity != "negative" {
			continue
		}

		var match bool
		var snippet string
		if p.Directive != "" && strings.Contains(lower, strings.ToLower(p.Directive)) {
			match = true
			snippet = p.Directive
		}
		if !match && p.Experience != "" && strings.Contains(lower, strings.ToLower(p.Experience)) {
			match = true
			snippet = p.Experience
		}
		if !match && len(o.Tags) > 0 && containsAny(proposal, o.Tags) {
			match = true
			snippet = "tag overlap"
		}
		if !match {
			continue
		}

		sev := p.Impact.Severity
		if sev == "" {
			sev = "medium"
		}
		weight := severityWeight(sev)
		msg := "Proposal overlaps negative pattern: " + snippet
		out = append(out, DriftIssue{
			Code:      "negative_pattern",
			Statement: msg,
			Score:     weight,
		})
	}

	return out
}
