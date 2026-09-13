package recall

import (
	"sort"
	"strings"
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
	minScore := ScoreFloorFromTop(scoreOfItem(all[0]))
	if minScore > 0 {
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

// ScoreFloorFromTop is the relative score floor used by compile grouped views and
// benchmark flatten. Returns 0 when top is too weak to apply a floor.
func ScoreFloorFromTop(top float64) float64 {
	if top < 0.12 {
		return 0
	}
	minScore := top * 0.08
	if minScore < 0.06 {
		minScore = 0.06
	}
	if top >= 0.75 && top < 2.0 {
		if rel := top * 0.28; rel > minScore {
			minScore = rel
		}
	} else if top >= 0.5 && top < 0.75 {
		if rel := top * 0.56; rel > minScore {
			minScore = rel
		}
	} else if top >= 2.0 {
		if rel := top * 0.12; rel > minScore {
			minScore = rel
		}
	}
	return minScore
}

func scoreForFloor(s ScoredMemory) float64 {
	if s.Components != nil {
		return s.Components.RelevanceScore
	}
	return s.Score
}

// filterScoredByFloor drops confirmed off-topic rows when the situation names a
// product anchor (or a row already has a wrong-domain penalty). Utility cannot
// revive those rows. Queries without an anchor keep r=0 rows so a narrow
// sentence does not wipe still-true governing/recent decisions.
func filterScoredByFloor(scored []ScoredMemory, situationQuery string) []ScoredMemory {
	if len(scored) == 0 || strings.TrimSpace(situationQuery) == "" {
		return scored
	}
	if !textMentionsProductAnchorTerm(situationQuery) {
		return scored
	}
	out := make([]ScoredMemory, 0, len(scored))
	for _, s := range scored {
		if s.Components != nil && s.Components.WrongDomainPenalty >= 0.3 {
			continue
		}
		if scoreForFloor(s) <= 0 {
			continue
		}
		out = append(out, s)
	}
	return out
}

func scoreOfItem(it MemoryItem) float64 {
	if it.Justification != nil {
		return it.Justification.Score
	}
	return 0
}
