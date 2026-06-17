package mcp

import (
	"fmt"
	"strings"

	"control-plane/internal/app"
	"control-plane/internal/formation"
)

// MemoryFormationPolicy gates mcp_episode_ingest (low-noise episodic capture). Nil uses defaults.
type MemoryFormationPolicy struct {
	// EpisodeIngestEnabled when false rejects the tool with a clear message (server may still POST advisory via REST).
	EpisodeIngestEnabled bool
	// MinSummaryChars minimum trimmed summary length (default 12; low-friction ingest).
	MinSummaryChars int
	// RequireSignalKeyword when true, summary must contain at least one distillation-oriented token (default false).
	RequireSignalKeyword bool
}

// DefaultMemoryFormationPolicy returns defaults tuned for low-friction episodic capture (agents should not fight the gate).
// Stricter behavior: set YAML require_signal_keyword: true and/or min_summary_chars higher.
func DefaultMemoryFormationPolicy() *MemoryFormationPolicy {
	return &MemoryFormationPolicy{
		EpisodeIngestEnabled: true,
		MinSummaryChars:      12,
		RequireSignalKeyword: false,
	}
}

// NormalizeMemoryFormation fills zero values from defaults.
func NormalizeMemoryFormation(p *MemoryFormationPolicy) *MemoryFormationPolicy {
	d := DefaultMemoryFormationPolicy()
	if p == nil {
		return d
	}
	out := *p
	if out.MinSummaryChars <= 0 {
		out.MinSummaryChars = d.MinSummaryChars
	}
	return &out
}

// distillSignalTokens are substrings that must appear (any) when RequireSignalKeyword is true.
// Deterministic; aligned with keyword distillation heuristics (not LLM).
var distillSignalTokens = []string{
	"fail", "error", "mistake", "regression", "incident", "outage", "retry", "violation",
	"block", "forbid", "must not", "never", "always", "constraint", "decision", "chose",
	"fixed", "corrected", "learned", "rollback", "timeout", "denied", "override", "risk",
}

// ValidateMcpEpisodeSummary enforces low-noise rules before POST /v1/advisory-episodes.
func ValidateMcpEpisodeSummary(summary string, p *MemoryFormationPolicy) error {
	return ValidateMcpEpisodeSummaryWithGate(summary, p, nil)
}

// ValidateMcpEpisodeSummaryWithGate enforces MCP and shared formation junk rules (Phase 5 parity).
func ValidateMcpEpisodeSummaryWithGate(summary string, p *MemoryFormationPolicy, gate *formation.Gate) error {
	pol := NormalizeMemoryFormation(p)
	if !pol.EpisodeIngestEnabled {
		return fmt.Errorf("MCP episodic ingest is disabled in server config (mcp.memory_formation.episode_ingest_enabled)")
	}
	s := strings.TrimSpace(summary)
	minRunes := pol.MinSummaryChars
	if minRunes <= 0 {
		minRunes = 12
	}
	if err := formation.ValidateAdvisorySummaryShared(s, minRunes, gate); err != nil {
		return err
	}
	if !pol.RequireSignalKeyword {
		return nil
	}
	low := strings.ToLower(s)
	for _, tok := range distillSignalTokens {
		if strings.Contains(low, tok) {
			return nil
		}
	}
	return fmt.Errorf("summary must contain at least one learning signal token (e.g. failure, decision, constraint, fix); refusing noisy episodic ingest")
}

// PolicyFromAppConfig maps app.MCP.MemoryFormation to MemoryFormationPolicy (nil cfg → defaults).
func PolicyFromAppConfig(cfg *app.Config) *MemoryFormationPolicy {
	p := DefaultMemoryFormationPolicy()
	if cfg == nil || cfg.MCP == nil || cfg.MCP.MemoryFormation == nil {
		return p
	}
	mf := cfg.MCP.MemoryFormation
	if mf.EpisodeIngestEnabled != nil {
		p.EpisodeIngestEnabled = *mf.EpisodeIngestEnabled
	}
	if mf.MinSummaryChars > 0 {
		p.MinSummaryChars = mf.MinSummaryChars
	}
	if mf.RequireSignalKeyword != nil {
		p.RequireSignalKeyword = *mf.RequireSignalKeyword
	}
	return p
}
