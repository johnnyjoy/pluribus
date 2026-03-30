package curation

import (
	"fmt"
	"strings"

	"control-plane/pkg/api"
)

// Readiness levels for candidate promotion assistance (deterministic; not ML).
const (
	ReadinessNotReady            = "not_ready"
	ReadinessReviewRecommended   = "review_recommended"
	ReadinessHighConfidence      = "high_confidence"
)

// ClassifyPromotionReadiness derives a human-readable readiness label and reason from proposal + salience.
func ClassifyPromotionReadiness(p *ProposalPayloadV1, salience float64) (readiness, reason string) {
	if p == nil || strings.TrimSpace(p.Statement) == "" {
		return ReadinessNotReady, "Missing proposal or statement."
	}
	stmt := strings.TrimSpace(p.Statement)
	if len([]rune(stmt)) < 12 {
		return ReadinessNotReady, "Statement is too short for reliable promotion."
	}
	n := supportCount(p)
	if salience < 0.35 {
		return ReadinessNotReady, "Salience is below threshold for safe review."
	}
	if p.Kind == api.MemoryKindConstraint {
		return ReadinessReviewRecommended, "Constraints require explicit human review before promotion."
	}
	if n >= 4 && salience >= 0.65 && (p.Kind == api.MemoryKindFailure || p.Kind == api.MemoryKindPattern) && len([]rune(stmt)) >= 24 {
		return ReadinessHighConfidence, fmt.Sprintf("Observed across %d merged supports with consistent wording; salience %.2f; kind %s is suitable for accelerated review.", n, salience, p.Kind)
	}
	if p.Kind == api.MemoryKindDecision && n >= 3 && salience >= 0.7 && len([]rune(stmt)) >= 24 {
		return ReadinessHighConfidence, fmt.Sprintf("Decision candidate with strong support (%d) and salience %.2f.", n, salience)
	}
	if n >= 2 && salience >= 0.5 {
		return ReadinessReviewRecommended, "Adequate support and salience; manual review is recommended."
	}
	return ReadinessNotReady, "Support or salience is insufficient for confident promotion."
}
