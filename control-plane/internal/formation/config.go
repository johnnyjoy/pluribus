package formation

// DirectCreateConfig gates POST /v1/memory and MCP memory_create (high-risk path).
type DirectCreateConfig struct {
	Enabled                      bool   `yaml:"enabled"`
	RequireAdminForGoverning     bool   `yaml:"require_admin_for_governing"`
	MaxClientAuthority           int    `yaml:"max_client_authority"`
	GoverningDefaultStatus       string `yaml:"governing_default_status"` // pending
	RequireProvenanceForGoverning bool  `yaml:"require_provenance_for_governing"`
	RequireProvenanceAuthorityGE int    `yaml:"require_provenance_authority_ge"` // default 8
	RejectJunk                   bool   `yaml:"reject_junk"`
	HighRiskAuthorityThreshold   int    `yaml:"high_risk_authority_threshold"` // default 8
	// AllowActiveHighRiskGoverning when true permits governing direct creates to remain active
	// without review when other governing pending gates are relaxed (integration proof harness).
	AllowActiveHighRiskGoverning bool `yaml:"allow_active_high_risk_governing"`
	// AllowActiveWithQualityWarnings when true keeps direct creates active when quality
	// evaluation would otherwise accept_pending for warnings-only (missing schema_type, etc.).
	AllowActiveWithQualityWarnings bool `yaml:"allow_active_with_quality_warnings"`
	// AllowActiveDespiteQualityReview when true records formation quality on direct creates
	// but does not force pending for integration/proof harness runs (production default false).
	AllowActiveDespiteQualityReview bool `yaml:"allow_active_despite_quality_review"`
	// AllowActivePromotedGoverning when true permits materialize/promote path governing
	// constraints to remain active without forced pending review (proof/integration only).
	AllowActivePromotedGoverning bool `yaml:"allow_active_promoted_governing"`
}

// RecordExperienceConfig gates probationary memory from advisory ingest / record_experience.
type RecordExperienceConfig struct {
	RejectJunk                    bool `yaml:"reject_junk"`
	MinActionableWords            int  `yaml:"min_actionable_words"` // concrete content threshold
	RiskyConstraintPending        bool `yaml:"risky_constraint_pending"`
	MaxProbationaryAuthority      int  `yaml:"max_probationary_authority"` // cap even after qualification
}

// Config is the memory formation quality gate configuration.
type Config struct {
	// HiveDefaults enables agent-weighted ingest: active at low authority, ranking/utility curate later.
	// When true, ApplyHiveDefaults overlays direct_create and record_experience for hive operation.
	HiveDefaults         bool                   `yaml:"hive_defaults"`
	DirectCreate         DirectCreateConfig     `yaml:"direct_create"`
	RecordExperience     RecordExperienceConfig `yaml:"record_experience"`
	// ContradictionOnWrite when true runs lightweight negation check for high-risk governing writes.
	ContradictionOnWrite bool `yaml:"contradiction_on_write"`
}

// DefaultConfig returns shipped production defaults: hive profile (generous ingest, ruthless ranking).
func DefaultConfig() Config {
	c := WarehouseConfig()
	c.HiveDefaults = true
	c.ApplyHiveDefaults()
	return c
}

// WarehouseConfig returns explicit review-board / warehouse formation (pending governing defaults).
// Use in tests or operator configs that intentionally require human review before active recall.
func WarehouseConfig() Config {
	return Config{
		HiveDefaults: false,
		DirectCreate: DirectCreateConfig{
			Enabled:                       true,
			RequireAdminForGoverning:      true,
			MaxClientAuthority:            4,
			GoverningDefaultStatus:        "pending",
			RequireProvenanceForGoverning: true,
			RequireProvenanceAuthorityGE:  8,
			RejectJunk:                    true,
			HighRiskAuthorityThreshold:    8,
		},
		RecordExperience: RecordExperienceConfig{
			RejectJunk:               true,
			MinActionableWords:       4,
			RiskyConstraintPending:   true,
			MaxProbationaryAuthority: 2,
		},
		ContradictionOnWrite: true,
	}
}

// Normalize fills zero values from defaults.
func (c *Config) Normalize() {
	d := DefaultConfig()
	if c.DirectCreate.MaxClientAuthority <= 0 {
		c.DirectCreate.MaxClientAuthority = d.DirectCreate.MaxClientAuthority
	}
	if c.DirectCreate.GoverningDefaultStatus == "" {
		c.DirectCreate.GoverningDefaultStatus = d.DirectCreate.GoverningDefaultStatus
	}
	if c.DirectCreate.RequireProvenanceAuthorityGE <= 0 {
		c.DirectCreate.RequireProvenanceAuthorityGE = d.DirectCreate.RequireProvenanceAuthorityGE
	}
	if c.DirectCreate.HighRiskAuthorityThreshold <= 0 {
		c.DirectCreate.HighRiskAuthorityThreshold = d.DirectCreate.HighRiskAuthorityThreshold
	}
	if c.RecordExperience.MinActionableWords <= 0 {
		c.RecordExperience.MinActionableWords = d.RecordExperience.MinActionableWords
	}
	if c.RecordExperience.MaxProbationaryAuthority <= 0 {
		c.RecordExperience.MaxProbationaryAuthority = d.RecordExperience.MaxProbationaryAuthority
	}
	// Booleans default true when unset — DirectCreate.Enabled and RejectJunk use zero=false;
	// callers should use Gate which applies DefaultConfig when cfg is nil.
	if c.HiveDefaults {
		c.ApplyHiveDefaults()
	}
}

// HiveConfig returns the default agent-weighted formation profile (generous ingest, ruthless ranking).
func HiveConfig() Config {
	return DefaultConfig()
}

// ApplyHiveDefaults overlays hive ingest policy: memories enter active at capped authority;
// utility feedback and recall ranking demote weak signal over time.
func (c *Config) ApplyHiveDefaults() {
	if c == nil {
		return
	}
	c.HiveDefaults = true
	dc := &c.DirectCreate
	dc.Enabled = true
	dc.RejectJunk = true
	dc.GoverningDefaultStatus = "active"
	dc.RequireAdminForGoverning = false
	dc.RequireProvenanceForGoverning = false
	dc.RequireProvenanceAuthorityGE = 11
	dc.AllowActiveHighRiskGoverning = true
	dc.AllowActiveWithQualityWarnings = true
	dc.AllowActiveDespiteQualityReview = true
	dc.AllowActivePromotedGoverning = true
	if dc.MaxClientAuthority <= 0 {
		dc.MaxClientAuthority = DefaultConfig().DirectCreate.MaxClientAuthority
	}
	c.RecordExperience.RiskyConstraintPending = false
	if !c.RecordExperience.RejectJunk {
		c.RecordExperience.RejectJunk = true
	}
}
