package utilitypolicy

import "strings"

// ClassifyCandidate maps evaluator-validated signals to policy classes.
func ClassifyCandidate(c CandidateInput, cfg PolicyConfig) (class string, evidence []string) {
	sig := strings.ToLower(strings.TrimSpace(c.SignalType))
	evidence = append(evidence, "signal_type="+sig)

	switch sig {
	case "recall_only":
		return ClassRejected, append(evidence, "recall is not usefulness")
	case "self_report_only":
		return ClassRejected, append(evidence, "self-report is not usefulness")
	case "citation_only":
		return ClassRejected, append(evidence, "citation alone is not usefulness")
	case "missing_evaluation", "":
		if c.EvaluationID.String() == "00000000-0000-0000-0000-000000000000" {
			return ClassRejected, append(evidence, "missing evaluation")
		}
	case "failed_evaluation":
		if !c.EvaluatorPassed {
			return ClassNegativeApplyable, append(evidence, "failed evaluation negative")
		}
		return ClassReviewRequired, append(evidence, "failed evaluation requires review")
	}

	if c.EvaluationID.String() == "00000000-0000-0000-0000-000000000000" {
		return ClassRejected, append(evidence, "missing evaluation_id")
	}

	for _, vc := range c.ViolationCodes {
		switch strings.ToLower(vc) {
		case "refuted_used", "refuted_memory_used":
			return ClassNegativeApplyable, append(evidence, "refuted use is negative evidence")
		case "superseded_used", "superseded_memory_used":
			return ClassNegativeApplyable, append(evidence, "superseded use is negative evidence")
		case "wrong_scope_used":
			return ClassNegativeApplyable, append(evidence, "wrong scope use is negative evidence")
		case "negative_scope_used":
			return ClassNegativeApplyable, append(evidence, "negative scope use is negative evidence")
		}
	}

	switch sig {
	case "refuted_used", "superseded_used", "wrong_scope_used", "negative_scope_used":
		return ClassNegativeApplyable, append(evidence, "misuse signal")
	case "used_correctly":
		if !c.EvaluatorPassed {
			return ClassReviewRequired, append(evidence, "evaluator did not pass")
		}
		return ClassPositiveApplyable, append(evidence, "evaluator validated correct use")
	case "helped_output":
		if !c.EvaluatorPassed {
			return ClassReviewRequired, append(evidence, "helped_output without evaluator pass")
		}
		return ClassPositiveApplyable, append(evidence, "evaluator validated helpful output")
	case "misused", "unsafe_use", "harmed_output", "unsupported_claim":
		return ClassNegativeApplyable, append(evidence, "negative misuse signal")
	case "ignored_correctly":
		return ClassNeutralRecordOnly, append(evidence, "neutral ignored behavior")
	case "historical_only_correctly":
		if cfg.AllowHistoricalPositive && c.EvaluatorPassed && c.SafeToApply {
			return ClassPositiveApplyable, append(evidence, "historical positive explicitly configured")
		}
		return ClassNeutralRecordOnly, append(evidence, "historical-only is not current-guidance promotion")
	case "failed_evaluation":
		if !c.EvaluatorPassed {
			return ClassNegativeApplyable, append(evidence, "failed evaluation negative")
		}
		return ClassReviewRequired, append(evidence, "ambiguous failed_evaluation signal")
	default:
		return ClassRejected, append(evidence, "unrecognized signal type")
	}
}
