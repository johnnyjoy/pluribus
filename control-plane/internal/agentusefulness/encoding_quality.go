package agentusefulness

import (
	"strings"
)

// EncodingCues are retrieval-oriented metadata stored with a memory fixture.
type EncodingCues struct {
	TaskType       string   `json:"task_type,omitempty"`
	Domain         string   `json:"domain,omitempty"`
	Project        string   `json:"project,omitempty"`
	System         string   `json:"system,omitempty"`
	RetrievalTerms []string `json:"retrieval_terms,omitempty"`
	Scope          string   `json:"scope,omitempty"`
	NegativeScope  []string `json:"negative_scope,omitempty"`
}

// CueMatchResult reports encoding specificity alignment for a task-memory pair.
type CueMatchResult struct {
	Label           string  `json:"label"`
	MatchScore      float64 `json:"match_score"`
	UnderEncoded    bool    `json:"under_encoded"`
	MisleadingCue   bool    `json:"misleading_cue_risk"`
	WrongContext    bool    `json:"wrong_context"`
	NegativeScopeHit bool   `json:"negative_scope_hit"`
}

// EvaluateCueMatch scores cue overlap between task and memory encoding cues.
// Engineering interpretation of Tulving & Thomson encoding specificity (1973).
func EvaluateCueMatch(task TaskFixture, mem FixtureMemory) CueMatchResult {
	res := CueMatchResult{Label: mem.Label}
	cues := mem.EncodingCues
	if cues == nil && len(mem.RetrievalTerms) == 0 && mem.Scope == "" {
		res.UnderEncoded = true
		res.MatchScore = 0
		return res
	}
	if cues == nil {
		cues = &EncodingCues{
			RetrievalTerms: mem.RetrievalTerms,
			Scope:          mem.Scope,
			NegativeScope:  mem.NegativeScope,
		}
	}

	// Negative scope: memory explicitly does not apply to task domain.
	for _, ns := range cues.NegativeScope {
		for _, dt := range task.DomainTags {
			if ns == dt {
				res.NegativeScopeHit = true
				res.WrongContext = true
				res.MatchScore = 0
				return res
			}
		}
	}

	// Scope / domain tag overlap.
	scopeMatch := false
	if cues.Scope != "" {
		for _, dt := range task.DomainTags {
			if cues.Scope == dt {
				scopeMatch = true
				break
			}
		}
	} else {
		scopeMatch = domainOverlap(task.DomainTags, mem.Tags)
	}

	// Retrieval term overlap with task query.
	query := strings.ToLower(task.RecallQuery + " " + task.TaskPrompt)
	var termHits, termTotal int
	for _, term := range cues.RetrievalTerms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" {
			continue
		}
		termTotal++
		if strings.Contains(query, term) {
			termHits++
		}
	}
	termScore := 1.0
	if termTotal > 0 {
		termScore = float64(termHits) / float64(termTotal)
	}

	if termTotal == 0 && cues.Scope == "" && len(mem.Tags) == 0 {
		res.UnderEncoded = true
	}

	res.MatchScore = termScore
	if scopeMatch {
		res.MatchScore = (res.MatchScore + 1.0) / 2.0
	} else if len(task.DomainTags) > 0 {
		res.WrongContext = true
		res.MatchScore *= 0.25
	}

	// Misleading: high term overlap but wrong scope (collision).
	if termScore >= 0.5 && !scopeMatch && len(task.DomainTags) > 0 {
		res.MisleadingCue = true
	}

	return res
}

// MinCueMatchThreshold is the minimum match score to treat encoding as adequate.
const MinCueMatchThreshold = 0.5
