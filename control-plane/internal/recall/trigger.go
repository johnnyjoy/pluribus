package recall

import (
	"strconv"
	"strings"
	"unicode"
)

// TriggerKind is one of three sprint-defined trigger types.
type TriggerKind string

const (
	TriggerKindRisk        TriggerKind = "risk"
	TriggerKindDecision    TriggerKind = "decision"
	TriggerKindSimilarity  TriggerKind = "similarity"
)

// ValidTriggerKind returns true if k is risk|decision|similarity.
func ValidTriggerKind(k string) bool {
	switch TriggerKind(k) {
	case TriggerKindRisk, TriggerKindDecision, TriggerKindSimilarity:
		return true
	default:
		return false
	}
}

// TriggerInput carries optional text and IDs for heuristic detection (no I/O).
type TriggerInput struct {
	ProposalText      string
	TaskTitle         string
	TargetGoal        string
	ExistingQuery     string
	Tags              []string
	ChangedFilesCount *int
}

// TriggerDecision is an explainable trigger match for logs and API metadata.
type TriggerDecision struct {
	Kind       TriggerKind `json:"kind"`
	Reason     string      `json:"reason"`
	Confidence float64     `json:"confidence,omitempty"`
}

var englishStop = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "to": {}, "of": {}, "in": {}, "on": {},
	"for": {}, "with": {}, "at": {}, "by": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"be": {}, "been": {}, "being": {}, "have": {}, "has": {}, "had": {}, "do": {}, "does": {},
	"did": {}, "will": {}, "would": {}, "could": {}, "should": {}, "this": {}, "that": {},
	"it": {}, "we": {}, "you": {}, "they": {}, "as": {}, "from": {},
}

func contentTokens(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	var tokens []string
	for _, w := range strings.FieldsFunc(s, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r)
	}) {
		w = strings.TrimSpace(w)
		if len(w) < 3 {
			continue
		}
		if _, stop := englishStop[w]; stop {
			continue
		}
		tokens = append(tokens, w)
	}
	return tokens
}

func combinedContext(in TriggerInput) string {
	var b strings.Builder
	if in.ProposalText != "" {
		b.WriteString(in.ProposalText)
		b.WriteByte(' ')
	}
	if in.TargetGoal != "" {
		b.WriteString(in.TargetGoal)
		b.WriteByte(' ')
	}
	if in.TaskTitle != "" {
		b.WriteString(in.TaskTitle)
		b.WriteByte(' ')
	}
	if in.ExistingQuery != "" {
		b.WriteString(in.ExistingQuery)
	}
	return strings.TrimSpace(b.String())
}

var riskSubstrings = []string{
	"deploy", "release", "production", "migrate", "migration", "rollback",
	"destructive", "truncate", "drop table", "authz", "permission", "secret",
	"privilege", "firewall", "delete all", "bulk delete",
}

var decisionSubstrings = []string{
	"either ", " or ", " vs ", "versus", "choose between", "which option",
	"should we", "option a", "option b", "alternatives", "tradeoff", "trade-off",
	"pick between", "decide between",
}

// DetectTriggers returns heuristic trigger decisions (pure; no DB/LLM).
// Order: risk, decision, similarity (at most one of each when caps applied later).
func DetectTriggers(in TriggerInput, minTokens int) []TriggerDecision {
	if minTokens <= 0 {
		minTokens = 4
	}
	ctx := combinedContext(in)
	toks := contentTokens(ctx)
	if len(toks) < minTokens {
		return nil
	}
	low := strings.ToLower(ctx)

	var out []TriggerDecision

	// Risk
	for _, sub := range riskSubstrings {
		if strings.Contains(low, sub) {
			out = append(out, TriggerDecision{
				Kind:       TriggerKindRisk,
				Reason:     "keyword:" + strings.TrimSpace(sub),
				Confidence: 0.75,
			})
			break
		}
	}
	if in.ChangedFilesCount != nil && *in.ChangedFilesCount >= 25 {
		out = append(out, TriggerDecision{
			Kind:       TriggerKindRisk,
			Reason:     "large_change:changed_files_count>=" + strconv.Itoa(*in.ChangedFilesCount),
			Confidence: 0.6,
		})
	}

	// Decision (single keyword pass)
	for _, sub := range decisionSubstrings {
		if strings.Contains(low, strings.TrimSpace(sub)) {
			out = append(out, TriggerDecision{
				Kind:       TriggerKindDecision,
				Reason:     "phrasing:" + strings.TrimSpace(sub),
				Confidence: 0.7,
			})
			break
		}
	}

	// Similarity: rich situational context (lexical / token richness — not embedding).
	if len(toks) >= minTokens+2 {
		out = append(out, TriggerDecision{
			Kind:       TriggerKindSimilarity,
			Reason:     "situational_context_rich:token_count=" + strconv.Itoa(len(toks)),
			Confidence: 0.55,
		})
	}

	return dedupeTriggerKinds(out)
}

func dedupeTriggerKinds(in []TriggerDecision) []TriggerDecision {
	seen := make(map[TriggerKind]struct{})
	var out []TriggerDecision
	for _, t := range in {
		if _, ok := seen[t.Kind]; ok {
			continue
		}
		seen[t.Kind] = struct{}{}
		out = append(out, t)
	}
	return out
}

// TriggerFragment returns a short query fragment to augment RetrievalQuery for a trigger.
func TriggerFragment(t TriggerDecision) string {
	switch t.Kind {
	case TriggerKindRisk:
		return "risks constraints failures"
	case TriggerKindDecision:
		return "decisions tradeoffs"
	case TriggerKindSimilarity:
		return "patterns experience"
	default:
		return ""
	}
}
