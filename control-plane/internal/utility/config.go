package utility

// Config gates Phase 7 utility behavior (YAML: memory.utility).
type Config struct {
	// ReinforceOnRecall when true restores legacy recall-compile authority bumps. Default false.
	ReinforceOnRecall *bool `yaml:"reinforce_on_recall,omitempty"`
	// ReinforceDuplicateAuthority when true restores duplicate write authority +1. Default false.
	ReinforceDuplicateAuthority *bool `yaml:"reinforce_duplicate_authority,omitempty"`
	// UtilityRankingWeight scales utility score in recall ranking (default 0.12).
	UtilityRankingWeight float64 `yaml:"utility_ranking_weight,omitempty"`
}

// ReinforceOnRecallEnabled returns false when nil or explicitly false.
func (c *Config) ReinforceOnRecallEnabled() bool {
	if c == nil || c.ReinforceOnRecall == nil {
		return false
	}
	return *c.ReinforceOnRecall
}

// ReinforceDuplicateAuthorityEnabled returns false when nil or explicitly false.
func (c *Config) ReinforceDuplicateAuthorityEnabled() bool {
	if c == nil || c.ReinforceDuplicateAuthority == nil {
		return false
	}
	return *c.ReinforceDuplicateAuthority
}

// RankingWeight returns bounded utility ranking weight (default 0.12).
func (c *Config) RankingWeight() float64 {
	if c == nil || c.UtilityRankingWeight <= 0 {
		return 0.12
	}
	if c.UtilityRankingWeight > 0.5 {
		return 0.5
	}
	return c.UtilityRankingWeight
}
