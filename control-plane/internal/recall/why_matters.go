package recall

import (
	"fmt"
	"strings"
)

// sessionTagMatches is true when tags contain mcp:session:<correlationID> (same convention as advisory ingest).
func sessionTagMatches(correlationID string, tags []string) bool {
	correlationID = strings.TrimSpace(correlationID)
	if correlationID == "" {
		return false
	}
	want := "mcp:session:" + correlationID
	for _, t := range tags {
		if t == want {
			return true
		}
	}
	return false
}

// BuildWhyMattersLine returns a deterministic "why this matters" line for agent-facing recall (no LLM).
func BuildWhyMattersLine(kind string, rankReason string, evCount int, sessionMatch bool) string {
	k := strings.TrimSpace(strings.ToLower(kind))
	var role string
	switch k {
	case "constraint":
		role = "This constraint limits what you should do here."
	case "failure":
		role = "This failure record helps you avoid repeating a known break."
	case "pattern":
		role = "This pattern encodes a repeatable approach."
	case "decision", "state":
		role = "This supports continuity with prior decisions or state."
	default:
		role = "This memory is relevant to your current situation."
	}
	reason := strings.TrimSpace(rankReason)
	if reason == "" {
		reason = "ranking"
	}
	s := role + " Why surfaced: " + reason + "."
	if sessionMatch {
		s += " Session-local: tagged for your correlation_id."
	}
	if evCount > 0 {
		s += fmt.Sprintf(" Supported by %d evidence record(s).", evCount)
	}
	return s
}

// AugmentWhyMattersWithEvidence rebuilds why_matters after SupportingEvidence is attached.
func AugmentWhyMattersWithEvidence(b *RecallBundle) {
	if b == nil {
		return
	}
	refresh := func(items []MemoryItem) []MemoryItem {
		for i := range items {
			it := &items[i]
			n := len(it.SupportingEvidence)
			reason := "ranking"
			if it.Justification != nil && strings.TrimSpace(it.Justification.Reason) != "" {
				reason = it.Justification.Reason
			}
			it.WhyMatters = BuildWhyMattersLine(it.Kind, reason, n, it.SessionLocal)
		}
		return items
	}
	b.GoverningConstraints = refresh(b.GoverningConstraints)
	b.Decisions = refresh(b.Decisions)
	b.KnownFailures = refresh(b.KnownFailures)
	b.ApplicablePatterns = refresh(b.ApplicablePatterns)
	b.Continuity = refresh(b.Continuity)
	b.Constraints = refresh(b.Constraints)
	b.Experience = refresh(b.Experience)
}
