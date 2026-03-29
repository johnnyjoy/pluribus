package similarity

import (
	"strings"
)

// CanonicalTokenJaccard is |A∩B|/|A∪B| over token sets (e.g. memorynorm.StatementCanonical text).
// Used for recall bundle near-dup collapse (Phase F) and advisory resemblance.
func CanonicalTokenJaccard(a, b string) float64 {
	return tokenJaccard(a, b)
}

// tokenJaccard is |A∩B|/|A∪B| over whitespace-normalized tokens (case-insensitive).
// Self-contained: no external services; used for default advisory resemblance.
func tokenJaccard(a, b string) float64 {
	tokA := tokenizeAdvisory(a)
	tokB := tokenizeAdvisory(b)
	if len(tokA) == 0 && len(tokB) == 0 {
		return 1
	}
	if len(tokA) == 0 || len(tokB) == 0 {
		return 0
	}
	setA := make(map[string]struct{}, len(tokA))
	for _, t := range tokA {
		setA[t] = struct{}{}
	}
	inter := 0
	for _, t := range tokB {
		if _, ok := setA[t]; ok {
			inter++
		}
	}
	union := len(tokA) + len(tokB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func tokenizeAdvisory(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	f := strings.Fields(s)
	if len(f) == 0 {
		return nil
	}
	out := make([]string, 0, len(f))
	for _, w := range f {
		w = strings.Trim(w, ".,;:!?\"'()[]{}")
		if len(w) >= 2 {
			out = append(out, w)
		}
	}
	return out
}

// tagJaccard computes Jaccard similarity on normalized tag strings.
func tagJaccard(a, b []string) float64 {
	setA := tagSet(a)
	setB := tagSet(b)
	if len(setA) == 0 && len(setB) == 0 {
		return 0
	}
	if len(setA) == 0 || len(setB) == 0 {
		return 0
	}
	inter := 0
	for t := range setA {
		if _, ok := setB[t]; ok {
			inter++
		}
	}
	union := len(setA) + len(setB) - inter
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}

func tagSet(tags []string) map[string]struct{} {
	m := make(map[string]struct{})
	for _, t := range tags {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			m[t] = struct{}{}
		}
	}
	return m
}

// resemblanceScore combines lexical overlap with optional tag overlap (all local signals).
// filterTags: when non-empty, caller must already enforce tagOverlap(rec) before scoring.
func resemblanceScore(query, summary string, filterTags, episodeTags []string) (score float64, signals []string) {
	lex := tokenJaccard(query, summary)
	if lex >= 0.12 {
		signals = append(signals, "lexical_overlap")
	}
	tj := tagJaccard(filterTags, episodeTags)
	if len(filterTags) > 0 && len(episodeTags) > 0 && tj > 0 {
		signals = append(signals, "shared_tags")
	}

	if len(filterTags) > 0 {
		// Query narrowed by tags: blend lexical with tag agreement.
		score = 0.65*lex + 0.35*tj
	} else {
		score = lex
		if len(episodeTags) > 0 && lex >= 0.08 {
			// Weak boost when episode is tagged and query matches some tag tokens in text
			// (does not require query to list tags).
			for _, et := range episodeTags {
				et = strings.TrimSpace(strings.ToLower(et))
				if et == "" {
					continue
				}
				if strings.Contains(strings.ToLower(summary), et) || strings.Contains(strings.ToLower(query), et) {
					if score+0.05 < 1.0 {
						score += 0.05
					} else {
						score = 1.0
					}
					signals = append(signals, "tag_echo_in_text")
					break
				}
			}
		}
	}
	return score, signals
}
