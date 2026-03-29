package signal

import (
	"control-plane/internal/merge"
)

// SignalConfig tunes high-signal detection (Phase 4).
type SignalConfig struct {
	MinAgreements         int
	MinTotalScore         float64
	MinUniqueRunes        int
	MinDistinctTokens     int
	AgreementOverlapTau   float64
	IntentOverlapTau      float64
}

// DefaultSignalConfig returns production-style defaults (tunable via tests).
func DefaultSignalConfig() SignalConfig {
	return SignalConfig{
		MinAgreements:       1,
		MinTotalScore:       4.0,
		MinUniqueRunes:      20,
		MinDistinctTokens:   4,
		AgreementOverlapTau: 0.15,
		IntentOverlapTau:    0.1,
	}
}

// NormalizeSignalConfig fills zero fields in c with defaults from DefaultSignalConfig.
func NormalizeSignalConfig(c SignalConfig) SignalConfig {
	d := DefaultSignalConfig()
	if c.MinAgreements <= 0 {
		c.MinAgreements = d.MinAgreements
	}
	if c.MinTotalScore <= 0 {
		c.MinTotalScore = d.MinTotalScore
	}
	if c.MinUniqueRunes <= 0 {
		c.MinUniqueRunes = d.MinUniqueRunes
	}
	if c.MinDistinctTokens <= 0 {
		c.MinDistinctTokens = d.MinDistinctTokens
	}
	if c.AgreementOverlapTau <= 0 {
		c.AgreementOverlapTau = d.AgreementOverlapTau
	}
	if c.IntentOverlapTau <= 0 {
		c.IntentOverlapTau = d.IntentOverlapTau
	}
	return c
}

func normalizeSignalConfig(c SignalConfig) SignalConfig {
	return NormalizeSignalConfig(c)
}

// IsHighSignal returns true when merge output is safe to promote (deterministic, no LLM).
func IsHighSignal(m merge.MergeResult, intent IntentText, cfg SignalConfig) bool {
	cfg = normalizeSignalConfig(cfg)
	if m.FallbackUsed {
		return false
	}
	if len(m.Drift.Violations) > 0 {
		return false
	}
	if len(m.Agreements) < cfg.MinAgreements {
		return false
	}
	filtered := FilterUniques(m.Unique, m.Agreements, intent, cfg)
	scores := SegmentScores(m, filtered)
	total := TotalSignal(scores)
	return total >= cfg.MinTotalScore
}
