package eval

import (
	"strings"

	"control-plane/internal/recall"
)

func RunFollowupBehavior(s Scenario, b *recall.RecallBundle) string {
	var lines []string
	lines = append(lines, "Task: "+s.BehaviorExpectations.NewTask)
	for _, c := range b.Continuity {
		if c.Kind == "state" {
			lines = append(lines, "Continue state: "+c.Statement)
		}
		if c.Kind == "decision" {
			lines = append(lines, "Follow decision: "+c.Statement)
		}
	}
	for _, c := range b.Constraints {
		lines = append(lines, "Guardrail: avoid "+c.Statement)
	}
	for _, p := range b.Experience {
		lines = append(lines, "Apply pattern: "+p.Statement)
	}
	return strings.Join(lines, "\n")
}

func ValidateBehavior(s Scenario, b *recall.RecallBundle, output string) BehaviorResult {
	res := BehaviorResult{Output: output}
	lower := strings.ToLower(output)

	res.AvoidedFailure = true
	for _, bad := range s.BehaviorExpectations.MustAvoid {
		if containsPositiveAction(lower, strings.ToLower(bad)) {
			res.AvoidedFailure = false
			res.Details = append(res.Details, "did not avoid failure action: "+bad)
		}
	}

	res.AppliedPattern = true
	for _, need := range s.BehaviorExpectations.MustApply {
		if !strings.Contains(lower, strings.ToLower(need)) {
			res.AppliedPattern = false
			res.Details = append(res.Details, "did not apply pattern: "+need)
		}
	}

	res.AlignedWithDecision = true
	for _, c := range b.Continuity {
		if c.Kind != "decision" {
			continue
		}
		if !strings.Contains(lower, strings.ToLower(c.Statement)) {
			res.AlignedWithDecision = false
			res.Details = append(res.Details, "not aligned with decision: "+c.Statement)
		}
	}

	res.Pass = res.AvoidedFailure && res.AppliedPattern && res.AlignedWithDecision
	return res
}

func containsPositiveAction(output, term string) bool {
	idx := strings.Index(output, term)
	if idx < 0 {
		return false
	}
	start := idx - 32
	if start < 0 {
		start = 0
	}
	prefix := output[start:idx]
	return !(strings.Contains(prefix, "avoid") || strings.Contains(prefix, "do not") || strings.Contains(prefix, "don't") || strings.Contains(prefix, "never"))
}
