package recall

import (
	"sort"
)

// FlattenBundleByScore returns deduplicated bundle items sorted by justification score descending.
// Used by recall benchmarks and diagnostic tooling.
func FlattenBundleByScore(b *RecallBundle) []MemoryItem {
	if b == nil {
		return nil
	}
	seen := map[string]bool{}
	var all []MemoryItem
	for _, slice := range [][]MemoryItem{
		b.GoverningConstraints,
		b.Decisions,
		b.KnownFailures,
		b.ApplicablePatterns,
		b.Continuity,
		b.Constraints,
		b.Experience,
	} {
		for _, it := range slice {
			if it.ID == "" || seen[it.ID] {
				continue
			}
			seen[it.ID] = true
			all = append(all, it)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		si, sj := scoreOfItem(all[i]), scoreOfItem(all[j])
		if si != sj {
			return si > sj
		}
		return all[i].ID < all[j].ID
	})
	if len(all) == 0 {
		return all
	}
	top := scoreOfItem(all[0])
	if top >= 0.12 {
		minScore := top * 0.08
		if minScore < 0.06 {
			minScore = 0.06
		}
		if top >= 0.75 && top < 2.0 {
			relFloor := top * 0.28
			if relFloor > minScore {
				minScore = relFloor
			}
		} else if top >= 0.5 && top < 0.75 {
			relFloor := top * 0.56
			if relFloor > minScore {
				minScore = relFloor
			}
		} else if top >= 2.0 {
			relFloor := top * 0.12
			if relFloor > minScore {
				minScore = relFloor
			}
		}
		filtered := all[:0]
		for _, it := range all {
			if scoreOfItem(it) >= minScore {
				filtered = append(filtered, it)
			}
		}
		all = filtered
	}
	return all
}

func scoreOfItem(it MemoryItem) float64 {
	if it.Justification != nil {
		return it.Justification.Score
	}
	return 0
}
