package eval

import (
	"context"
	"fmt"
	"strings"

	"control-plane/internal/recall"
)

func runStressScenario(ctx context.Context, svc *recall.Service, s Scenario, triggered bool) (*StressReport, error) {
	if len(s.Steps) == 0 {
		return nil, nil
	}
	report := &StressReport{ScenarioType: strings.TrimSpace(s.ScenarioType), ContinuityMaintained: true, FailureAvoided: true, PatternReused: true}
	for i, st := range s.Steps {
		stepScenario := scenarioForStep(s, st)
		var (
			bundle *recall.RecallBundle
			err    error
		)
		if triggered {
			bundle, _, err = runScenarioTriggered(ctx, svc, stepScenario)
		} else {
			bundle, err = runScenarioExplicit(ctx, svc, stepScenario)
		}
		if err != nil {
			return nil, fmt.Errorf("step %d: %w", i+1, err)
		}
		recallRes := ValidateRecall(stepScenario, bundle)
		out := RunFollowupBehavior(stepScenario, bundle)
		beh := ValidateBehavior(stepScenario, bundle, out)
		trace := StepTrace{
			Index:               i + 1,
			Task:                firstNonEmpty(st.Task, "n/a"),
			Action:              firstNonEmpty(st.Action, "n/a"),
			RecallUsed:          len(bundle.Continuity)+len(bundle.Constraints)+len(bundle.Experience) > 0,
			RestoredState:       len(bundle.Continuity),
			RestoredConstraints: len(bundle.Constraints),
		}
		trace.DriftEvents = append(trace.DriftEvents, detectStepDrift(recallRes, beh)...)
		if len(trace.DriftEvents) > 0 {
			report.DriftDetected = true
			for _, d := range trace.DriftEvents {
				report.Issues = append(report.Issues, d.Type+": "+d.Cause)
			}
		}
		if !beh.AlignedWithDecision {
			report.ContinuityMaintained = false
		}
		if !beh.AvoidedFailure {
			report.FailureAvoided = false
		}
		if !beh.AppliedPattern {
			report.PatternReused = false
		}
		report.StepTraces = append(report.StepTraces, trace)
	}
	for _, r := range s.ResumePoints {
		if r.Step <= 0 || r.Step > len(report.StepTraces) {
			continue
		}
		st := &report.StepTraces[r.Step-1]
		st.Action = strings.TrimSpace(st.Action + " [resume gap: " + strings.TrimSpace(r.Gap) + "]")
	}
	return report, nil
}

func scenarioForStep(base Scenario, st WorkflowStep) Scenario {
	out := base
	out.Trap = firstNonEmpty(strings.TrimSpace(st.Trap), strings.TrimSpace(st.Action), strings.TrimSpace(base.Trap))
	out.RecallExpectations.Query = firstNonEmpty(strings.TrimSpace(st.Query), strings.TrimSpace(base.RecallExpectations.Query))
	if len(st.MustInclude) > 0 {
		out.RecallExpectations.MustInclude = append([]string{}, st.MustInclude...)
	}
	if nt := strings.TrimSpace(st.NewTask); nt != "" {
		out.BehaviorExpectations.NewTask = nt
	}
	if len(st.MustAvoid) > 0 {
		out.BehaviorExpectations.MustAvoid = append([]string{}, st.MustAvoid...)
	}
	if len(st.MustApply) > 0 {
		out.BehaviorExpectations.MustApply = append([]string{}, st.MustApply...)
	}
	return out
}

func detectStepDrift(recallRes CheckResult, beh BehaviorResult) []DriftEvent {
	var out []DriftEvent
	for _, d := range recallRes.Details {
		ld := strings.ToLower(d)
		switch {
		case strings.Contains(ld, "must_be_first"), strings.Contains(ld, "missing"):
			out = append(out, DriftEvent{Type: "goal_deviation", Cause: d})
		}
	}
	for _, d := range beh.Details {
		ld := strings.ToLower(d)
		switch {
		case strings.Contains(ld, "not aligned with decision"):
			out = append(out, DriftEvent{Type: "contradicted_decision", Cause: d})
		case strings.Contains(ld, "did not avoid failure"):
			out = append(out, DriftEvent{Type: "reintroduced_failure", Cause: d})
		case strings.Contains(ld, "did not apply pattern"):
			out = append(out, DriftEvent{Type: "goal_deviation", Cause: d})
		}
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}
