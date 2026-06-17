package utility

import "math"

// DefaultEventWeight maps event types to score deltas.
var DefaultEventWeight = map[string]float64{
	EventHelpful:    1.0,
	EventConfirmed:  2.0,
	EventIrrelevant: -0.5,
	EventOutdated:   -1.0,
	EventHarmful:    -2.0,
	EventWrong:      -3.0,
	EventRefuted:    -4.0,
	EventDuplicateSeen: 0.0,
}

// EventWeight returns configured weight for an event type.
func EventWeight(eventType string) float64 {
	if w, ok := DefaultEventWeight[eventType]; ok {
		return w
	}
	return 0
}

// ClampUtilityScore bounds utility to [-10, 10].
func ClampUtilityScore(v float64) float64 {
	if v < MinUtilityScore {
		return MinUtilityScore
	}
	if v > MaxUtilityScore {
		return MaxUtilityScore
	}
	return v
}

// RankingTerm converts utility score to a bounded additive recall ranking term.
// utility_score in [-10,10], weight typically 0.12 → max boost ~0.18, max penalty ~0.30.
func RankingTerm(utilityScore, weight float64) float64 {
	if weight <= 0 {
		return 0
	}
	norm := utilityScore / MaxUtilityScore // [-1, 1]
	term := weight * norm
	const maxBoost = 0.18
	const maxPenalty = 0.30
	if term > maxBoost {
		return maxBoost
	}
	if term < -maxPenalty {
		return -maxPenalty
	}
	return term
}

// IsPositiveEvent reports whether the type increments positive counters.
func IsPositiveEvent(eventType string) bool {
	switch eventType {
	case EventHelpful, EventConfirmed:
		return true
	default:
		return false
	}
}

// IsNegativeEvent reports whether the type increments negative counters.
func IsNegativeEvent(eventType string) bool {
	switch eventType {
	case EventHarmful, EventWrong, EventOutdated, EventIrrelevant, EventRefuted:
		return true
	default:
		return false
	}
}

// CountField increments the appropriate counter on Score for eventType.
func ApplyCountField(s *Score, eventType string) {
	if s == nil {
		return
	}
	switch eventType {
	case EventHelpful:
		s.HelpfulCount++
	case EventHarmful:
		s.HarmfulCount++
	case EventWrong:
		s.WrongCount++
	case EventOutdated:
		s.OutdatedCount++
	case EventIrrelevant:
		s.IrrelevantCount++
	case EventConfirmed:
		s.ConfirmedCount++
	case EventRefuted:
		s.RefutedCount++
	}
}

// DiminishingWeight reduces repeated same-type impact after the third event of that type.
func DiminishingWeight(baseWeight float64, sameTypeCount int) float64 {
	if sameTypeCount <= 3 {
		return baseWeight
	}
	factor := 1.0 / (1.0 + 0.25*float64(sameTypeCount-3))
	return baseWeight * factor
}

// ApplyScoreDelta updates utility score with bounded delta.
func ApplyScoreDelta(current, delta float64) float64 {
	return ClampUtilityScore(current + delta)
}

// UtilityMultiplier is an alternative helper (unused in Phase 7 ranking; additive term preferred).
func UtilityMultiplier(utilityScore float64) float64 {
	return 1.0 + math.Tanh(utilityScore/5.0)*0.15
}
