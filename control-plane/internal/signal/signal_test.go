package signal

import (
	"testing"

	"control-plane/internal/merge"
	"control-plane/internal/runmulti"
)

func TestLengthWeight(t *testing.T) {
	if w := LengthWeight(5); w >= 1 {
		t.Errorf("short line should be low weight, got %v", w)
	}
	if w := LengthWeight(100); w < 2 {
		t.Errorf("medium line should be rewarded, got %v", w)
	}
}

func TestPositionPenalty(t *testing.T) {
	if PositionPenalty(0) != 0 {
		t.Errorf("pos 0 penalty 0")
	}
	if PositionPenalty(10) <= PositionPenalty(1) {
		t.Errorf("later position should cost more")
	}
}

func TestTotalSignal(t *testing.T) {
	s := []SegmentScore{{Score: 2}, {Score: 3}}
	if TotalSignal(s) != 5 {
		t.Errorf("TotalSignal = %v", TotalSignal(s))
	}
}

func TestFilterUniques_shortDropped(t *testing.T) {
	cfg := DefaultSignalConfig()
	agreements := []string{"this is a long enough agreement about the api handler"}
	uniques := []string{"short"}
	got := FilterUniques(uniques, agreements, IntentText{}, cfg)
	if len(got) != 0 {
		t.Errorf("want empty, got %v", got)
	}
}

func TestFilterUniques_keepsOverlapWithAgreement(t *testing.T) {
	cfg := DefaultSignalConfig()
	agreements := []string{"we must validate all inputs at the api boundary carefully"}
	uniques := []string{"we must validate all inputs at the api boundary carefully for security reasons"}
	got := FilterUniques(uniques, agreements, IntentText{}, cfg)
	if len(got) != 1 {
		t.Fatalf("want 1, got %v", got)
	}
}

func TestFilterUniques_constraintKeyword(t *testing.T) {
	cfg := DefaultSignalConfig()
	agreements := []string{"unrelated agreement text that is long enough for tests here"}
	uniques := []string{"must ensure the database connection pool is configured correctly"}
	got := FilterUniques(uniques, agreements, IntentText{}, cfg)
	if len(got) != 1 {
		t.Fatalf("keyword unique should pass without agreement overlap, got %v", got)
	}
}

func TestFilterUniques_intentRequiredWhenSet(t *testing.T) {
	cfg := DefaultSignalConfig()
	agreements := []string{"shared token alpha beta gamma delta epsilon"}
	uniques := []string{"shared token alpha beta gamma delta epsilon zeta eta unique addition"}
	intent := IntentText{Prompt: "completely different vocabulary xray zebra quartz"}
	got := FilterUniques(uniques, agreements, intent, cfg)
	if len(got) != 0 {
		t.Errorf("expected intent filter to drop unique, got %v", got)
	}
}

func TestIsHighSignal_positive(t *testing.T) {
	cfg := DefaultSignalConfig()
	m := merge.MergeResult{
		MergedOutput: "[CORE AGREEMENTS]\n- first long agreement line for scoring purposes\n- second long agreement line for scoring purposes\n",
		UsedVariants: []string{"balanced", "failure_heavy"},
		Agreements: []string{
			"first long agreement line for scoring purposes",
			"second long agreement line for scoring purposes",
		},
		Unique:       nil,
		FallbackUsed: false,
		Drift:        runmulti.DriftResult{Passed: true},
	}
	if !IsHighSignal(m, IntentText{}, cfg) {
		t.Error("expected high signal")
	}
}

func TestIsHighSignal_fallback(t *testing.T) {
	cfg := DefaultSignalConfig()
	m := merge.MergeResult{FallbackUsed: true, Agreements: []string{"long enough agreement line here"}}
	if IsHighSignal(m, IntentText{}, cfg) {
		t.Error("fallback should not be high signal")
	}
}

func TestIsHighSignal_driftViolations(t *testing.T) {
	cfg := DefaultSignalConfig()
	m := merge.MergeResult{
		Agreements:   []string{"long agreement line one", "long agreement line two"},
		UsedVariants: []string{"a", "b"},
		Drift:        runmulti.DriftResult{Violations: []runmulti.DriftIssue{{Code: "x"}}},
	}
	if IsHighSignal(m, IntentText{}, cfg) {
		t.Error("violations => not high signal")
	}
}

func TestIsHighSignal_lowScore(t *testing.T) {
	cfg := DefaultSignalConfig()
	cfg.MinTotalScore = 1000
	m := merge.MergeResult{
		Agreements:   []string{"x"},
		UsedVariants: []string{"only"},
		FallbackUsed: false,
		Drift:        runmulti.DriftResult{},
	}
	if IsHighSignal(m, IntentText{}, cfg) {
		t.Error("min score gate")
	}
}
