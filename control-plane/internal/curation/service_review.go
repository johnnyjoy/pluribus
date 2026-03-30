package curation

import (
	"context"
	"encoding/json"
	"fmt"

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

	return out, nil
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
