package drift

import (
	"context"
	"fmt"

	"control-plane/internal/memory"
	"control-plane/internal/tooling"
	"control-plane/pkg/api"
)

// Service provides drift check use cases.
type Service struct {
	Repo                           *Repo
	Memory                         MemorySearcher
	FailureFuzzyThreshold          float64  // 0 = disabled; 0.8 = word-overlap threshold for failure_pattern (Task 76)
	RequireSecondDriftCheck        bool     // Task 95: when true and request has SlowPathRequired, result.RequiresFollowupCheck until second call
	LSP                            tooling.LSPClient // Task 100: optional; when set with LSPHighRiskReferenceThreshold, high ref count escalates risk
	LSPHighRiskReferenceThreshold  int      // Task 100: 0 = off; if symbol reference count > this, set risk high and block
	PatternHighBlocks              bool     // when true, high-severity negative pattern matches block execution
}

// MemorySearcher is satisfied by memory.Service.
type MemorySearcher interface {
	Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error)
}

// Check loads active constraints and failures from shared memory (optional tags), runs the checker
// on the proposal, persists the result, and returns CheckResult.
// Negative pattern payload overlaps (directive/experience/tag) are added as violations/warnings.
func (s *Service) Check(ctx context.Context, req CheckRequest) (*CheckResult, error) {
	var constraintStmts, failureStmts []string
	var objs []memory.MemoryObject
	if s.Memory != nil {
		searchReq := memory.SearchRequest{
			Tags:   req.Tags,
			Status: "active",
			Max:    200,
		}
		var err error
		objs, err = s.Memory.Search(ctx, searchReq)
		if err != nil {
			return nil, err
		}
		for _, o := range objs {
			if o.Statement == "" {
				continue
			}
			switch o.Kind {
			case api.MemoryKindConstraint:
				constraintStmts = append(constraintStmts, o.Statement)
			case api.MemoryKindFailure:
				failureStmts = append(failureStmts, o.Statement)
			}
		}
	}
	threshold := s.FailureFuzzyThreshold
	if threshold <= 0 {
		threshold = 0
	}
	violations := Check(req.Proposal, constraintStmts, failureStmts, threshold)

	// Negative pattern matching: directive/experience/tag overlap; high/catastrophic → violation, else → warning
	warnings := []string{}
	patternIssues := NegativePatternMatches(req.Proposal, objs)
	patternViolationCount := 0
	patternWarningCount := 0
	for _, issue := range patternIssues {
		if issue.Score >= 4 {
			violations = append(violations, issue)
			patternViolationCount++
		} else {
			warnings = append(warnings, issue.Statement)
			patternWarningCount++
		}
	}

	passed := len(violations) == 0

	// Structural risk escalation (Task 77)
	var riskLevel string
	var blockExecution bool
	if req.StructuralSignals != nil {
		riskLevel, blockExecution, warnings = AssessRisk(*req.StructuralSignals)
	}

	// Task 100: LSP reference-count risk — touching a heavily referenced symbol escalates to high risk / block
	if s.LSP != nil && s.LSPHighRiskReferenceThreshold > 0 && req.RepoRoot != "" && len(req.TouchedSymbols) > 0 {
		for _, pos := range req.TouchedSymbols {
			refs, err := s.LSP.FindReferences(ctx, req.RepoRoot, pos.Path, pos.Line, pos.Column)
			if err != nil {
				warnings = append(warnings, "LSP reference check failed for "+pos.Path+": "+err.Error())
				continue
			}
			if len(refs) > s.LSPHighRiskReferenceThreshold {
				riskLevel = "high"
				blockExecution = true
				warnings = append(warnings, fmt.Sprintf("symbol at %s:%d:%d has high reference count (%d); drift risk escalated", pos.Path, pos.Line, pos.Column, len(refs)))
				break
			}
		}
	}

	// Negative pattern risk: elevate risk when proposal overlaps negative patterns.
	if riskLevel == "" && len(patternIssues) > 0 {
		riskLevel = "medium"
		for _, issue := range patternIssues {
			if issue.Score >= 4 && s.PatternHighBlocks {
				riskLevel = "high"
				blockExecution = true
				break
			}
		}
	}

	if s.Repo != nil {
		_, _ = s.Repo.CreateCheck(ctx, passed, violations, warnings)
	}
	out := &CheckResult{
		Passed:     passed,
		Violations: violations,
		Warnings:   warnings,
		PatternViolationCount: patternViolationCount,
		PatternWarningCount: patternWarningCount,
	}
	if riskLevel != "" {
		out.RiskLevel = riskLevel
		out.BlockExecution = blockExecution
	}
	// Task 95: slow-path requires second drift check before execution.
	if req.SlowPathRequired && s.RequireSecondDriftCheck && !req.IsFollowupCheck {
		out.RequiresFollowupCheck = true
		out.FollowupReason = "slow-path requires second drift check before execution"
	}
	return out, nil
}
