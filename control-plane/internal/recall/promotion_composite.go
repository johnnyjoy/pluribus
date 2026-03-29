package recall

// NormalizedPolicyWeights returns blend weights for PolicyComposite (sum to 1). All zero => 0.4 / 0.3 / 0.3.
func NormalizedPolicyWeights(pol PromotionPolicy) (wc, ws, we float64) {
	wc, ws, we = pol.WeightConfidence, pol.WeightSignal, pol.WeightEvidence
	if wc <= 0 && ws <= 0 && we <= 0 {
		wc, ws, we = 0.4, 0.3, 0.3
	}
	sum := wc + ws + we
	if sum <= 0 {
		return 0, 0, 0
	}
	return wc / sum, ws / sum, we / sum
}

// PolicyCompositeUsesEvidenceTerm is true when MinPolicyComposite is on and the normalized evidence weight > 0.
func PolicyCompositeUsesEvidenceTerm(pol PromotionPolicy) bool {
	_, _, we := NormalizedPolicyWeights(pol)
	return pol.MinPolicyComposite > 0 && we > 0
}

// PolicyComposite blends run confidence, normalized merge total_signal, and optional evidence average.
// Weights default to 0.4 / 0.3 / 0.3 when all zero. total_signal is divided by SignalNormDivisor (default 15) and capped to [0,1].
// Missing evidence (nil evidenceAvg) contributes 0 to the evidence term.
func PolicyComposite(runConfidence, totalSignal float64, evidenceAvg *float64, pol PromotionPolicy) float64 {
	wc, ws, we := NormalizedPolicyWeights(pol)

	div := pol.SignalNormDivisor
	if div <= 0 {
		div = 15
	}
	sigN := totalSignal / div
	if sigN < 0 {
		sigN = 0
	}
	if sigN > 1 {
		sigN = 1
	}

	ev := 0.0
	if evidenceAvg != nil {
		ev = *evidenceAvg
		if ev < 0 {
			ev = 0
		}
		if ev > 1 {
			ev = 1
		}
	}

	cc := runConfidence
	if cc < 0 {
		cc = 0
	}
	if cc > 1 {
		cc = 1
	}

	return wc*cc + ws*sigN + we*ev
}
