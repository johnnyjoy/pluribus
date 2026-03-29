package runmulti

import (
	"testing"
)

func TestRiskLevelNumeric(t *testing.T) {
	tests := []struct {
		risk string
		want float64
	}{
		{"low", 0},
		{"medium", 1},
		{"high", 2},
		{"", 0},
		{"unknown", 0},
	}
	for _, tt := range tests {
		if got := RiskLevelNumeric(tt.risk); got != tt.want {
			t.Errorf("RiskLevelNumeric(%q) = %v, want %v", tt.risk, got, tt.want)
		}
	}
}

func TestScoreRun_violations_rejected(t *testing.T) {
	d := &DriftResult{
		Passed:     false,
		Violations: []DriftIssue{{Code: "constraint", Statement: "x"}},
		Warnings:   []string{"w1"},
		RiskLevel:  "low",
	}
	score, rejected := ScoreRun(d)
	if !rejected {
		t.Error("expected rejected true when violations present")
	}
	if score != 1 {
		t.Errorf("score = %v, want 1 (0 + 1 warning)", score)
	}
}

func TestScoreRun_no_violations_ordering(t *testing.T) {
	low := &DriftResult{RiskLevel: "low", Warnings: []string{}}
	high := &DriftResult{RiskLevel: "high", Warnings: []string{}}
	sLow, rejLow := ScoreRun(low)
	sHigh, rejHigh := ScoreRun(high)
	if rejLow || rejHigh {
		t.Error("expected not rejected when no violations")
	}
	if sLow >= sHigh {
		t.Errorf("low score %v should be < high score %v", sLow, sHigh)
	}
}

func TestScoreRun_nil_drift(t *testing.T) {
	score, rejected := ScoreRun(nil)
	if score != 0 || rejected {
		t.Errorf("nil drift: score=%v rejected=%v, want 0 false", score, rejected)
	}
}

func TestSelectBest_one_valid(t *testing.T) {
	runs := []RunResult{
		{Variant: "a", Score: 10, Rejected: true},
		{Variant: "b", Score: 1, Rejected: false},
		{Variant: "c", Score: 5, Rejected: false},
	}
	sel := SelectBest(runs)
	if sel == nil || sel.Variant != "b" {
		t.Errorf("SelectBest: got %v, want variant b", sel)
	}
}

func TestSelectBest_all_rejected_fallback(t *testing.T) {
	runs := []RunResult{
		{Variant: "a", Score: 10, Rejected: true},
		{Variant: "b", Score: 2, Rejected: true},
		{Variant: "c", Score: 5, Rejected: true},
	}
	sel := SelectBest(runs)
	if sel == nil {
		t.Fatal("SelectBest: expected fallback, got nil")
	}
	if sel.Variant != "b" || sel.Score != 2 {
		t.Errorf("SelectBest fallback: got %v (score %v), want b (2)", sel.Variant, sel.Score)
	}
}

func TestSelectBest_tie_break_variant(t *testing.T) {
	runs := []RunResult{
		{Variant: "z", Score: 1, Rejected: false},
		{Variant: "a", Score: 1, Rejected: false},
	}
	sel := SelectBest(runs)
	if sel == nil || sel.Variant != "a" {
		t.Errorf("SelectBest tie-break: got %v, want a", sel)
	}
}

func TestSelectBest_empty(t *testing.T) {
	sel := SelectBest(nil)
	if sel != nil {
		t.Errorf("SelectBest(nil) = %v, want nil", sel)
	}
	sel = SelectBest([]RunResult{})
	if sel != nil {
		t.Errorf("SelectBest(empty) = %v, want nil", sel)
	}
}
