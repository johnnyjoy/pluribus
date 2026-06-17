package benchmark

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"control-plane/internal/memory"
)

// RealEmbedderReport is the Phase 10C live embedder benchmark artifact.
type RealEmbedderReport struct {
	GeneratedAt string `json:"generated_at"`
	Disclaimer  string `json:"disclaimer"`
	DeterministicNote string `json:"deterministic_note"`
	ProductionNote    string `json:"production_note"`
	Provider          string `json:"provider"`
	Model             string `json:"model"`
	Dimension         int    `json:"dimension"`
	ConfigSource      string `json:"config_source"`
	Lexical           *RealEmbedderModeSummary `json:"lexical"`
	LiveHybrid        *RealEmbedderModeSummary `json:"live_hybrid"`
	Comparison        *RealEmbedderComparison  `json:"comparison"`
}

type RealEmbedderModeSummary struct {
	Mode              string  `json:"mode"`
	CaseCount         int     `json:"case_count"`
	RecallAtK         float64 `json:"recall_at_k"`
	PrecisionAtK      float64 `json:"precision_at_k"`
	ForbiddenHitRate  float64 `json:"forbidden_hit_rate"`
	LifecycleViolRate float64 `json:"lifecycle_violation_rate"`
	DateBoundViolRate float64 `json:"date_bound_violation_rate"`
	UtilityViolRate   float64 `json:"utility_violation_rate"`
	ModeViolRate      float64 `json:"mode_violation_rate"`
	FallbackRate      float64 `json:"fallback_rate"`
	StaleEmbeddingCount int   `json:"stale_embedding_count"`
	MissingEmbeddingCount int `json:"missing_embedding_count"`
	LatencyP50Ms      float64 `json:"latency_p50_ms,omitempty"`
	LatencyP95Ms      float64 `json:"latency_p95_ms,omitempty"`
}

type RealEmbedderComparison struct {
	RecallDelta         float64 `json:"recall_delta"`
	PrecisionDelta      float64 `json:"precision_delta"`
	ForbiddenDelta      float64 `json:"forbidden_delta"`
	LifecycleDelta      float64 `json:"lifecycle_delta"`
	DateBoundDelta      float64 `json:"date_bound_delta"`
	UtilityDelta        float64 `json:"utility_delta"`
	ModeDelta           float64 `json:"mode_delta"`
	LivePassesThreshold bool    `json:"live_passes_threshold"`
}

// BuildRealEmbedderReport compares lexical vs live hybrid on semantic cases.
func BuildRealEmbedderReport(lexical, live []CaseResult, env memory.LiveEmbedderEnvConfig) *RealEmbedderReport {
	lexSum := summarizeRealMode(lexical, RetrievalModeLexical)
	liveSum := summarizeRealMode(live, RetrievalModeHybridLive)
	comp := compareRealModes(lexSum, liveSum)
	return &RealEmbedderReport{
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		Disclaimer:        "This benchmark used a real embedder.",
		DeterministicNote: "This benchmark is not deterministic unless the model/provider is deterministic.",
		ProductionNote:    "This benchmark does not enable production semantic recall.",
		Provider:          env.Provider,
		Model:             env.Model,
		Dimension:         env.Dimension,
		ConfigSource:      env.Source,
		Lexical:           lexSum,
		LiveHybrid:        liveSum,
		Comparison:        comp,
	}
}

func summarizeRealMode(results []CaseResult, mode string) *RealEmbedderModeSummary {
	if len(results) == 0 {
		return &RealEmbedderModeSummary{Mode: mode}
	}
	var recall, prec, forb, life, date, util, modeV float64
	var fallbackCases int
	var stale, missing int
	var latencies []float64
	for _, r := range results {
		recall += r.Metrics.RecallAtK
		prec += r.Metrics.PrecisionAtK
		forb += r.Metrics.ForbiddenHitRate
		v := r.Metrics.Violations
		life += v.LifecycleViolationRate
		date += v.DateBoundViolationRate
		util += v.UtilityViolationRate
		modeV += v.ModeViolationRate
		if r.Semantic != nil && r.Semantic.FallbackReason != "" {
			fallbackCases++
		}
		if r.LiveStats != nil {
			stale += r.LiveStats.StaleSkipCount
			missing += r.LiveStats.MissingCount
			if r.LiveStats.P50LatencyMs > 0 {
				latencies = append(latencies, r.LiveStats.P50LatencyMs)
			}
		}
	}
	n := float64(len(results))
	s := &RealEmbedderModeSummary{
		Mode:                  mode,
		CaseCount:             len(results),
		RecallAtK:             recall / n,
		PrecisionAtK:          prec / n,
		ForbiddenHitRate:      forb / n,
		LifecycleViolRate:     life / n,
		DateBoundViolRate:     date / n,
		UtilityViolRate:       util / n,
		ModeViolRate:          modeV / n,
		FallbackRate:          float64(fallbackCases) / n,
		StaleEmbeddingCount:   stale,
		MissingEmbeddingCount: missing,
	}
	if len(latencies) > 0 {
		s.LatencyP50Ms = percentile(latencies, 0.5)
		s.LatencyP95Ms = percentile(latencies, 0.95)
	}
	return s
}

func compareRealModes(lex, live *RealEmbedderModeSummary) *RealEmbedderComparison {
	if lex == nil || live == nil {
		return nil
	}
	passes := live.RecallAtK >= lex.RecallAtK &&
		live.ForbiddenHitRate <= lex.ForbiddenHitRate &&
		live.LifecycleViolRate == 0 &&
		live.DateBoundViolRate == 0 &&
		live.UtilityViolRate == 0 &&
		live.ModeViolRate == 0 &&
		live.StaleEmbeddingCount == 0 &&
		(live.PrecisionAtK >= lex.PrecisionAtK-0.02)
	return &RealEmbedderComparison{
		RecallDelta:         live.RecallAtK - lex.RecallAtK,
		PrecisionDelta:      live.PrecisionAtK - lex.PrecisionAtK,
		ForbiddenDelta:      live.ForbiddenHitRate - lex.ForbiddenHitRate,
		LifecycleDelta:      live.LifecycleViolRate - lex.LifecycleViolRate,
		DateBoundDelta:      live.DateBoundViolRate - lex.DateBoundViolRate,
		UtilityDelta:        live.UtilityViolRate - lex.UtilityViolRate,
		ModeDelta:           live.ModeViolRate - lex.ModeViolRate,
		LivePassesThreshold: passes,
	}
}

func percentile(vals []float64, p float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	cp := append([]float64(nil), vals...)
	for i := 0; i < len(cp); i++ {
		for j := i + 1; j < len(cp); j++ {
			if cp[j] < cp[i] {
				cp[i], cp[j] = cp[j], cp[i]
			}
		}
	}
	idx := int(float64(len(cp)-1) * p)
	if idx < 0 {
		idx = 0
	}
	return cp[idx]
}

// WriteRealEmbedderJSON writes the Phase 10C/10D live embedder artifact.
func WriteRealEmbedderJSON(report *RealEmbedderReport, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

// WriteLiveComparisonJSON writes Phase 10D lexical vs live-hybrid comparison artifact.
func WriteLiveComparisonJSON(report *RealEmbedderReport, path string) error {
	return WriteRealEmbedderJSON(report, path)
}

// DefaultLiveComparisonPath is the Phase 10D comparison artifact location.
func DefaultLiveComparisonPath(repoRoot string) string {
	return filepath.Join(repoRoot, "artifacts", "recall-benchmark-live-comparison.json")
}

// GateLiveHybridComparison enforces Phase 10C live hybrid thresholds vs lexical.
func GateLiveHybridComparison(lexical, live []CaseResult) error {
	report := BuildRealEmbedderReport(lexical, live, memory.LiveEmbedderEnvConfig{})
	if report.Comparison == nil {
		return fmt.Errorf("live hybrid comparison: missing summaries")
	}
	if !report.Comparison.LivePassesThreshold {
		return fmt.Errorf("live hybrid gate failed: recall_delta=%.3f precision_delta=%.3f forbidden_delta=%.3f stale=%d",
			report.Comparison.RecallDelta,
			report.Comparison.PrecisionDelta,
			report.Comparison.ForbiddenDelta,
			report.LiveHybrid.StaleEmbeddingCount,
		)
	}
	return nil
}
