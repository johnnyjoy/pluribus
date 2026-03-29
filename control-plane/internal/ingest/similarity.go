package ingest

import (
	"strings"
)

// DefaultSimilarJaccardMin is the minimum token Jaccard score on both predicate and object
// (same subject required) to treat two facts as a minor variation eligible for similar_unify.
const DefaultSimilarJaccardMin = 0.6

// DefaultConflictObjectMaxJaccard: same normalized predicate and subject, but object Jaccard
// below this threshold implies a divergent (contradictory-style) pair for debug-only reporting.
const DefaultConflictObjectMaxJaccard = 0.35

// TokenJaccard computes |A∩B|/|A∪B| over whitespace tokens (already normalized strings).
func TokenJaccard(a, b string) float64 {
	tokA := tokenizeFactString(a)
	tokB := tokenizeFactString(b)
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

func tokenizeFactString(s string) []string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return nil
	}
	return f
}

// SimilarForUnify reports whether two triples are the same subject and “close enough”
// on predicate and object to unify wording to a shared canonical triple.
func SimilarForUnify(subjectA, predA, objA, subjectB, predB, objB string, minJaccard float64) bool {
	if subjectA != subjectB {
		return false
	}
	if minJaccard <= 0 {
		minJaccard = DefaultSimilarJaccardMin
	}
	return TokenJaccard(predA, predB) >= minJaccard && TokenJaccard(objA, objB) >= minJaccard
}
