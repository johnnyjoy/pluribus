package drift

import (
	"testing"
)

func TestCheck_duplicate_responsibility(t *testing.T) {
	constraints := []string{"duplicate query builders"}
	failures := []string{}
	proposal := "I added a duplicate query builders helper."
	issues := Check(proposal, constraints, failures, 0)
	if len(issues) != 1 || issues[0].Code != "constraint" || issues[0].Statement != "duplicate query builders" {
		t.Errorf("expected one constraint violation, got %v", issues)
	}
}

func TestCheck_fluent_regression(t *testing.T) {
	constraints := []string{}
	failures := []string{"Fluent terminators broken"}
	proposal := "refactor broke Fluent terminators broken again."
	issues := Check(proposal, constraints, failures, 0)
	if len(issues) != 1 || issues[0].Code != "failure" || issues[0].Statement != "Fluent terminators broken" {
		t.Errorf("expected one failure violation, got %v", issues)
	}
}

func TestCheck_no_match(t *testing.T) {
	constraints := []string{"No duplicate query builders"}
	proposal := "We use a single query builder and no duplicates exist."
	issues := Check(proposal, constraints, nil, 0)
	if len(issues) != 0 {
		t.Errorf("expected no violations, got %v", issues)
	}
}

func TestCheck_case_insensitive(t *testing.T) {
	constraints := []string{"Duplicate Builder"}
	proposal := "added a duplicate builder."
	issues := Check(proposal, constraints, nil, 0)
	if len(issues) != 1 {
		t.Errorf("expected 1 violation (case-insensitive), got %v", issues)
	}
}

func TestCheck_multiple_violations(t *testing.T) {
	constraints := []string{"no globals", "single builder"}
	proposal := "We avoid no globals and use a single builder."
	issues := Check(proposal, constraints, nil, 0)
	if len(issues) != 2 {
		t.Errorf("expected 2 violations, got %d: %v", len(issues), issues)
	}
}

func TestCheck_failure_fuzzy_pattern(t *testing.T) {
	// No exact substring match, but high word overlap -> failure_pattern (Task 76).
	failures := []string{"fluent terminators broken"}
	proposal := "The fluent and terminators are broken again in this refactor."
	issues := Check(proposal, nil, failures, 0.8)
	var fp []DriftIssue
	for _, i := range issues {
		if i.Code == "failure_pattern" {
			fp = append(fp, i)
		}
	}
	if len(fp) != 1 || fp[0].Statement != "fluent terminators broken" || fp[0].Score < 0.8 {
		t.Errorf("expected one failure_pattern with score >= 0.8, got %v", issues)
	}
}
