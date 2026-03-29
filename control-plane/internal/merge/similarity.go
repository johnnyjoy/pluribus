package merge

import (
	"strings"
	"unicode"
)

// AgreementSimilarityThreshold is the minimum similarity for clustering into agreement.
const AgreementSimilarityThreshold = 0.55

// MinSegmentTokens is the minimum token count for overlap ratio denominator guard.
const MinSegmentTokens = 2

// Normalize collapses whitespace and lowercases for comparison.
func Normalize(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if unicode.IsSpace(r) {
			if !lastSpace && b.Len() > 0 {
				b.WriteRune(' ')
				lastSpace = true
			}
			continue
		}
		lastSpace = false
		// strip most punctuation at edges of "words" lightly: keep alnum
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	return strings.TrimSpace(b.String())
}

func tokenSet(s string) map[string]struct{} {
	s = Normalize(s)
	toks := strings.Fields(s)
	m := make(map[string]struct{})
	for _, t := range toks {
		if len(t) < 2 {
			continue
		}
		m[t] = struct{}{}
	}
	return m
}

// Similarity returns a score in [0,1]: identical normalized → 1; else token overlap with optional substring boost.
func Similarity(a, b string) float64 {
	na, nb := Normalize(a), Normalize(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	A := tokenSet(a)
	B := tokenSet(b)
	if len(A) < MinSegmentTokens && len(B) < MinSegmentTokens {
		// fall back to containment
		if strings.Contains(na, nb) || strings.Contains(nb, na) {
			return 0.75
		}
		return 0
	}
	var inter int
	smaller := len(A)
	if len(B) < smaller {
		smaller = len(B)
	}
	for t := range A {
		if _, ok := B[t]; ok {
			inter++
		}
	}
	if smaller == 0 {
		return 0
	}
	ratio := float64(inter) / float64(smaller)
	// substring boost
	if len(na) >= len(nb) && strings.Contains(na, nb) {
		ratio += 0.1
	} else if len(nb) > len(na) && strings.Contains(nb, na) {
		ratio += 0.1
	}
	if ratio > 1 {
		return 1
	}
	return ratio
}
