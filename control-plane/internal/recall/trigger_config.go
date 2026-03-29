package recall

// TriggerRecallConfig gates triggered recall (YAML: recall.triggered_recall).
type TriggerRecallConfig struct {
	Enabled bool `yaml:"enabled" json:"enabled"`
	// MaxTriggersPerRequest caps how many trigger kinds apply per compile (default 2).
	MaxTriggersPerRequest int `yaml:"max_triggers_per_request" json:"max_triggers_per_request"`
	// MinContextTokens is minimum content tokens (after stopwords) before any trigger fires.
	MinContextTokens int `yaml:"min_context_tokens" json:"min_context_tokens"`
	EnableRisk       bool `yaml:"enable_risk" json:"enable_risk"`
	EnableDecision   bool `yaml:"enable_decision" json:"enable_decision"`
	EnableSimilarity bool `yaml:"enable_similarity" json:"enable_similarity"`
	// LargeChangeFilesThreshold: if ChangedFilesCount >= this, add risk trigger (0 = off).
	LargeChangeFilesThreshold int `yaml:"large_change_files_threshold" json:"large_change_files_threshold"`
}

// DefaultTriggerRecallConfig returns conservative defaults (enabled=false in config means off).
func DefaultTriggerRecallConfig() *TriggerRecallConfig {
	return &TriggerRecallConfig{
		MaxTriggersPerRequest:     2,
		MinContextTokens:        4,
		EnableRisk:                true,
		EnableDecision:            true,
		EnableSimilarity:        true,
		LargeChangeFilesThreshold: 25,
	}
}

// NormalizeTriggerRecall fills zero values; nil-safe.
func NormalizeTriggerRecall(c *TriggerRecallConfig) *TriggerRecallConfig {
	if c == nil {
		return nil
	}
	d := DefaultTriggerRecallConfig()
	out := *c
	if out.MaxTriggersPerRequest <= 0 {
		out.MaxTriggersPerRequest = d.MaxTriggersPerRequest
	}
	if out.MinContextTokens <= 0 {
		out.MinContextTokens = d.MinContextTokens
	}
	if out.LargeChangeFilesThreshold <= 0 {
		out.LargeChangeFilesThreshold = d.LargeChangeFilesThreshold
	}
	return &out
}

func filterTriggersByConfig(all []TriggerDecision, c *TriggerRecallConfig) []TriggerDecision {
	if c == nil {
		return nil
	}
	var out []TriggerDecision
	for _, t := range all {
		switch t.Kind {
		case TriggerKindRisk:
			if c.EnableRisk {
				out = append(out, t)
			}
		case TriggerKindDecision:
			if c.EnableDecision {
				out = append(out, t)
			}
		case TriggerKindSimilarity:
			if c.EnableSimilarity {
				out = append(out, t)
			}
		}
	}
	return dedupeTriggerKinds(out)
}

func capTriggers(triggers []TriggerDecision, max int) ([]TriggerDecision, string) {
	if max <= 0 {
		max = 2
	}
	if len(triggers) <= max {
		return triggers, ""
	}
	return triggers[:max], "max_triggers_per_request"
}
