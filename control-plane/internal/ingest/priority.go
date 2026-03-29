package ingest

import (
	"database/sql"
	"math"
	"time"
)

// PriorityFormulaVersion identifies the weighted formula in API/debug (deterministic replay).
const PriorityFormulaVersion = "m6-v1"

// PriorityWeights defines the blend for:
//
//	priority_score = WSignal*signal + WFreq*frequency + WRecency*recency + WAgreement*agreement
//
// - signal: final fact confidence after reinforce, clamped [0,1]
// - frequency: min(1, prior_row_count / 10) for this (project, normalized_hash) before insert
// - recency: 1.0 if first occurrence; else exp(-hours_since_last_duplicate / 168) (168h = 1 week scale)
// - agreement: client trust_weight clamped [0,1] (from client_profiles; default 1.0)
//
// Sum of weights should be 1.0; zero Service.PriorityWeights uses DefaultPriorityWeights().
type PriorityWeights struct {
	WSignal    float64
	WFreq      float64
	WRecency   float64
	WAgreement float64
}

// DefaultPriorityWeights is the documented default (sum = 1.0).
func DefaultPriorityWeights() PriorityWeights {
	return PriorityWeights{
		WSignal:    0.35,
		WFreq:      0.25,
		WRecency:   0.25,
		WAgreement: 0.15,
	}
}

func normalizeWeights(w PriorityWeights) PriorityWeights {
	if w.WSignal == 0 && w.WFreq == 0 && w.WRecency == 0 && w.WAgreement == 0 {
		return DefaultPriorityWeights()
	}
	sum := w.WSignal + w.WFreq + w.WRecency + w.WAgreement
	if sum <= 0 {
		return DefaultPriorityWeights()
	}
	return PriorityWeights{
		WSignal:    w.WSignal / sum,
		WFreq:      w.WFreq / sum,
		WRecency:   w.WRecency / sum,
		WAgreement: w.WAgreement / sum,
	}
}

// ComputePriorityScore returns a score in [0,1].
func ComputePriorityScore(confidence float64, priorCount int64, lastSeen sql.NullTime, now time.Time, trust float64, w PriorityWeights) float64 {
	w = normalizeWeights(w)
	signal := math.Min(1, math.Max(0, confidence))
	freq := math.Min(1.0, float64(priorCount)/10.0)
	recency := 1.0
	if lastSeen.Valid {
		h := now.Sub(lastSeen.Time).Hours()
		if h < 0 {
			h = 0
		}
		recency = math.Exp(-h / 168.0)
	}
	agree := trust
	if agree > 1 {
		agree = 1
	}
	if agree < 0 {
		agree = 0
	}
	s := w.WSignal*signal + w.WFreq*freq + w.WRecency*recency + w.WAgreement*agree
	return math.Min(1.0, math.Max(0, s))
}

func priorityWeightsDebugMap(w PriorityWeights) map[string]float64 {
	w = normalizeWeights(w)
	return map[string]float64{
		"w_signal":    w.WSignal,
		"w_frequency": w.WFreq,
		"w_recency":   w.WRecency,
		"w_agreement": w.WAgreement,
	}
}
