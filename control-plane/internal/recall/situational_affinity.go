package recall

import (
	"path/filepath"
	"strings"
)

// SituationAffinityInput carries optional situational hints for ranking boosts only.
// Never filters the candidate pool (global memory doctrine).
type SituationAffinityInput struct {
	SituationQuery string
	RepoRoot       string
	RequestTags    []string
}

var situationStopWords = map[string]struct{}{
	"the": {}, "a": {}, "an": {}, "and": {}, "or": {}, "to": {}, "of": {}, "in": {}, "on": {},
	"for": {}, "with": {}, "at": {}, "by": {}, "from": {}, "into": {}, "onto": {}, "via": {},
	"this": {}, "that": {}, "what": {}, "when": {}, "where": {}, "which": {}, "while": {},
	"have": {}, "has": {}, "had": {}, "been": {}, "being": {}, "were": {}, "was": {}, "are": {}, "is": {},
}

// domainTokensFromText extracts lowercase tokens for situational affinity (len>=4, not stopwords).
func domainTokensFromText(text string) map[string]struct{} {
	out := map[string]struct{}{}
	q := strings.ToLower(strings.TrimSpace(text))
	if q == "" {
		return out
	}
	for _, tok := range strings.FieldsFunc(q, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	}) {
		tok = strings.TrimSpace(tok)
		if len(tok) < 3 {
			continue
		}
		if _, stop := situationStopWords[tok]; stop {
			continue
		}
		out[tok] = struct{}{}
	}
	return out
}

// querySituationTokens builds the query-side token set: situation text + optional repo basename + request tags.
func querySituationTokens(in SituationAffinityInput) map[string]struct{} {
	tokens := domainTokensFromText(in.SituationQuery)
	if in.RepoRoot != "" {
		base := strings.ToLower(filepath.Base(filepath.Clean(in.RepoRoot)))
		if base != "" && base != "." && len(base) >= 3 {
			tokens[base] = struct{}{}
		}
	}
	for _, tag := range in.RequestTags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		// Tags may be slugs; include whole tag and last path segment when dotted/slashed.
		tokens[tag] = struct{}{}
		if i := strings.LastIndexAny(tag, "/:"); i >= 0 && i+1 < len(tag) {
			seg := tag[i+1:]
			if len(seg) >= 3 {
				tokens[seg] = struct{}{}
			}
		}
	}
	return tokens
}

// memorySituationTokens builds memory-side tokens from statement and tags.
func memorySituationTokens(statement string, tags []string) map[string]struct{} {
	tokens := domainTokensFromText(statement)
	for _, tag := range tags {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" || isBenchmarkInfraTag(tag) {
			continue
		}
		tokens[tag] = struct{}{}
		if i := strings.LastIndexAny(tag, "/:"); i >= 0 && i+1 < len(tag) {
			seg := tag[i+1:]
			if len(seg) >= 3 {
				tokens[seg] = struct{}{}
			}
		}
	}
	return tokens
}

// tokenWeight increases influence of longer, more domain-specific query tokens.
func tokenWeight(tok string) float64 {
	switch {
	case len(tok) >= 8:
		return 1.5
	case len(tok) >= 6:
		return 1.25
	default:
		return 1.0
	}
}

// SituationAffinityScore returns [0,1]: weighted share of query-side domain tokens matched in the memory.
// Query coverage favors distinctive tokens (longer words) over generic ones shared across projects.
// Additive ranking only; does not exclude candidates.
func SituationAffinityScore(statement string, tags []string, in SituationAffinityInput) float64 {
	qTokens := querySituationTokens(in)
	if len(qTokens) == 0 {
		return 0
	}
	mTokens := memorySituationTokens(statement, tags)
	if len(mTokens) == 0 {
		return 0
	}
	var matchedW, totalW float64
	for tok := range qTokens {
		w := tokenWeight(tok)
		if isGenericRecallTerm(tok) {
			w *= 0.15
		}
		totalW += w
		if _, ok := mTokens[tok]; ok {
			matchedW += w
		}
	}
	if totalW == 0 {
		return 0
	}
	return matchedW / totalW
}
