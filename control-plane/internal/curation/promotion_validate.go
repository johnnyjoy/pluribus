package curation

import (
	"context"
	"fmt"
	"strings"

	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

const minPromotionStatementRunes = 16

// ActiveDuplicateChecker checks for an existing active/pending memory with the same statement key (optional).
type ActiveDuplicateChecker interface {
	FindActiveDuplicate(ctx context.Context, kind api.MemoryKind, statementKey string) (*uuid.UUID, error)
}

// ValidatePromotionCandidate enforces guardrails before materialize (manual or auto).
func (s *Service) ValidatePromotionCandidate(ctx context.Context, c *CandidateEvent, p *ProposalPayloadV1) PromotionValidationResult {
	if p == nil || strings.TrimSpace(p.Statement) == "" {
		return PromotionValidationResult{Allow: false, Reason: "candidate has no structured proposal statement"}
	}
	if !isBehaviorKind(p.Kind) {
		return PromotionValidationResult{Allow: false, Reason: "materialize only supports behavior memory kinds"}
	}
	stmt := strings.TrimSpace(p.Statement)
	if len([]rune(stmt)) < minPromotionStatementRunes {
		return PromotionValidationResult{Allow: false, Reason: fmt.Sprintf("statement must be at least %d characters", minPromotionStatementRunes)}
	}
	promo := s.Promotion
	if promo != nil && promo.RequireEvidence && len(p.EvidenceIDs) == 0 {
		return PromotionValidationResult{Allow: false, Reason: "promotion requires evidence links"}
	}
	if promo != nil && promo.MinEvidenceLinks > 0 && len(p.EvidenceIDs) < promo.MinEvidenceLinks {
		return PromotionValidationResult{Allow: false, Reason: fmt.Sprintf("need at least %d evidence link(s)", promo.MinEvidenceLinks)}
	}
	stmtKey := memorynorm.StatementKey(p.Statement)
	if s.MemoryDup != nil && stmtKey != "" {
		dupID, err := s.MemoryDup.FindActiveDuplicate(ctx, p.Kind, stmtKey)
		if err != nil {
			return PromotionValidationResult{Allow: false, Reason: err.Error()}
		}
		if dupID != nil {
			sup := strings.TrimSpace(p.SupersedesMemoryID)
			if sup == "" {
				return PromotionValidationResult{Allow: false, Reason: "active or pending memory already exists for this statement key"}
			}
			su, err := uuid.Parse(sup)
			if err != nil {
				return PromotionValidationResult{Allow: false, Reason: "invalid supersedes_memory_id: " + err.Error()}
			}
			if su != *dupID {
				return PromotionValidationResult{Allow: false, Reason: "supersedes_memory_id must equal the existing memory id for this statement key"}
			}
		}
	}
	n := supportCount(p)
	if c.SalienceScore < 0.15 && n > 6 {
		return PromotionValidationResult{Allow: false, Reason: "inconsistent salience vs merged support count"}
	}
	return PromotionValidationResult{Allow: true, Reason: "ok"}
}
