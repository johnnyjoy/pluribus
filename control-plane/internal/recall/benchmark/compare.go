package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ComparisonReport compares lexical vs hybrid benchmark runs.
type ComparisonReport struct {
	GeneratedAt              time.Time `json:"generated_at"`
	Note                     string    `json:"note"`
	Lexical                  ModeSummary `json:"lexical"`
	Hybrid                   ModeSummary `json:"hybrid"`
	HybridDeltaVsLexical     DeltaSummary `json:"hybrid_delta_vs_lexical"`
}

// ModeSummary aggregates one retrieval mode run.
type ModeSummary struct {
	RetrievalMode            string  `json:"retrieval_mode"`
	CaseCount                int     `json:"case_count"`
	PassedCount              int     `json:"passed_count"`
	FailedCount              int     `json:"failed_count"`
	OverallRecallAtK         float64 `json:"overall_recall_at_k"`
	OverallPrecisionAtK      float64 `json:"overall_precision_at_k"`
	ForbiddenHitRate         float64 `json:"forbidden_hit_rate"`
	LifecycleViolationRate   float64 `json:"lifecycle_violation_rate"`
	DateBoundViolationRate   float64 `json:"date_bound_violation_rate"`
	UtilityViolationRate     float64 `json:"utility_violation_rate"`
	ModeViolationRate        float64 `json:"mode_violation_rate"`
	SemanticFallbackRate     float64 `json:"semantic_fallback_rate"`
}

// DeltaSummary is hybrid minus lexical (negative forbidden/lifecycle deltas are good).
type DeltaSummary struct {
	RecallAtK              float64 `json:"recall_at_k"`
	PrecisionAtK           float64 `json:"precision_at_k"`
	ForbiddenHitRate       float64 `json:"forbidden_hit_rate"`
	LifecycleViolationRate float64 `json:"lifecycle_violation_rate"`
	DateBoundViolationRate float64 `json:"date_bound_violation_rate"`
	UtilityViolationRate   float64 `json:"utility_violation_rate"`
}

// SummarizeMode builds ModeSummary from case results.
func SummarizeMode(mode string, results []CaseResult) ModeSummary {
	sum := Summarize(results)
	ms := ModeSummary{
		RetrievalMode:       mode,
		CaseCount:           sum.CaseCount,
		PassedCount:         sum.PassedCount,
		FailedCount:         sum.FailedCount,
		OverallRecallAtK:    sum.OverallRecallAtK,
		OverallPrecisionAtK: sum.OverallPrecisionAtK,
		ForbiddenHitRate:    sum.ForbiddenHitRate,
	}
	var lifeSum, dateSum, utilSum, modeSum, fallbackSum float64
	for _, r := range results {
		v := r.Metrics.Violations
		lifeSum += v.LifecycleViolationRate
		dateSum += v.DateBoundViolationRate
		utilSum += v.UtilityViolationRate
		modeSum += v.ModeViolationRate
		if r.Semantic != nil && r.Semantic.FallbackReason != "" {
			fallbackSum += 1
		}
	}
	n := float64(len(results))
	if n > 0 {
		ms.LifecycleViolationRate = lifeSum / n
		ms.DateBoundViolationRate = dateSum / n
		ms.UtilityViolationRate = utilSum / n
		ms.ModeViolationRate = modeSum / n
		ms.SemanticFallbackRate = fallbackSum / n
	}
	return ms
}

// BuildComparisonReport compares lexical and hybrid result sets on the same cases.
func BuildComparisonReport(lexical, hybrid []CaseResult) ComparisonReport {
	lm := SummarizeMode(RetrievalModeLexical, lexical)
	hm := SummarizeMode(RetrievalModeHybrid, hybrid)
	return ComparisonReport{
		GeneratedAt: time.Now().UTC(),
		Note:        "Hybrid benchmark uses deterministic test embeddings, not a production embedding model.",
		Lexical:     lm,
		Hybrid:      hm,
		HybridDeltaVsLexical: DeltaSummary{
			RecallAtK:              hm.OverallRecallAtK - lm.OverallRecallAtK,
			PrecisionAtK:           hm.OverallPrecisionAtK - lm.OverallPrecisionAtK,
			ForbiddenHitRate:       hm.ForbiddenHitRate - lm.ForbiddenHitRate,
			LifecycleViolationRate: hm.LifecycleViolationRate - lm.LifecycleViolationRate,
			DateBoundViolationRate: hm.DateBoundViolationRate - lm.DateBoundViolationRate,
			UtilityViolationRate:   hm.UtilityViolationRate - lm.UtilityViolationRate,
		},
	}
}

// WriteComparisonJSON writes hybrid comparison artifact.
func WriteComparisonJSON(path string, rep ComparisonReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// GateHybridComparison enforces hybrid vs lexical thresholds for semantic-only cases.
func GateHybridComparison(rep ComparisonReport) error {
	h := rep.Hybrid
	l := rep.Lexical
	if h.LifecycleViolationRate > 0 {
		return fmt.Errorf("hybrid lifecycle_violation_rate=%.3f required 0", h.LifecycleViolationRate)
	}
	if h.DateBoundViolationRate > 0 {
		return fmt.Errorf("hybrid date_bound_violation_rate=%.3f required 0", h.DateBoundViolationRate)
	}
	if h.UtilityViolationRate > 0 {
		return fmt.Errorf("hybrid utility_violation_rate=%.3f required 0", h.UtilityViolationRate)
	}
	if h.ModeViolationRate > 0 {
		return fmt.Errorf("hybrid mode_violation_rate=%.3f required 0", h.ModeViolationRate)
	}
	if h.ForbiddenHitRate > l.ForbiddenHitRate+1e-9 {
		return fmt.Errorf("hybrid forbidden_hit_rate=%.3f exceeds lexical=%.3f", h.ForbiddenHitRate, l.ForbiddenHitRate)
	}
	if h.OverallRecallAtK+1e-9 < l.OverallRecallAtK {
		return fmt.Errorf("hybrid recall=%.3f below lexical=%.3f", h.OverallRecallAtK, l.OverallRecallAtK)
	}
	if h.OverallPrecisionAtK+1e-9 < l.OverallPrecisionAtK-0.02 {
		return fmt.Errorf("hybrid precision=%.3f below lexical-0.02=%.3f", h.OverallPrecisionAtK, l.OverallPrecisionAtK-0.02)
	}
	return nil
}
