package ingest

import "math"

// ReinforceDelta is added to the peak confidence when the same normalized fact
// When the same normalized_hash is seen again. Capped at 1.0. Deterministic.
const ReinforceDelta = 0.1

// ApplyReinforce returns confidence after duplicate reinforcement.
// priorPeak is MAX(confidence) for existing extractions with the same hash in the project.
// incoming is this extraction's client-supplied confidence (before reinforce).
func ApplyReinforce(priorPeak, incoming float64) float64 {
	base := priorPeak
	if incoming > base {
		base = incoming
	}
	out := base + ReinforceDelta
	return math.Min(1.0, out)
}

func reinforceMergeAction(normalizedHash string, sourceIndex int, priorPeak, incoming, resulting float64, duplicateCount int64) map[string]interface{} {
	return map[string]interface{}{
		"action":               "reinforce",
		"normalized_hash":      normalizedHash,
		"source_index":         sourceIndex,
		"prior_peak":           priorPeak,
		"incoming_confidence":  incoming,
		"resulting_confidence": resulting,
		"prior_row_count":      duplicateCount,
	}
}
