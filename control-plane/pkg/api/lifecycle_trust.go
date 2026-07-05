package api

// PendingTrustDampener scales recall ranking and binding strength for rare pending rows.
const PendingTrustDampener = 0.88

// EffectiveBindingAuthority returns authority adjusted for lifecycle status (pending dampens).
func EffectiveBindingAuthority(authority int, status Status) float64 {
	if authority <= 0 {
		return 0
	}
	f := float64(authority)
	if status == StatusPending {
		f *= PendingTrustDampener
	}
	return f
}
