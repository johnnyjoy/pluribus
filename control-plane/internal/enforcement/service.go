package enforcement

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"control-plane/internal/app"
	"control-plane/internal/evidence"
	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// SuccessAuthorityReinforcer bumps memory authority on validated success paths (optional).
type SuccessAuthorityReinforcer interface {
	ReinforceSuccess(ctx context.Context, ids []uuid.UUID, reason string) error
}

// Service evaluates proposals against binding memory.
type Service struct {
	Repo                *memory.Repo
	Evidence            EvidenceLister
	Config              *app.EnforcementConfig
	SuccessReinforcer   SuccessAuthorityReinforcer // optional
}

// EvidenceLister loads evidence linked to a memory (optional).
type EvidenceLister interface {
	ListEvidenceForMemory(ctx context.Context, memoryID uuid.UUID) ([]evidence.Record, error)
}

// Evaluate runs the enforcement gate.
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResponse, error) {
	if s == nil || s.Config == nil {
		return nil, ErrDisabled
	}
	if !s.Config.IsEnabled() {
		return nil, ErrDisabled
	}
	if err := ValidateEvaluateRequest(req); err != nil {
		return nil, err
	}
	if s.Repo == nil {
		return nil, errors.New("enforcement: memory repo not configured")
	}
	cfg := s.Config
	memories, err := s.Repo.ListBindingMemory(ctx, memory.ListBindingRequest{
		MinAuthority:  cfg.MinBindingAuthority,
		Max:           cfg.MaxBindingMemories,
		Kinds:         nil,
	})
	if err != nil {
		return nil, err
	}

	hits := evaluateAll(memories, req.ProposalText, req.Intent, cfg.FailureOverlapThreshold, cfg.PatternBlockScore)
	validation := summarizeValidation(req, hits, DecisionAllow)
	if len(hits) == 0 {
		validation.Passed = validation.MovesTowardGoal
		if validation.Passed {
			validation.NextAction = "proceed"
		} else {
			validation.NextAction = "revise"
		}
		return &EvaluateResponse{
			Decision:          DecisionAllow,
			Explanation:       "No binding trusted memory conflicts with this proposal.",
			TriggeredMemories: []TriggeredMemory{},
			Validation:        validation,
			EvaluationEngine:  EvaluationEngineRuleBasedHeuristicV1,
			EvaluationNote:    EvaluationNoteRuleBased,
		}, nil
	}

	decisions := make([]EnforcementDecision, len(hits))
	for i := range hits {
		decisions[i] = hits[i].Decision
	}
	final := worstDecision(decisions)
	validation = summarizeValidation(req, hits, final)

	var explanations []string
	for _, x := range hits {
		explanations = append(explanations, x.Detail)
	}
	explanation := explanations[0]
	if len(explanations) > 1 {
		explanation = fmt.Sprintf("%d issues: %s", len(explanations), explanations[0])
	}

	tm, err := s.buildTriggered(ctx, hits, cfg.MaxEvidencePerMemory)
	if err != nil {
		return nil, err
	}

	resp := &EvaluateResponse{
		Decision:          final,
		Explanation:       explanation,
		TriggeredMemories: tm,
		Validation:        validation,
		EvaluationEngine:  EvaluationEngineRuleBasedHeuristicV1,
		EvaluationNote:    EvaluationNoteRuleBased,
		RemediationHints: []string{
			"Validation failed: revise proposal or reject this action per NextAction.",
			"After revision, re-run recall and enforcement before proceeding.",
		},
	}
	if final == DecisionBlockOverrideable {
		resp.Override = &OverrideHint{
			Required: true,
			Summary:  "Document rationale and obtain review before overriding this gate.",
		}
	}
	return resp, nil
}

func summarizeValidation(req EvaluateRequest, hits []evalHit, final EnforcementDecision) ValidationSummary {
	v := ValidationSummary{
		MovesTowardGoal: movesTowardGoal(req.Goal, req.ProposalText),
	}
	for _, h := range hits {
		switch {
		case h.ReasonCode == "normative_conflict" && h.Memory.Kind == api.MemoryKindConstraint:
			v.ViolatedConstraints = true
		case h.ReasonCode == "anti_pattern_overlap" || (h.ReasonCode == "negative_pattern" && h.Memory.Kind == api.MemoryKindFailure):
			v.RepeatedFailures = true
		case h.ReasonCode == "normative_conflict" && h.Memory.Kind == api.MemoryKindDecision:
			v.ContradictedDecisions = true
		}
	}
	v.Passed = !(v.ViolatedConstraints || v.RepeatedFailures || v.ContradictedDecisions) && v.MovesTowardGoal
	switch {
	case final == DecisionBlock || final == DecisionBlockOverrideable:
		v.NextAction = "reject"
	case v.Passed:
		v.NextAction = "proceed"
	default:
		v.NextAction = "revise"
	}
	return v
}

func movesTowardGoal(goal, proposal string) bool {
	goal = strings.TrimSpace(strings.ToLower(goal))
	proposal = strings.TrimSpace(strings.ToLower(proposal))
	if goal == "" {
		return false
	}
	if proposal == "" {
		return false
	}
	goalWords := strings.Fields(goal)
	if len(goalWords) == 0 {
		return false
	}
	var overlap int
	for _, w := range goalWords {
		if strings.Contains(proposal, w) {
			overlap++
		}
	}
	return overlap > 0
}

func (s *Service) buildTriggered(ctx context.Context, tr []evalHit, maxEv int) ([]TriggeredMemory, error) {
	if maxEv <= 0 {
		maxEv = 3
	}
	out := make([]TriggeredMemory, 0, len(tr))
	for _, x := range tr {
		tm := TriggeredMemory{
			MemoryID:         x.Memory.ID,
			Kind:             string(x.Memory.Kind),
			Authority:        x.Memory.Authority,
			StatementSnippet: statementSnippet(x.Memory.Statement, 220),
			ReasonCode:       x.ReasonCode,
			Detail:           x.Detail,
		}
		if s.Evidence != nil {
			recs, err := s.Evidence.ListEvidenceForMemory(ctx, x.Memory.ID)
			if err != nil {
				return nil, err
			}
			for i, r := range recs {
				if i >= maxEv {
					break
				}
				tm.Evidence = append(tm.Evidence, EvidenceRef{ID: r.ID, Kind: r.Kind, Path: r.Path})
			}
		}
		out = append(out, tm)
	}
	return out, nil
}

func statementSnippet(s string, maxRunes int) string {
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}
