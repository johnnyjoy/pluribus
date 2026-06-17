package utilitypolicy

import (
	"strings"
	"time"

	"control-plane/internal/utility"

	"github.com/google/uuid"
)

// EvaluatePolicy decides policy outcome without mutating scores.
func EvaluatePolicy(c CandidateInput, cfg PolicyConfig, ctx PolicyContext) PolicyDecision {
	class, evidence := ClassifyCandidate(c, cfg)
	dec := PolicyDecision{
		CandidateID:    c.CandidateID.String(),
		MemoryID:       c.MemoryID,
		PolicyVersion:  PolicyVersion,
		Evidence:       evidence,
		Classification: class,
	}

	if class == ClassRejected {
		dec.Decision = DecisionReject
		dec.Reason = "candidate rejected by classifier"
		return dec
	}

	if strings.TrimSpace(c.MemoryID) == "" && class == ClassPositiveApplyable {
		dec.Decision = DecisionReject
		dec.Reason = "positive apply requires memory_id"
		dec.Evidence = append(dec.Evidence, "missing memory_id")
		return dec
	}

	if !c.CreatedAt.IsZero() && cfg.StaleCandidateWindow > 0 {
		if time.Since(c.CreatedAt) > cfg.StaleCandidateWindow {
			dec.Decision = DecisionReviewRequired
			dec.Reason = "candidate stale beyond policy window"
			dec.Evidence = append(dec.Evidence, "stale")
			return dec
		}
	}

	if ctx.AlreadyApplied {
		dec.Decision = DecisionReject
		dec.Reason = "candidate already applied"
		dec.Evidence = append(dec.Evidence, "duplicate_apply")
		return dec
	}

	if ctx.Tampered {
		dec.Decision = DecisionReject
		dec.Reason = "candidate failed integrity check"
		dec.Evidence = append(dec.Evidence, "tampered")
		return dec
	}

	switch class {
	case ClassPositiveApplyable:
		if !c.SafeToApply {
			dec.Decision = DecisionReviewRequired
			dec.Reason = "positive signal requires safe_to_apply"
			return dec
		}
		if c.SessionID == uuid.Nil {
			dec.Decision = DecisionReviewRequired
			dec.Reason = "positive apply requires session_id"
			return dec
		}
		if ctx.SessionPositiveCount >= cfg.MaxPositivePerSessionPerMemory {
			dec.Decision = DecisionReject
			dec.Reason = "session positive apply cap exceeded"
			dec.Evidence = append(dec.Evidence, "session_cap")
			return dec
		}
		if ctx.AgentPositiveCount >= cfg.MaxPositivePerAgentPerMemory {
			dec.Decision = DecisionReject
			dec.Reason = "agent positive apply cap exceeded"
			dec.Evidence = append(dec.Evidence, "agent_cap")
			return dec
		}
		delta := positiveDelta(c, cfg)
		dec.Decision = DecisionApplyPositive
		dec.Delta = delta
		dec.Reason = "evaluator-validated positive utility"
		dec.RollbackToken = uuid.New().String()
		return dec

	case ClassNegativeApplyable:
		delta := negativeDelta(c, cfg)
		dec.Decision = DecisionApplyNegative
		dec.Delta = -delta
		dec.Reason = "evaluator-validated negative utility"
		dec.RollbackToken = uuid.New().String()
		return dec

	case ClassNeutralRecordOnly:
		dec.Decision = DecisionRecordOnly
		dec.Delta = 0
		dec.Reason = "neutral signal recorded without mutation"
		return dec

	case ClassReviewRequired:
		dec.Decision = DecisionReviewRequired
		dec.Delta = 0
		dec.Reason = "requires human review before mutation"
		return dec

	default:
		dec.Decision = DecisionReject
		dec.Reason = "unclassified candidate"
		return dec
	}
}

// PolicyContext carries runtime gates for evaluation.
type PolicyContext struct {
	AlreadyApplied       bool
	Tampered             bool
	SessionPositiveCount int
	AgentPositiveCount   int
}

func positiveDelta(c CandidateInput, cfg PolicyConfig) float64 {
	strength := c.SignalStrength
	if strength <= 0 {
		strength = 0.25
	}
	delta := strength * 0.5
	if delta > cfg.MaxPositiveDelta {
		delta = cfg.MaxPositiveDelta
	}
	if delta < 0.1 {
		delta = 0.1
	}
	return delta
}

func negativeDelta(c CandidateInput, cfg PolicyConfig) float64 {
	strength := c.SignalStrength
	if strength < 0 {
		strength = -strength
	}
	if strength <= 0 {
		strength = 0.5
	}
	delta := strength
	if delta > cfg.MaxNegativeDelta {
		delta = cfg.MaxNegativeDelta
	}
	if delta < 0.25 {
		delta = 0.25
	}
	return delta
}

// ApplyScoreBounds clamps a score to utility bounds.
func ApplyScoreBounds(score float64) float64 {
	return utility.ClampUtilityScore(score)
}

// ComputeNewScore applies delta with bounds.
func ComputeNewScore(prev, delta float64) float64 {
	return utility.ClampUtilityScore(utility.ApplyScoreDelta(prev, delta))
}
