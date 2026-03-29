package merge

import (
	"sort"
	"strings"
)

// MergeOptions controls merge uniques cap/dedupe and strict conflicts. Nil or zero values preserve default merge behavior.
type MergeOptions struct {
	// StrictConflicts enables grey-zone conflict detection (e.g. should vs should not) in addition to conservative rules.
	StrictConflicts bool `json:"strict_conflicts,omitempty"`
	// MaxUniqueBullets caps the number of unique lines after other filters; 0 = unlimited.
	MaxUniqueBullets int `json:"max_unique_bullets,omitempty"`
	// DedupeSimilarUniques removes near-duplicate lines among uniques (greedy, deterministic).
	DedupeSimilarUniques bool `json:"dedupe_similar_uniques,omitempty"`
	// DropUniqueIfSimilarToAgreement in (0,1]: drop a unique if max_agreement Similarity(unique, agreement) >= this (redundant with CORE).
	DropUniqueIfSimilarToAgreement float64 `json:"drop_unique_if_similar_to_agreement,omitempty"`
}

// DedupeUniquesSimilarityThreshold is used when DedupeSimilarUniques is true.
const DedupeUniquesSimilarityThreshold = 0.85

// applyUniquesPipeline runs Phase 5.2 filters in order: agreement-similarity drop → pairwise dedupe → count cap.
// Returns filtered uniques and how many lines were removed vs the input slice.
func applyUniquesPipeline(uniques, agreements []string, opts *MergeOptions) (out []string, removed int) {
	if opts == nil {
		return uniques, 0
	}
	out = append([]string(nil), uniques...)
	before := len(out)

	if opts.DropUniqueIfSimilarToAgreement > 0 && opts.DropUniqueIfSimilarToAgreement <= 1 && len(agreements) > 0 {
		th := opts.DropUniqueIfSimilarToAgreement
		var keep []string
		for _, u := range out {
			drop := false
			for _, a := range agreements {
				if Similarity(u, a) >= th {
					drop = true
					break
				}
			}
			if !drop {
				keep = append(keep, u)
			}
		}
		out = keep
	}

	if opts.DedupeSimilarUniques && len(out) > 1 {
		out = dedupeSimilarUniques(out, DedupeUniquesSimilarityThreshold)
	}

	if opts.MaxUniqueBullets > 0 && len(out) > opts.MaxUniqueBullets {
		// Deterministic: sort by normalized text, keep first N.
		sort.Slice(out, func(i, j int) bool {
			return Normalize(out[i]) < Normalize(out[j])
		})
		out = out[:opts.MaxUniqueBullets]
	}

	removed = before - len(out)
	return out, removed
}

func dedupeSimilarUniques(in []string, threshold float64) []string {
	if len(in) <= 1 {
		return in
	}
	sorted := append([]string(nil), in...)
	sort.Slice(sorted, func(i, j int) bool {
		return Normalize(sorted[i]) < Normalize(sorted[j])
	})
	var out []string
	for _, s := range sorted {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		dup := false
		for _, kept := range out {
			if Similarity(s, kept) >= threshold {
				dup = true
				break
			}
		}
		if !dup {
			out = append(out, s)
		}
	}
	return out
}
