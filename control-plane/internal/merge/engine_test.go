package merge

import (
	"context"
	"strings"
	"testing"

	"control-plane/internal/runmulti"
)

func TestRun_zeroValid_fallback(t *testing.T) {
	runs := []runmulti.RunResult{
		{Variant: "a", Output: "out", Rejected: true, Score: 9},
	}
	sel := &runmulti.RunResult{Variant: "a", Output: "fallback text"}
	m := Run(context.Background(), EngineInput{
		Runs:     runs,
		Selected: sel,
	})
	if !m.FallbackUsed || m.MergedOutput != "fallback text" {
		t.Fatalf("got %+v", m)
	}
}

func TestRun_singleValid_passDrift(t *testing.T) {
	runs := []runmulti.RunResult{
		{Variant: "a", Output: "only valid output here", Rejected: false, Drift: runmulti.DriftResult{Violations: nil}},
	}
	sel := &runmulti.RunResult{Variant: "a", Output: "x"}
	check := func(ctx context.Context, proposal string) (runmulti.DriftResult, error) {
		return runmulti.DriftResult{Passed: true}, nil
	}
	m := Run(context.Background(), EngineInput{
		Runs:       runs,
		Selected:   sel,
		DriftCheck: check,
	})
	if m.FallbackUsed || m.MergedOutput != "only valid output here" {
		t.Fatalf("got %+v", m)
	}
}

func TestRun_singleValid_driftFail_fallback(t *testing.T) {
	runs := []runmulti.RunResult{
		{Variant: "a", Output: "bad for drift", Rejected: false, Drift: runmulti.DriftResult{}},
	}
	sel := &runmulti.RunResult{Variant: "a", Output: "selected fallback"}
	check := func(ctx context.Context, proposal string) (runmulti.DriftResult, error) {
		return runmulti.DriftResult{Violations: []runmulti.DriftIssue{{Code: "c", Statement: "s"}}}, nil
	}
	m := Run(context.Background(), EngineInput{
		Runs:       runs,
		Selected:   sel,
		DriftCheck: check,
	})
	if !m.FallbackUsed || m.MergedOutput != "selected fallback" {
		t.Fatalf("got %+v", m)
	}
}

func TestRun_multi_merge_passDrift(t *testing.T) {
	overlap := "We recommend using the repository pattern for data access with clear boundaries and sufficient length."
	runs := []runmulti.RunResult{
		{Variant: "v1", Output: overlap + "\n\nExtra unique detail from variant one here.", Rejected: false, Drift: runmulti.DriftResult{}},
		{Variant: "v2", Output: overlap + "\n\nDifferent unique insight from variant two here.", Rejected: false, Drift: runmulti.DriftResult{}},
		{Variant: "v3", Output: "You must not use global state.\n\n" + overlap, Rejected: false, Drift: runmulti.DriftResult{}},
	}
	sel := &runmulti.RunResult{Variant: "v1", Output: "selected"}
	check := func(ctx context.Context, proposal string) (runmulti.DriftResult, error) {
		return runmulti.DriftResult{Passed: true}, nil
	}
	m := Run(context.Background(), EngineInput{
		Runs:       runs,
		Selected:   sel,
		DriftCheck: check,
	})
	if m.FallbackUsed {
		t.Fatalf("unexpected fallback: %+v", m)
	}
	if len(m.Conflicts) == 0 {
		t.Log("note: no conflicts detected (heuristic may not fire on short lines)")
	}
	if !strings.Contains(m.MergedOutput, "[CORE AGREEMENTS]") || !strings.Contains(m.MergedOutput, "[REFINED STRUCTURE]") {
		t.Fatalf("bad output: %s", m.MergedOutput)
	}
}

func TestRun_multi_driftFail_fallback(t *testing.T) {
	text := "shared text about repository pattern for data access layer with enough characters"
	runs := []runmulti.RunResult{
		{Variant: "a", Output: text, Rejected: false, Drift: runmulti.DriftResult{}},
		{Variant: "b", Output: text, Rejected: false, Drift: runmulti.DriftResult{}},
	}
	sel := &runmulti.RunResult{Variant: "a", Output: "safe selected"}
	check := func(ctx context.Context, proposal string) (runmulti.DriftResult, error) {
		return runmulti.DriftResult{Violations: []runmulti.DriftIssue{{Code: "x", Statement: "y"}}}, nil
	}
	m := Run(context.Background(), EngineInput{
		Runs:       runs,
		Selected:   sel,
		DriftCheck: check,
	})
	if !m.FallbackUsed || m.MergedOutput != "safe selected" {
		t.Fatalf("got %+v", m)
	}
}
