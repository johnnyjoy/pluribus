package enforcement

import (
	"testing"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

func TestSummarizeValidation_rejectOnConstraintViolation(t *testing.T) {
	req := EvaluateRequest{
		ProposalText: "switch db to sqlite",
		Goal:         "ship db migration safely",
	}
	hits := []evalHit{
		{
			Memory:     memory.MemoryObject{Kind: api.MemoryKindConstraint},
			ReasonCode: "normative_conflict",
		},
	}
	got := summarizeValidation(req, hits, DecisionBlock)
	if !got.ViolatedConstraints {
		t.Fatal("expected violated_constraints=true")
	}
	if got.NextAction != "reject" {
		t.Fatalf("next_action=%q want reject", got.NextAction)
	}
	if got.Passed {
		t.Fatal("expected passed=false")
	}
}

func TestSummarizeValidation_reviseWhenGoalNotPresent(t *testing.T) {
	req := EvaluateRequest{
		ProposalText: "refactor API handler",
		Goal:         "",
	}
	got := summarizeValidation(req, nil, DecisionAllow)
	if got.MovesTowardGoal {
		t.Fatal("expected moves_toward_goal=false when goal missing")
	}
	if got.NextAction != "revise" {
		t.Fatalf("next_action=%q want revise", got.NextAction)
	}
	if got.Passed {
		t.Fatal("expected passed=false")
	}
}

func TestSummarizeValidation_proceedWhenChecksPass(t *testing.T) {
	req := EvaluateRequest{
		ProposalText: "improve onboarding flow and reduce churn",
		Goal:         "reduce churn",
	}
	got := summarizeValidation(req, nil, DecisionAllow)
	if !got.MovesTowardGoal {
		t.Fatal("expected moves_toward_goal=true")
	}
	if got.NextAction != "proceed" {
		t.Fatalf("next_action=%q want proceed", got.NextAction)
	}
	if !got.Passed {
		t.Fatal("expected passed=true")
	}
}

// Phase 5 mandatory test 2: failure avoidance.
func TestMandatory_FailureAvoidance_repeatedFailureForcesReviseOrReject(t *testing.T) {
	req := EvaluateRequest{
		ProposalText: "repeat known anti pattern",
		Goal:         "avoid regressions",
	}
	hits := []evalHit{
		{
			Memory:     memory.MemoryObject{Kind: api.MemoryKindFailure},
			ReasonCode: "anti_pattern_overlap",
		},
	}
	got := summarizeValidation(req, hits, DecisionRequireReview)
	if !got.RepeatedFailures {
		t.Fatal("expected repeated_known_failures=true")
	}
	if got.NextAction != "revise" && got.NextAction != "reject" {
		t.Fatalf("next_action=%q want revise or reject", got.NextAction)
	}
	if got.Passed {
		t.Fatal("expected passed=false")
	}
}
