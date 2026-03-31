package curation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"control-plane/internal/memory"
	"control-plane/internal/similarity"

	"github.com/google/uuid"
)

// ReviewCandidate returns read-only review assistance for a candidate (no writes, no promotion).
func (s *Service) ReviewCandidate(ctx context.Context, id uuid.UUID) (*CandidateReviewResponse, error) {
	if s == nil || s.Repo == nil {
		return nil, fmt.Errorf("curation service not configured")
	}
	c, err := s.Repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if c == nil {
		return nil, fmt.Errorf("candidate not found")
	}

	var p *ProposalPayloadV1
	if len(c.ProposalJSON) > 0 {
		var parsed ProposalPayloadV1
		if err := json.Unmarshal(c.ProposalJSON, &parsed); err == nil {
			p = &parsed
		}
	}

	var tags []string
	if p != nil {
		tags = append([]string(nil), p.Tags...)
	}
	grouped := groupTags(tags)

	out := &CandidateReviewResponse{
		CandidateID:     id.String(),
		PromotionStatus: c.PromotionStatus,
		TagsGrouped:     grouped,
		EntitiesDisplay: append([]string(nil), grouped.Entities...),
	}
	if p != nil {
		r, why := ClassifyPromotionReadiness(p, c.SalienceScore)
		out.PromotionReadiness = r
		out.ReadinessReason = why
	}

	out.Explanation = buildExplanation(p, c.RawText, c.SalienceScore, grouped)
	out.SignalStrength, out.SignalDetail = computeSignalStrength(p, c.SalienceScore)

	out.SupportingEpisodes = loadSupportingSummaries(ctx, s.Episodes, p)
	out.PromotionPreview = buildPromotionPreview(p, s.Promotion)
	out.RelationshipHints = buildRelationshipHints(ctx, s, p)
	out.ConsolidationPreview = buildConsolidationPreviewFromService(ctx, s, p)

	return out, nil
}

func buildConsolidationPreviewFromService(ctx context.Context, s *Service, p *ProposalPayloadV1) *ConsolidationPreview {
	if s == nil || s.Memory == nil || p == nil || !isBehaviorKind(p.Kind) {
		return nil
	}
	ccfg := memory.NormalizeCanonicalConsolidation(&memory.CanonicalConsolidationConfig{Enabled: false})
	if s.Promotion != nil && s.Promotion.CanonicalConsolidation != nil {
		ccfg = memory.NormalizeCanonicalConsolidation(s.Promotion.CanonicalConsolidation)
	}
	if !ccfg.Enabled {
		return &ConsolidationPreview{
			Action:              "create_new",
			ConsolidationReason: "canonical_consolidation_disabled",
			ExpectedEffect:      "Materialize uses memory create (exact dedup still reinforces authority server-side).",
		}
	}
	dec, err := s.Memory.FindCanonicalConsolidationMatch(ctx, ccfg, &memory.ConsolidationProposalInput{
		Kind:      p.Kind,
		Statement: p.Statement,
		Tags:      p.Tags,
	})
	if err != nil || dec == nil {
		return nil
	}
	return buildConsolidationPreview(dec)
}

func buildRelationshipHints(ctx context.Context, s *Service, p *ProposalPayloadV1) []string {
	if p == nil {
		return nil
	}
	var hints []string
	sid := strings.TrimSpace(p.SupersedesMemoryID)
	if sid != "" && s.MemoryLookup != nil {
		uid, err := uuid.Parse(sid)
		if err == nil {
			old, err := s.MemoryLookup.GetByID(ctx, uid)
			if err == nil && old != nil {
				hints = append(hints, fmt.Sprintf("Materialize will supersede prior memory %s (row remains in store; status superseded).", uid.String()))
				if s.Relationships != nil {
					outb, inb, rerr := s.Relationships.ListForMemory(ctx, uid)
					if rerr == nil && len(outb)+len(inb) > 0 {
						hints = append(hints, fmt.Sprintf("That prior memory already has %d typed relationship edge(s) in the lightweight graph.", len(outb)+len(inb)))
					}
				}
			}
		}
	}
	if len(hints) == 0 {
		return nil
	}
	return hints
}

func loadSupportingSummaries(ctx context.Context, repo *similarity.Repo, p *ProposalPayloadV1) []SupportingEpisodeSummary {
	if repo == nil || p == nil {
		return nil
	}
	ids := uniqueEpisodeIDsInOrder(p.SourceAdvisoryEpisodeIDs, p.SourceAdvisoryEpisodeID)
	if len(ids) == 0 {
		return nil
	}
	var out []SupportingEpisodeSummary
	for _, uid := range ids {
		rec, err := repo.GetByID(ctx, uid)
		if err != nil || rec == nil {
			continue
		}
		out = append(out, SupportingEpisodeSummary{
			EpisodeID: uid.String(),
			Summary:   clipText(rec.SummaryText, maxSummaryClip),
		})
	}
	return out
}
