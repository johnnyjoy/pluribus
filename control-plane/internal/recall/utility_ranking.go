package recall

// utilityRankingTerm converts bounded utility score to additive recall ranking term.
// utility_score in [-10,10], weight typically 0.12.
func utilityRankingTerm(utilityScore, weight float64) float64 {
	if weight <= 0 {
		return 0
	}
	const maxScore = 10.0
	norm := utilityScore / maxScore
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
