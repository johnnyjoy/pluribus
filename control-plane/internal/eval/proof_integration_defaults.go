package eval

import (
	"control-plane/internal/app"
	"control-plane/internal/formation"
	"control-plane/internal/memory"
	"control-plane/internal/utility"
)

// ApplyProofIntegrationDefaults configures the control plane for REST proof harness runs.
// Proof scenarios assume seeded memories are immediately active in recall (not formation-pending)
// and that duplicate-write authority reinforce is enabled where scenarios assert +1 bumps.
func ApplyProofIntegrationDefaults(cfg *app.Config) {
	if cfg == nil {
		return
	}
	cfg.Similarity.Enabled = app.BoolPtr(true)
	cfg.Similarity.MinResemblance = 0.05
	cfg.Distillation.Enabled = true
	cfg.Enforcement.MinBindingAuthority = 3

	if cfg.Recall.SemanticRetrieval == nil {
		cfg.Recall.SemanticRetrieval = &memory.SemanticRetrievalConfig{}
	}
	cfg.Recall.SemanticRetrieval.Enabled = app.BoolPtr(true)

	if cfg.Memory.Formation == nil {
		d := formation.DefaultConfig()
		cfg.Memory.Formation = &d
	}
	dc := cfg.Memory.Formation.DirectCreate
	dc.MaxClientAuthority = 10
	dc.HighRiskAuthorityThreshold = 11
	dc.GoverningDefaultStatus = "active"
	dc.RequireProvenanceForGoverning = false
	dc.RequireAdminForGoverning = false
	dc.RequireProvenanceAuthorityGE = 11
	dc.AllowActiveHighRiskGoverning = true
	dc.AllowActiveWithQualityWarnings = true
	dc.AllowActiveDespiteQualityReview = true
	dc.AllowActivePromotedGoverning = true
	cfg.Memory.Formation.DirectCreate = dc
	cfg.Memory.Formation.Normalize()

	if cfg.Memory.Utility == nil {
		cfg.Memory.Utility = &utility.Config{}
	}
	cfg.Memory.Utility.ReinforceDuplicateAuthority = app.BoolPtr(true)
}
