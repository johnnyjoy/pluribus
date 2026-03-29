package recall

import "testing"

func TestVariantModifierForStrategy_balanced_returns_nil(t *testing.T) {
	mod := VariantModifierForStrategy(StrategyBalanced, 5)
	if mod != nil {
		t.Errorf("balanced should return nil modifier, got %+v", mod)
	}
}

func TestVariantModifierForStrategy_failure_heavy(t *testing.T) {
	maxPerKind := 5
	mod := VariantModifierForStrategy(StrategyFailureHeavy, maxPerKind)
	if mod == nil {
		t.Fatal("failure_heavy should return non-nil modifier")
	}
	if mod.Limits == nil {
		t.Fatal("failure_heavy should have Limits")
	}
	if mod.Limits.Failures == nil || *mod.Limits.Failures != maxPerKind+2 {
		t.Errorf("failure_heavy Failures want %d, got %v", maxPerKind+2, mod.Limits.Failures)
	}
	if mod.Limits.Patterns == nil || *mod.Limits.Patterns != 3 {
		t.Errorf("failure_heavy Patterns want 3 (max(1,5-2)), got %v", mod.Limits.Patterns)
	}
	if mod.Ranking == nil {
		t.Fatal("failure_heavy should have Ranking override")
	}
	if mod.Ranking.FailureOverlap != 1.0 {
		t.Errorf("failure_heavy FailureOverlap want 1.0, got %v", mod.Ranking.FailureOverlap)
	}
	if mod.Ranking.SymbolOverlap != 0.8 {
		t.Errorf("failure_heavy SymbolOverlap want 0.8, got %v", mod.Ranking.SymbolOverlap)
	}
	if mod.Ranking.PatternPriority != 0.9 {
		t.Errorf("failure_heavy PatternPriority want 0.9, got %v", mod.Ranking.PatternPriority)
	}
}

func TestVariantModifierForStrategy_authority_heavy(t *testing.T) {
	maxPerKind := 5
	mod := VariantModifierForStrategy(StrategyAuthorityHeavy, maxPerKind)
	if mod == nil {
		t.Fatal("authority_heavy should return non-nil modifier")
	}
	if mod.Limits == nil {
		t.Fatal("authority_heavy should have Limits")
	}
	if mod.Limits.Constraints == nil || *mod.Limits.Constraints != maxPerKind+2 {
		t.Errorf("authority_heavy Constraints want %d, got %v", maxPerKind+2, mod.Limits.Constraints)
	}
	if mod.Limits.Decisions == nil || *mod.Limits.Decisions != maxPerKind+2 {
		t.Errorf("authority_heavy Decisions want %d, got %v", maxPerKind+2, mod.Limits.Decisions)
	}
	if mod.Ranking == nil {
		t.Fatal("authority_heavy should have Ranking override")
	}
	if mod.Ranking.Authority != 1.5 {
		t.Errorf("authority_heavy Authority want 1.5, got %v", mod.Ranking.Authority)
	}
}

func TestVariantModifierForStrategy_unknown_returns_nil(t *testing.T) {
	mod := VariantModifierForStrategy("unknown", 5)
	if mod != nil {
		t.Errorf("unknown strategy should return nil, got %+v", mod)
	}
}

func TestResolveStrategies_default_three(t *testing.T) {
	got := ResolveStrategies(3, "default")
	if len(got) != 3 {
		t.Errorf("ResolveStrategies(3, default) want len 3, got %d: %v", len(got), got)
	}
	if got[0] != StrategyBalanced || got[1] != StrategyFailureHeavy || got[2] != StrategyAuthorityHeavy {
		t.Errorf("ResolveStrategies(3, default) want [balanced, failure_heavy, authority_heavy], got %v", got)
	}
}

func TestResolveStrategies_default_two(t *testing.T) {
	got := ResolveStrategies(2, "default")
	if len(got) != 2 {
		t.Errorf("ResolveStrategies(2, default) want len 2, got %d", len(got))
	}
	if got[0] != StrategyBalanced || got[1] != StrategyFailureHeavy {
		t.Errorf("ResolveStrategies(2, default) want [balanced, failure_heavy], got %v", got)
	}
}

func TestResolveStrategies_default_cap_at_list_length(t *testing.T) {
	got := ResolveStrategies(10, "default")
	all := DefaultStrategyList()
	if len(got) != len(all) {
		t.Errorf("ResolveStrategies(10, default) should cap at %d, got %d", len(all), len(got))
	}
}

func TestResolveStrategies_non_default_returns_single(t *testing.T) {
	got := ResolveStrategies(3, "failure_heavy")
	if len(got) != 1 || got[0] != "failure_heavy" {
		t.Errorf("ResolveStrategies(3, failure_heavy) want [failure_heavy], got %v", got)
	}
}

func TestResolveStrategies_zero_variants_uses_full_list(t *testing.T) {
	got := ResolveStrategies(0, "default")
	all := DefaultStrategyList()
	if len(got) != len(all) {
		t.Errorf("ResolveStrategies(0, default) want full list len %d, got %d", len(all), len(got))
	}
}
