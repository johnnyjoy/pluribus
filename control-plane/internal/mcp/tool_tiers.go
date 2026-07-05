package mcp

import (
	"os"
	"strings"
)

const (
	ToolsTierAll      = "all"
	ToolsTierStandard = "standard"
	ToolsTierCore     = "core"
)

var activeToolsTier = ToolsTierAll

// SetToolsTier configures which tools appear in tools/list (tools/call still accepts all names).
func SetToolsTier(tier string) {
	switch strings.ToLower(strings.TrimSpace(tier)) {
	case ToolsTierCore:
		activeToolsTier = ToolsTierCore
	case ToolsTierStandard:
		activeToolsTier = ToolsTierStandard
	default:
		activeToolsTier = ToolsTierAll
	}
}

// ActiveToolsTier returns the configured tools/list tier.
func ActiveToolsTier() string {
	return activeToolsTier
}

// InitToolsTier applies PLURIBUS_TOOLS env override, then config default.
func InitToolsTier(configTier string) {
	if v := strings.TrimSpace(os.Getenv("PLURIBUS_TOOLS")); v != "" {
		SetToolsTier(v)
		return
	}
	if strings.TrimSpace(configTier) != "" {
		SetToolsTier(configTier)
		return
	}
	SetToolsTier(ToolsTierAll)
}

var coreToolNames = map[string]struct{}{
	"wakeup_context":         {},
	"recall_context":         {},
	"memory_context_resolve": {},
	"record_experience":      {},
	"mcp_episode_ingest":     {},
	"enforcement_evaluate":   {},
	"list_chores":            {},
	"resolve_chore":          {},
	"memory_feedback":        {},
	"health":                 {},
}

var standardExtraToolNames = map[string]struct{}{
	"memory_log_if_relevant":       {},
	"auto_log_episode_if_relevant": {},
	"memory_create":                {},
	"memory_promote":               {},
	"memory_quarantine":            {},
	"memory_delete":                {},
	"curation_digest":              {},
	"curation_pending":             {},
	"curation_promotion_suggestions": {},
	"curation_strengthened":        {},
	"curation_review_candidate":    {},
	"curation_materialize":         {},
	"curation_promote_candidate":   {},
	"curation_reject_candidate":    {},
	"curation_auto_promote":        {},
}

func toolAllowedInTier(name, tier string) bool {
	switch tier {
	case ToolsTierAll:
		return true
	case ToolsTierCore:
		_, ok := coreToolNames[name]
		return ok
	case ToolsTierStandard:
		if _, ok := coreToolNames[name]; ok {
			return true
		}
		_, ok := standardExtraToolNames[name]
		return ok
	default:
		return true
	}
}

func filterRegistryByTier(reg []ToolSpec, tier string) []ToolSpec {
	if tier == ToolsTierAll || tier == "" {
		return reg
	}
	out := make([]ToolSpec, 0, len(reg))
	for _, t := range reg {
		if toolAllowedInTier(t.Name, tier) {
			out = append(out, t)
		}
	}
	return out
}

func registryForList() []ToolSpec {
	return filterRegistryByTier(toolRegistry(), activeToolsTier)
}
