package recall

// stable rejection_code values for debug.promotion_decision (run-multi).
// See memory-bank/creative/creative-hive-phase9-promotion-hardening.md
const (
	RejectionCodeOK                      = "ok"
	RejectionCodePromoteNotAttempted     = "promote_not_attempted"
	RejectionCodeMergeRequired           = "merge_required"
	RejectionCodeMergeEmpty              = "merge_empty"
	RejectionCodeMergeFallback           = "merge_fallback"
	RejectionCodeMergeDrift              = "merge_drift"
	RejectionCodeBehaviorValidation      = "behavior_validation"
	RejectionCodeSignalLow               = "signal_low"
	RejectionCodeConfidenceBelowMinimum  = "confidence_below_minimum"
	RejectionCodePromoterUnconfigured    = "promoter_unconfigured"
	RejectionCodePromoteFailed           = "promote_failed"
	RejectionCodePromoteDeclined         = "promote_declined"
	// Evidence gates (run-multi / promote).
	RejectionCodeEvidenceRequired    = "evidence_required"
	RejectionCodeEvidenceInvalid     = "evidence_invalid"
	RejectionCodeEvidenceScoreLow    = "evidence_score_low"
	RejectionCodeEvidenceUnavailable = "evidence_policy_unavailable"
	// Optional weighted composite gate.
	RejectionCodePolicyCompositeLow = "policy_composite_low"
)

// PromotionPolicy holds promotion gates loaded from config (recall.Service).
// Zero value = same behavior as pre–Phase 9 (no extra gates).
type PromotionPolicy struct {
	// RequireEvidence when true requires linked evidence before promote (enforced in Phase 9.2+).
	RequireEvidence bool
	// MinEvidenceLinks minimum count of evidence links after promote (Phase 9.2+).
	MinEvidenceLinks int
	// MinEvidenceScore minimum aggregate evidence score (Phase 9.2+; 0 = off).
	MinEvidenceScore float64
	// RequireReview when true creates promoted memories in pending review (Phase 9.4+).
	RequireReview bool
	// MinPromoteConfidence in [0,1]. When > 0, run-multi out.Confidence must be >= this to promote.
	MinPromoteConfidence float64
	// MinPolicyComposite in [0,1]. When > 0, PolicyComposite(...) must be >= this. Default 0 = off.
	MinPolicyComposite float64
	// WeightConfidence, WeightSignal, WeightEvidence — blend weights for PolicyComposite; all zero => 0.4/0.3/0.3.
	WeightConfidence float64
	WeightSignal     float64
	WeightEvidence   float64
	// SignalNormDivisor divides merge total_signal before capping to [0,1]. Default 15 when 0.
	SignalNormDivisor float64
}

func (s *Service) promotionPolicy() PromotionPolicy {
	if s == nil || s.Promotion == nil {
		return PromotionPolicy{}
	}
	return *s.Promotion
}
