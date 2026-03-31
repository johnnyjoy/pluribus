package curation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"control-plane/internal/evidence"
	"control-plane/internal/memory"
	"control-plane/internal/similarity"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// MemoryCreator creates behavior memory objects from curation signals.
type MemoryCreator interface {
	Create(ctx context.Context, req memory.CreateRequest) (*memory.MemoryObject, error)
	FindCanonicalConsolidationMatch(ctx context.Context, cfg *memory.CanonicalConsolidationConfig, in *memory.ConsolidationProposalInput) (*memory.ConsolidationDecision, error)
	ConsolidatePromotion(ctx context.Context, req memory.ConsolidatePromotionRequest) (*memory.MemoryObject, error)
}

// FailureStatementCounter counts durable failures for a statement key (optional; digest repeated-failure promotion).
type FailureStatementCounter interface {
	CountActiveFailuresWithStatementKey(ctx context.Context, statementKey string) (int, error)
}

// MemoryByIDGetter loads a canonical memory by id (review hints).
type MemoryByIDGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*memory.MemoryObject, error)
}

// Service provides curation use cases.
type Service struct {
	Repo   *Repo
	Config *SalienceConfig
	// Memory is optional; when set, promotions create behavior kinds only.
	Memory MemoryCreator
	// MemoryLookup optional: GET review uses for supersedes target preview.
	MemoryLookup MemoryByIDGetter
	// Relationships optional: concise edges for review (bounded).
	Relationships *memory.RelationshipRepo
	// Digest (optional): structured POST /v1/curation/digest.
	DigestLimits *DigestLimits
	Evidence     *evidence.Service
	Promotion    *PromotionDigestConfig
	// FailureCounter optional: when set, digest may emit an extra constraint when a failure matches an existing failure row.
	FailureCounter FailureStatementCounter
	// Episodes optional: when set, candidate review loads short summaries from advisory_episodes by proposal trace ids.
	Episodes *similarity.Repo
	// MemoryDup optional: when set, promotion validation rejects duplicates via FindActiveDuplicate.
	MemoryDup ActiveDuplicateChecker
}

// Evaluate scores the text and optionally creates a candidate; returns EvaluateResult.
func (s *Service) Evaluate(ctx context.Context, req EvaluateRequest) (*EvaluateResult, error) {
	cfg := s.Config
	if cfg == nil {
		cfg = &SalienceConfig{CandidateThreshold: 0.5, ReviewThreshold: 0.7, PromoteThreshold: 0.85}
	}
	score := ScoreText(req.Text, cfg)
	res := &EvaluateResult{
		SalienceScore: score,
		ShouldReview:  score >= cfg.ReviewThreshold,
		ShouldPromote: score >= cfg.PromoteThreshold,
	}
	if score >= cfg.CandidateThreshold {
		cand, err := s.Repo.Create(ctx, req.Text, score)
		if err != nil {
			return nil, err
		}
		res.Created = true
		res.CandidateID = cand.ID.String()
	}
	return res, nil
}

// ListPending returns pending candidates (global queue).
func (s *Service) ListPending(ctx context.Context) ([]CandidateEvent, error) {
	return s.Repo.ListPending(ctx)
}

// ListPromotionSuggestions returns pending candidates surfaced for promotion assist (high_confidence or review_recommended; never auto-promotes).
func (s *Service) ListPromotionSuggestions(ctx context.Context) ([]CandidateEvent, error) {
	all, err := s.ListPending(ctx)
	if err != nil {
		return nil, err
	}
	var out []CandidateEvent
	for _, c := range all {
		switch c.PromotionReadiness {
		case ReadinessHighConfidence, ReadinessReviewRecommended:
			out = append(out, c)
		}
	}
	return out, nil
}

// ListStrengthened returns pending candidates with distill_support_count >= minSupport (default 2).
func (s *Service) ListStrengthened(ctx context.Context, minSupport int) ([]CandidateEvent, error) {
	if s.Repo == nil {
		return nil, errors.New("curation service not configured")
	}
	return s.Repo.ListPendingStrengthened(ctx, minSupport)
}

// MarkPromoted sets the candidate's promotion_status to "promoted".
func (s *Service) MarkPromoted(ctx context.Context, id uuid.UUID) error {
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("candidate not found")
	}
	if c.PromotionStatus != "pending" {
		return fmt.Errorf("candidate already %s", c.PromotionStatus)
	}
	return s.Repo.UpdatePromotionStatus(ctx, id, "promoted")
}

// MarkRejected sets the candidate's promotion_status to "rejected".
func (s *Service) MarkRejected(ctx context.Context, id uuid.UUID) error {
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if c == nil {
		return fmt.Errorf("candidate not found")
	}
	if c.PromotionStatus != "pending" {
		return fmt.Errorf("candidate already %s", c.PromotionStatus)
	}
	return s.Repo.UpdatePromotionStatus(ctx, id, "rejected")
}

// PromoteToPatternRequest is accepted for compatibility with existing API handlers.
type PromoteToPatternRequest struct {
	Payload memory.PatternPayload `json:"payload"`
}

// PromoteToPattern creates a pattern memory from the payload and marks the candidate as promoted.
// Requires Service.Memory to be set. Returns the created memory object.
func (s *Service) PromoteToPattern(ctx context.Context, candidateID uuid.UUID, payload *memory.PatternPayload) (*memory.MemoryObject, error) {
	if payload == nil {
		return nil, memory.ErrPatternPayloadRequired
	}
	if err := memory.ValidatePatternPayload(payload); err != nil {
		return nil, err
	}
	c, err := s.Repo.GetByID(ctx, candidateID)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("candidate not found")
	}
	if c.PromotionStatus != "pending" {
		return nil, fmt.Errorf("candidate already %s", c.PromotionStatus)
	}
	if s.Memory == nil {
		return nil, fmt.Errorf("promote to pattern not configured (memory service required)")
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	raw := json.RawMessage(payloadBytes)
	statement := payload.Directive
	if statement == "" {
		statement = c.RawText
	}
	if statement == "" {
		statement = payload.Experience
	}
	req := memory.CreateRequest{
		Kind:      api.MemoryKindPattern,
		Authority: 5,
		Statement: statement,
		Payload:   &raw,
	}
	obj, err := s.Memory.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	if err := s.Repo.UpdatePromotionStatus(ctx, candidateID, "promoted"); err != nil {
		return nil, err
	}
	return obj, nil
}
