package curation

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"control-plane/internal/memory"
	"control-plane/internal/memorynorm"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// PromotionDigestConfig gates materialize behavior (from app promotion YAML).
type PromotionDigestConfig struct {
	RequireEvidence  bool
	MinEvidenceLinks int
	RequireReview    bool
	// AutoPromote enables POST /v1/curation/auto-promote (default false).
	AutoPromote bool
	// AutoMinSupportCount / AutoMinSalience / AutoAllowedKinds gate automatic materialization (conservative defaults from config).
	AutoMinSupportCount int
	AutoMinSalience     float64
	AutoAllowedKinds    []string
	// CanonicalConsolidation merges similar promotions into existing canonical rows (deterministic; default off).
	CanonicalConsolidation *memory.CanonicalConsolidationConfig
}

// Service already has Repo, Config, Memory; extend in service.go with:
// Targets, Tasks, Evidence, DigestLimits, Promotion *PromotionDigestConfig

// Digest turns post-work input into pending candidate rows (unless dry_run).
func (s *Service) Digest(ctx context.Context, req DigestRequest) (*DigestResult, error) {
	limits := s.digestLimits()
	if err := ValidateDigestRequest(&req, limits); err != nil {
		return nil, err
	}
	maxP := limits.MaxProposals
	if req.Options != nil && req.Options.MaxProposals > 0 && req.Options.MaxProposals < maxP {
		maxP = req.Options.MaxProposals
	}
	if err := s.validateDigestRefs(ctx, &req); err != nil {
		return nil, err
	}

	drafts := buildDrafts(req)
	drafts = dedupeDrafts(drafts)
	drafts = s.appendRepeatedFailureConstraints(ctx, req, drafts)
	drafts = dedupeDrafts(drafts)
	drafts = prioritizeTruncateDrafts(drafts, maxP)
	var rejected []DigestRejected
	if len(drafts) == 0 {
		rejected = append(rejected, DigestRejected{Reason: "no_proposals", Detail: "No extractable structured signal; provide curation_answers or a longer work_summary"})
		return &DigestResult{Rejected: rejected}, nil
	}

	cfg := s.Config
	if cfg == nil {
		cfg = &SalienceConfig{CandidateThreshold: 0.5, ReviewThreshold: 0.7, PromoteThreshold: 0.85}
	}

	var out []DigestProposal
	truncated := false
	for i, d := range drafts {
		if i >= maxP {
			truncated = true
			break
		}
		text := d.statement
		reason := truncateReason(d.reason, limits.ReasonMaxBytes)
		if len(text) > limits.StatementMaxBytes {
			text = text[:limits.StatementMaxBytes]
		}
		score := ScoreText(text+" "+reason, cfg)

		evidenceIDs := append([]uuid.UUID(nil), req.EvidenceIDs...)

		propID := uuid.New().String()
		payload := ProposalPayloadV1{
			V:                 1,
			Kind:              d.kind,
			Statement:         text,
			Reason:            reason,
			Tags:              d.tags,
			ProposedAuthority: defaultAuthority(d.kind),
			EvidenceIDs:       evidenceIDs,
			ArtifactRefs:      req.ArtifactRefs,
			WorkSummary:       truncateReason(req.WorkSummary, limits.WorkSummaryMaxBytes),
			ProposalID:        propID,
		}
		pj, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}

		dp := DigestProposal{
			ProposalID:        propID,
			Kind:              d.kind,
			Statement:         text,
			Reason:            reason,
			Tags:              d.tags,
			ProposedAuthority: payload.ProposedAuthority,
			EvidenceIDs:       evidenceIDs,
			SalienceScore:     score,
		}

		if req.Options != nil && req.Options.DryRun {
			out = append(out, dp)
			continue
		}

		cand, err := s.Repo.CreateDigest(ctx, text, score, pj)
		if err != nil {
			return nil, err
		}
		dp.CandidateID = cand.ID.String()
		out = append(out, dp)
	}

	return &DigestResult{Proposals: out, Truncated: truncated}, nil
}

const maxRepeatedFailureConstraints = 2

func (s *Service) appendRepeatedFailureConstraints(ctx context.Context, req DigestRequest, drafts []draft) []draft {
	if s.FailureCounter == nil {
		return drafts
	}
	added := 0
	for _, d := range drafts {
		if d.kind != api.MemoryKindFailure {
			continue
		}
		if added >= maxRepeatedFailureConstraints {
			break
		}
		sk := memorynorm.StatementKey(d.statement)
		if sk == "" {
			continue
		}
		n, err := s.FailureCounter.CountActiveFailuresWithStatementKey(ctx, sk)
		if err != nil || n < 1 {
			continue
		}
		shaped := shapeConstraintStatement(d.statement)
		if hasConstraintStatement(drafts, shaped) {
			continue
		}
		drafts = append(drafts, draft{
			kind:      api.MemoryKindConstraint,
			statement: shaped,
			reason:    "promoted: repeated failure",
			tags:      tagMerge(req.Signals, string(api.MemoryKindConstraint)),
		})
		added++
	}
	return drafts
}

func (s *Service) digestLimits() *DigestLimits {
	if s.DigestLimits != nil {
		return s.DigestLimits
	}
	return defaultDigestLimits()
}

func (s *Service) validateDigestRefs(ctx context.Context, req *DigestRequest) error {
	if s.Evidence != nil && len(req.EvidenceIDs) > 0 {
		for _, eid := range req.EvidenceIDs {
			rec, err := s.Evidence.Get(ctx, eid)
			if err != nil {
				return err
			}
			if rec == nil {
				return fmt.Errorf("evidence %s not found", eid)
			}
		}
	}
	return nil
}

func defaultAuthority(k api.MemoryKind) int {
	switch k {
	case api.MemoryKindConstraint:
		return 8
	case api.MemoryKindDecision:
		return 7
	case api.MemoryKindFailure:
		return 6
	case api.MemoryKindPattern:
		return 5
	default:
		return 5
	}
}

func isBehaviorKind(k api.MemoryKind) bool {
	switch k {
	case api.MemoryKindState, api.MemoryKindDecision, api.MemoryKindFailure, api.MemoryKindPattern, api.MemoryKindConstraint:
		return true
	default:
		return false
	}
}

// Materialize creates a memory object from a digest candidate's proposal_json and marks the candidate promoted.
func (s *Service) Materialize(ctx context.Context, candidateID uuid.UUID) (*MaterializeOutcome, error) {
	return s.materializeInternal(ctx, candidateID, false)
}

func (s *Service) materializeInternal(ctx context.Context, candidateID uuid.UUID, auto bool) (*MaterializeOutcome, error) {
	if s.Memory == nil {
		return nil, fmt.Errorf("memory service required for materialize")
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
	if len(c.ProposalJSON) == 0 {
		return nil, fmt.Errorf("candidate has no structured proposal_json; use empty-body promote or run digest first")
	}
	var p ProposalPayloadV1
	if err := json.Unmarshal(c.ProposalJSON, &p); err != nil {
		return nil, fmt.Errorf("proposal_json: %w", err)
	}

	val := s.ValidatePromotionCandidate(ctx, c, &p)
	if !val.Allow {
		return nil, fmt.Errorf("promotion validation: %s", val.Reason)
	}

	promo := s.Promotion
	status := api.StatusActive
	if promo != nil && promo.RequireReview {
		status = api.StatusPending
	}

	app := api.ApplicabilityAdvisory
	if p.Kind == api.MemoryKindConstraint {
		app = api.ApplicabilityGoverning
	}
	req := memory.CreateRequest{
		Kind:          p.Kind,
		Authority:     p.ProposedAuthority,
		Statement:     p.Statement,
		Tags:          p.Tags,
		Status:        status,
		Applicability: app,
	}
	if req.Authority <= 0 {
		req.Authority = defaultAuthority(p.Kind)
	}
	if pl := buildMaterializePayload(candidateID, &p); pl != nil {
		req.Payload = pl
	}
	if sid := strings.TrimSpace(p.SupersedesMemoryID); sid != "" {
		u, err := uuid.Parse(sid)
		if err != nil {
			return nil, fmt.Errorf("supersedes_memory_id: %w", err)
		}
		req.SupersedesID = &u
	}

	var preDup *uuid.UUID
	if s.MemoryDup != nil {
		sk := memorynorm.StatementKey(p.Statement)
		if sk != "" {
			preDup, _ = s.MemoryDup.FindActiveDuplicate(ctx, p.Kind, sk)
		}
	}

	ccfg := memory.NormalizeCanonicalConsolidation(&memory.CanonicalConsolidationConfig{Enabled: false})
	if promo != nil && promo.CanonicalConsolidation != nil {
		ccfg = memory.NormalizeCanonicalConsolidation(promo.CanonicalConsolidation)
	}

	supersedesSet := strings.TrimSpace(p.SupersedesMemoryID) != ""

	if !supersedesSet && ccfg.Enabled {
		dec, err := s.Memory.FindCanonicalConsolidationMatch(ctx, ccfg, &memory.ConsolidationProposalInput{
			Kind:      p.Kind,
			Statement: p.Statement,
			Tags:      p.Tags,
		})
		if err != nil {
			return nil, err
		}
		switch dec.Kind {
		case memory.ConsolidationReinforce:
			if dec.TargetID == nil {
				break
			}
			var plb []byte
			if pl := buildMaterializePayload(candidateID, &p); pl != nil {
				plb = *pl
			}
			obj, err := s.Memory.ConsolidatePromotion(ctx, memory.ConsolidatePromotionRequest{
				TargetID:           *dec.TargetID,
				CandidateID:        candidateID,
				IncomingTags:       p.Tags,
				AuthorityDelta:     ccfg.AuthorityDeltaPerReinforcement,
				Jaccard:            dec.Jaccard,
				Reason:             dec.Reason,
				ExactKeyMatch:      dec.ExactStatementKey,
				MaterializePayload: plb,
			})
			if err != nil {
				return nil, err
			}
			if len(p.EvidenceIDs) > 0 {
				if s.Evidence == nil {
					return nil, fmt.Errorf("evidence service required to link evidence_ids")
				}
				if err := s.Evidence.LinkPromotedEvidence(ctx, obj.ID, p.EvidenceIDs); err != nil {
					return nil, err
				}
			}
			if err := s.Repo.UpdatePromotionStatus(ctx, candidateID, "promoted"); err != nil {
				return nil, err
			}
			if auto {
				rd, msg := ClassifyPromotionReadiness(&p, c.SalienceScore)
				slog.Info(strings.TrimSpace(fmt.Sprintf("[AUTO PROMOTE] candidate_id=%s memory_id=%s readiness=%s reason=%s",
					candidateID.String(), obj.ID.String(), rd, msg)))
			}
			sid := dec.TargetID.String()
			return &MaterializeOutcome{
				Memory:                   obj,
				Created:                  false,
				ConsolidatedIntoMemoryID: &sid,
				Strengthened:             true,
				ConsolidationReason:      dec.Reason,
			}, nil

		case memory.ConsolidationContradictNew:
			obj, err := s.Memory.Create(ctx, req)
			if err != nil {
				return nil, err
			}
			if s.Relationships != nil && dec.ConflictTargetID != nil {
				if _, err := s.Relationships.CreateRelationship(ctx, obj.ID, *dec.ConflictTargetID, memory.RelContradicts,
					dec.Reason, "curation_materialize_contradiction"); err != nil {
					return nil, fmt.Errorf("contradiction edge: %w", err)
				}
			}
			if len(p.EvidenceIDs) > 0 {
				if s.Evidence == nil {
					return nil, fmt.Errorf("evidence service required to link evidence_ids")
				}
				if err := s.Evidence.LinkPromotedEvidence(ctx, obj.ID, p.EvidenceIDs); err != nil {
					return nil, err
				}
			}
			if err := s.Repo.UpdatePromotionStatus(ctx, candidateID, "promoted"); err != nil {
				return nil, err
			}
			if auto {
				rd, msg := ClassifyPromotionReadiness(&p, c.SalienceScore)
				slog.Info(strings.TrimSpace(fmt.Sprintf("[AUTO PROMOTE] candidate_id=%s memory_id=%s readiness=%s reason=%s",
					candidateID.String(), obj.ID.String(), rd, msg)))
			}
			cid := dec.ConflictTargetID.String()
			return &MaterializeOutcome{
				Memory:              obj,
				Created:             true,
				Strengthened:        false,
				ConsolidationReason: dec.Reason,
				ContradictsMemoryID: &cid,
			}, nil
		}
	}

	obj, err := s.Memory.Create(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(p.EvidenceIDs) > 0 {
		if s.Evidence == nil {
			return nil, fmt.Errorf("evidence service required to link evidence_ids")
		}
		if err := s.Evidence.LinkPromotedEvidence(ctx, obj.ID, p.EvidenceIDs); err != nil {
			return nil, err
		}
	}
	if err := s.Repo.UpdatePromotionStatus(ctx, candidateID, "promoted"); err != nil {
		return nil, err
	}
	if auto {
		rd, msg := ClassifyPromotionReadiness(&p, c.SalienceScore)
		slog.Info(strings.TrimSpace(fmt.Sprintf("[AUTO PROMOTE] candidate_id=%s memory_id=%s readiness=%s reason=%s",
			candidateID.String(), obj.ID.String(), rd, msg)))
	}
	out := &MaterializeOutcome{
		Memory:       obj,
		Created:      true,
		Strengthened: false,
	}
	if preDup != nil && obj != nil && obj.ID == *preDup {
		out.Created = false
		out.Strengthened = true
		sid := preDup.String()
		out.ConsolidatedIntoMemoryID = &sid
		out.ConsolidationReason = "memory_create_exact_dedup"
	}
	return out, nil
}
