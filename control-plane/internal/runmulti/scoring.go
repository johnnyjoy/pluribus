package runmulti

import (
	"sort"
)

// RiskLevelNumeric maps drift risk_level string to a number (lower = better).
// "" and unknown values map to 0.
func RiskLevelNumeric(riskLevel string) float64 {
	switch riskLevel {
	case "low":
		return 0
	case "medium":
		return 1
	case "high":
		return 2
	default:
		return 0
	}
}

// ScoreRun computes score and rejected from a drift result.
// Rejected is true if there are any violations. Score = riskLevelNumeric + len(warnings); lower is better.
func ScoreRun(d *DriftResult) (score float64, rejected bool) {
	if d == nil {
		return 0, false
	}
	rejected = len(d.Violations) > 0
	score = RiskLevelNumeric(d.RiskLevel) + float64(len(d.Warnings))
	return score, rejected
}

// SelectBest filters to non-rejected runs, sorts by score ascending then variant name, returns the best.
// If no valid runs, returns the run with lowest score among all (fallback). If runs is empty, returns nil, nil.
func SelectBest(runs []RunResult) (selected *RunResult) {
	if len(runs) == 0 {
		return nil
	}
	valid := make([]RunResult, 0, len(runs))
	for _, r := range runs {
		if !r.Rejected {
			valid = append(valid, r)
		}
	}
	if len(valid) == 0 {
		// Fallback: lowest score among all
		best := &runs[0]
		for i := 1; i < len(runs); i++ {
			if runs[i].Score < best.Score || (runs[i].Score == best.Score && runs[i].Variant < best.Variant) {
				best = &runs[i]
			}
		}
		return best
	}
	sort.Slice(valid, func(i, j int) bool {
		if valid[i].Score != valid[j].Score {
			return valid[i].Score < valid[j].Score
		}
		return valid[i].Variant < valid[j].Variant
	})
	return &valid[0]
}
