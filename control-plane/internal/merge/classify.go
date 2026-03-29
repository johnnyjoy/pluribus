package merge

import (
	"regexp"
	"sort"
	"strings"
)

var numRe = regexp.MustCompile(`\b\d+(?:\.\d+)?\b`)

// segmentsConflict returns true if two segments from different variants contradict (heuristic).
func segmentsConflict(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	na, nb := Normalize(a), Normalize(b)

	// Numeric mismatch when texts are somewhat similar but number sets differ
	numsA := numRe.FindAllString(na, -1)
	numsB := numRe.FindAllString(nb, -1)
	if len(numsA) > 0 && len(numsB) > 0 && Similarity(a, b) >= 0.25 {
		if !equalStringSets(numsA, numsB) {
			return true
		}
	}

	// must not / do not / never vs must / always (minimal rules)
	if negPosMust(na, nb) || negPosMust(nb, na) {
		return true
	}
	if (strings.Contains(na, "never") && strings.Contains(nb, "always")) ||
		(strings.Contains(nb, "never") && strings.Contains(na, "always")) {
		if Similarity(a, b) >= 0.35 {
			return true
		}
	}
	return false
}

func equalStringSets(a, b []string) bool {
	ma, mb := make(map[string]int), make(map[string]int)
	for _, x := range a {
		ma[x]++
	}
	for _, x := range b {
		mb[x]++
	}
	if len(ma) != len(mb) {
		return false
	}
	for k, v := range ma {
		if mb[k] != v {
			return false
		}
	}
	return true
}

func negPosMust(negLine, posLine string) bool {
	if !hasNegation(negLine) {
		return false
	}
	if !strings.Contains(posLine, "must ") || strings.Contains(posLine, "must not") {
		return false
	}
	// shared substantive tokens beyond "must"
	tn := tokenSet(negLine)
	tp := tokenSet(posLine)
	var inter int
	for t := range tn {
		if _, ok := tp[t]; ok {
			inter++
		}
	}
	return inter >= 3 && Similarity(negLine, posLine) >= 0.3
}

func hasNegation(s string) bool {
	s = Normalize(s)
	return strings.Contains(s, "must not") || strings.Contains(s, "do not") ||
		strings.Contains(s, "never ") || strings.Contains(s, "must never")
}

// hasNegationOrSoftBan marks common prohibitions beyond hasNegation (strict conflict mode).
func hasNegationOrSoftBan(s string) bool {
	s = Normalize(s)
	if hasNegation(s) {
		return true
	}
	return strings.Contains(s, "should not") || strings.Contains(s, " cannot ") ||
		strings.Contains(s, " can't ") || strings.Contains(s, " mustn't ")
}

// strictAugmentedConflict catches grey-zone opposite polarity when conservative segmentsConflict is false.
// Only used when MergeOptions.StrictConflicts is true.
func strictAugmentedConflict(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if segmentsConflict(a, b) {
		return false
	}
	na, nb := Normalize(a), Normalize(b)
	sim := Similarity(a, b)
	// High similarity is expected for "should not" vs "should" (single-token polarity flip); no upper bound.
	if sim < 0.36 {
		return false
	}
	negA := hasNegationOrSoftBan(na)
	negB := hasNegationOrSoftBan(nb)
	if negA == negB {
		return false
	}
	tn := tokenSet(na)
	tp := tokenSet(nb)
	inter := 0
	for t := range tn {
		if len(t) <= 2 {
			continue
		}
		if _, ok := tp[t]; ok {
			inter++
		}
	}
	return inter >= 3
}

// markConflicts returns indices of segments involved in pairwise conflicts (different variants).
// Equivalent to markConflictsWithStrict(segments, false).
func markConflicts(segments []Segment) (conflictTexts []string, bad map[int]struct{}) {
	return markConflictsWithStrict(segments, false)
}

// markConflictsWithStrict runs conflict detection; when strict is true, additional grey-zone pairs are marked.
func markConflictsWithStrict(segments []Segment, strict bool) (conflictTexts []string, bad map[int]struct{}) {
	bad = make(map[int]struct{})
	seen := make(map[string]struct{})
	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			if segments[i].Variant == segments[j].Variant {
				continue
			}
			a, b := segments[i].Text, segments[j].Text
			conf := segmentsConflict(a, b) || (strict && strictAugmentedConflict(a, b))
			if !conf {
				continue
			}
			bad[i] = struct{}{}
			bad[j] = struct{}{}
			for _, idx := range []int{i, j} {
				t := strings.TrimSpace(segments[idx].Text)
				key := segments[idx].Variant + "\x00" + Normalize(t)
				if _, ok := seen[key]; !ok {
					seen[key] = struct{}{}
					conflictTexts = append(conflictTexts, t)
				}
			}
		}
	}
	return conflictTexts, bad
}

type cluster struct {
	segs []Segment
}

// clusterSegments groups non-conflict segments by similarity (greedy, deterministic order).
func clusterSegments(segments []Segment, bad map[int]struct{}) []cluster {
	type idxSeg struct {
		i int
		s Segment
	}
	var list []idxSeg
	for i, s := range segments {
		if _, ok := bad[i]; ok {
			continue
		}
		list = append(list, idxSeg{i: i, s: s})
	}
	sort.Slice(list, func(i, j int) bool {
		ni, nj := Normalize(list[i].s.Text), Normalize(list[j].s.Text)
		if ni != nj {
			return ni < nj
		}
		return list[i].s.Variant < list[j].s.Variant
	})

	var clusters []cluster
outer:
	for _, item := range list {
		for c := range clusters {
			rep := clusters[c].segs[0].Text
			if Similarity(rep, item.s.Text) >= AgreementSimilarityThreshold {
				clusters[c].segs = append(clusters[c].segs, item.s)
				continue outer
			}
		}
		clusters = append(clusters, cluster{segs: []Segment{item.s}})
	}
	return clusters
}

func variantSet(segs []Segment) map[string]struct{} {
	m := make(map[string]struct{})
	for _, s := range segs {
		m[s.Variant] = struct{}{}
	}
	return m
}

// AgreementsUniquesFromClusters derives agreement representatives and unique lines.
func AgreementsUniquesFromClusters(clusters []cluster) (agreements, uniques []string) {
	var agreeReps []string
	for _, cl := range clusters {
		vs := variantSet(cl.segs)
		if len(vs) >= 2 {
			// representative: lexicographically first by normalized text among segments
			rep := cl.segs[0].Text
			for _, s := range cl.segs[1:] {
				if Normalize(s.Text) < Normalize(rep) {
					rep = s.Text
				}
			}
			agreeReps = append(agreeReps, strings.TrimSpace(rep))
		}
	}
	agreements = dedupeSorted(agreeReps)

	// Single-variant clusters: each non-subsumed segment text is a unique candidate.
	var uni []string
	for _, cl := range clusters {
		vs := variantSet(cl.segs)
		if len(vs) >= 2 {
			continue
		}
		for _, s := range cl.segs {
			t := strings.TrimSpace(s.Text)
			subsumed := false
			for _, ar := range agreements {
				if Similarity(t, ar) >= AgreementSimilarityThreshold {
					subsumed = true
					break
				}
			}
			if !subsumed {
				uni = append(uni, t)
			}
		}
	}
	uniques = dedupeSorted(uni)
	return agreements, uniques
}

func dedupeSorted(in []string) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		k := Normalize(s)
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return Normalize(out[i]) < Normalize(out[j])
	})
	return out
}

// UsedVariantsFromClusters lists variants that contributed to agreement clusters (≥2 variants) or unique lines.
func UsedVariantsFromClusters(clusters []cluster, uniques []string) []string {
	m := make(map[string]struct{})
	uniqNorm := make(map[string]struct{})
	for _, u := range uniques {
		uniqNorm[Normalize(u)] = struct{}{}
	}
	for _, cl := range clusters {
		vs := variantSet(cl.segs)
		if len(vs) >= 2 {
			for v := range vs {
				m[v] = struct{}{}
			}
			continue
		}
		for _, s := range cl.segs {
			if _, ok := uniqNorm[Normalize(s.Text)]; ok {
				m[s.Variant] = struct{}{}
			}
		}
	}
	var vs []string
	for v := range m {
		vs = append(vs, v)
	}
	sort.Strings(vs)
	return vs
}
