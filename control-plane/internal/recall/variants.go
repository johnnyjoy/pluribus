package recall

// Strategy name constants for multi-context recall.
const (
	StrategyBalanced      = "balanced"
	StrategyFailureHeavy  = "failure_heavy"
	StrategyAuthorityHeavy = "authority_heavy"
)

// DefaultStrategyList returns the full list of variant strategies for "default".
func DefaultStrategyList() []string {
	return []string{StrategyBalanced, StrategyFailureHeavy, StrategyAuthorityHeavy}
}

// VariantModifierForStrategy returns the modifier for the given strategy and maxPerKind.
// "balanced" returns nil (no override). failure_heavy and authority_heavy return limits and ranking overrides.
func VariantModifierForStrategy(strategy string, maxPerKind int) *VariantModifier {
	switch strategy {
	case StrategyBalanced:
		return nil
	case StrategyFailureHeavy:
		patterns := maxPerKind - 2
		if patterns < 1 {
			patterns = 1
		}
		w := DefaultRankingWeights()
		w.FailureOverlap = 1.0
		w.SymbolOverlap = 0.8
		// Phase 6: make negative pattern signals dominant in failure-heavy.
		w.PatternPriority = 0.9
		return &VariantModifier{
			Limits: &VariantLimits{
				Failures: intPtr(maxPerKind + 2),
				Patterns: intPtr(patterns),
			},
			Ranking: &w,
		}
	case StrategyAuthorityHeavy:
		w := DefaultRankingWeights()
		w.Authority = 1.5
		return &VariantModifier{
			Limits: &VariantLimits{
				Constraints: intPtr(maxPerKind + 2),
				Decisions:   intPtr(maxPerKind + 2),
			},
			Ranking: &w,
		}
	default:
		return nil
	}
}

func intPtr(n int) *int {
	return &n
}

// ResolveStrategies returns the list of strategy names for the given variants count and strategy key.
// If strategy is not "default", returns a single-element list with that strategy.
// If strategy is "default", returns the first N from DefaultStrategyList(), capped at list length.
func ResolveStrategies(variants int, strategy string) []string {
	all := DefaultStrategyList()
	if strategy != "default" && strategy != "" {
		return []string{strategy}
	}
	if variants <= 0 {
		variants = len(all)
	}
	if variants > len(all) {
		variants = len(all)
	}
	return all[:variants]
}
