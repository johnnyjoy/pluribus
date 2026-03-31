package distillation

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"control-plane/internal/curation"
	"control-plane/internal/memorynorm"
	"control-plane/internal/similarity"

	"github.com/google/uuid"
)

// Wire values for proposal_json.pluribus_distill_origin (distill mode); see docs/memory-doctrine.md Terminology.
const (
	originManual  = "manual"   // explicit distill (concept: explicit)
	originAuto    = "auto"     // auto_from_advisory
	originAutoMCP = "auto:mcp" // auto_from_advisory_mcp when ingest channel is mcp
	originMixed   = "mixed"
)

const (
	distilledSalienceScore   = 0.55
	distilledProposedAuth    = 3
	maxDistilledStatementLen = 512
	defaultMinStmtChars      = 20
	mergeSalienceBump        = 0.04
	maxMergedSalience        = 0.92
)

// Config gates POST /v1/episodes/distill.
type Config struct {
	Enabled bool
	// AutoFromAdvisoryEpisodes runs distillation after each advisory episode create (same logic as explicit distill).
	AutoFromAdvisoryEpisodes bool
	// MinStatementChars is minimum trimmed statement length to emit a candidate (0 = default 20).
	MinStatementChars int
}

// Service writes distilled rows to candidate_events via curation.Repo.CreateDigest / merge updates.
type Service struct {
	Curation *curation.Repo
	Episodes *similarity.Repo
	Config   *Config
}

func (s *Service) minChars() int {
	n := 0
	if s != nil && s.Config != nil {
		n = s.Config.MinStatementChars
	}
	if n <= 0 {
		return defaultMinStmtChars
	}
	return n
}

// Distill turns advisory episode text into pending candidate_events (never memories). Explicit HTTP path uses origin "manual".
func (s *Service) Distill(ctx context.Context, req DistillRequest) (*DistillResponse, error) {
	req.OriginDistill = originManual
	return s.distillWithOrigin(ctx, req)
}

// DistillAfterAdvisoryIngest runs the same extraction as Distill for a persisted episode id when auto-from-advisory is enabled in config.
func (s *Service) DistillAfterAdvisoryIngest(ctx context.Context, episodeID uuid.UUID) error {
	if s == nil || s.Config == nil || !s.Config.Enabled || !s.Config.AutoFromAdvisoryEpisodes {
		return nil
	}
	origin := originAuto
	if s.Episodes != nil {
		rec, err := s.Episodes.GetByID(ctx, episodeID)
		if err == nil && rec != nil && rec.Source == "mcp" {
			origin = originAutoMCP
		}
	}
	_, err := s.distillWithOrigin(ctx, DistillRequest{EpisodeID: episodeID.String(), OriginDistill: origin})
	return err
}

func distillOriginMerge(existing, incoming string) string {
	if existing == "" {
		existing = originManual
	}
	if incoming == "" {
		incoming = originManual
	}
	if existing == incoming {
		return existing
	}
	return originMixed
}

// distillWithOrigin is the single distillation path for explicit and automatic runs.
func (s *Service) distillWithOrigin(ctx context.Context, req DistillRequest) (*DistillResponse, error) {
	if s == nil || s.Config == nil || !s.Config.Enabled {
		return nil, ErrDistillationDisabled
	}
	if s.Curation == nil || s.Curation.DB == nil {
		return nil, errors.New("distillation: curation repo not configured")
	}
	origin := req.OriginDistill
	if origin == "" {
		origin = originManual
	}

	var summary string
	var tags []string
	var sourceEpisodeID string
	var entities []string
	var episodeSource string

	switch {
	case strings.TrimSpace(req.EpisodeID) != "":
		if s.Episodes == nil || s.Episodes.DB == nil {
			return nil, errors.New("distillation: advisory episode repo not configured")
		}
		id, err := uuid.Parse(strings.TrimSpace(req.EpisodeID))
		if err != nil {
			return nil, fmt.Errorf("distillation: invalid episode_id: %w", err)
		}
		rec, err := s.Episodes.GetByID(ctx, id)
		if err != nil {
			return nil, err
		}
		if rec == nil {
			return nil, ErrEpisodeNotFound
		}
		summary = rec.SummaryText
		tags = append([]string(nil), rec.Tags...)
		entities = rec.Entities
		sourceEpisodeID = rec.ID.String()
		episodeSource = rec.Source
	case strings.TrimSpace(req.Summary) != "":
		summary = strings.TrimSpace(req.Summary)
		tags = append([]string(nil), req.Tags...)
		entities = req.Entities
	default:
		return nil, errors.New("distillation: episode_id or summary required")
	}

	lower := normalizeLower(summary)
	drafts := extractDrafts(lower)
	if len(drafts) == 0 {
		return &DistillResponse{Candidates: nil}, nil
	}

	baseTags := distilledTags(tags, entities, episodeSource)
	minLen := s.minChars()
	var out []DistillCandidateOut

	for _, d := range drafts {
		stmt := clipStatement(summary, maxDistilledStatementLen)
		if len(strings.TrimSpace(stmt)) < minLen {
			continue
		}
		stmtKey := memorynorm.StatementKey(stmt)
		if stmtKey == "" {
			continue
		}

		existing, err := s.Curation.FindPendingDistilledByKindAndStatementKey(ctx, string(d.kind), stmtKey)
		if err != nil {
			return nil, err
		}

		if existing != nil {
			merged, err := s.mergeIntoExisting(ctx, existing, d, stmt, stmtKey, sourceEpisodeID, baseTags, origin)
			if err != nil {
				return nil, err
			}
			out = append(out, merged)
			continue
		}

		propID := uuid.New().String()
		var srcIDs []string
		if sourceEpisodeID != "" {
			srcIDs = []string{sourceEpisodeID}
		}
		payload := curation.ProposalPayloadV1{
			V:                        1,
			Kind:                     d.kind,
			Statement:                stmt,
			Reason:                   d.reason,
			Tags:                     append([]string(nil), baseTags...),
			ProposedAuthority:        distilledProposedAuth,
			ProposalID:               propID,
			SourceAdvisoryEpisodeID:  sourceEpisodeID,
			DistillStatementKey:      stmtKey,
			SourceAdvisoryEpisodeIDs: srcIDs,
			DistillSupportCount:      1,
			PluribusDistillOrigin:    origin,
		}
		pj, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		raw := distillRawText(stmt, 1, srcIDs)
		c, err := s.Curation.CreateDigest(ctx, raw, distilledSalienceScore, pj)
		if err != nil {
			return nil, err
		}
		out = append(out, DistillCandidateOut{
			CandidateID:              c.ID.String(),
			Kind:                     string(d.kind),
			Statement:                stmt,
			Reason:                   d.reason,
			Tags:                     baseTags,
			SourceAdvisoryEpisodeID:  sourceEpisodeID,
			SourceAdvisoryEpisodeIDs: srcIDs,
			SalienceScore:            distilledSalienceScore,
			DistillSupportCount:      1,
			Merged:                   false,
		})
	}
	if len(out) == 0 {
		return &DistillResponse{Candidates: nil}, nil
	}
	return &DistillResponse{Candidates: out}, nil
}

func (s *Service) mergeIntoExisting(ctx context.Context, existing *curation.CandidateEvent, d draft, stmt, stmtKey, sourceEpisodeID string, baseTags []string, incomingOrigin string) (DistillCandidateOut, error) {
	var p curation.ProposalPayloadV1
	if err := json.Unmarshal(existing.ProposalJSON, &p); err != nil {
		return DistillCandidateOut{}, err
	}
	if p.DistillSupportCount < 1 {
		p.DistillSupportCount = 1
	}
	ids := append([]string(nil), p.SourceAdvisoryEpisodeIDs...)
	if p.SourceAdvisoryEpisodeID != "" && !containsStr(ids, p.SourceAdvisoryEpisodeID) {
		ids = append(ids, p.SourceAdvisoryEpisodeID)
	}
	if sourceEpisodeID != "" {
		if !containsStr(ids, sourceEpisodeID) {
			ids = append(ids, sourceEpisodeID)
		}
	}
	p.SourceAdvisoryEpisodeIDs = ids
	if p.SourceAdvisoryEpisodeID == "" && len(ids) > 0 {
		p.SourceAdvisoryEpisodeID = ids[0]
	}
	p.DistillSupportCount++
	p.DistillStatementKey = stmtKey
	p.Statement = stmt
	p.Tags = unionTags(p.Tags, baseTags)
	p.PluribusDistillOrigin = distillOriginMerge(p.PluribusDistillOrigin, incomingOrigin)

	newSal := existing.SalienceScore + mergeSalienceBump
	if newSal > maxMergedSalience {
		newSal = maxMergedSalience
	}
	pj, err := json.Marshal(p)
	if err != nil {
		return DistillCandidateOut{}, err
	}
	raw := distillRawText(stmt, p.DistillSupportCount, ids)
	if err := s.Curation.UpdateDigestCandidate(ctx, existing.ID, raw, newSal, pj); err != nil {
		return DistillCandidateOut{}, err
	}
	return DistillCandidateOut{
		CandidateID:              existing.ID.String(),
		Kind:                     string(d.kind),
		Statement:                stmt,
		Reason:                   d.reason,
		Tags:                     baseTags,
		SourceAdvisoryEpisodeID:  p.SourceAdvisoryEpisodeID,
		SourceAdvisoryEpisodeIDs: append([]string(nil), ids...),
		SalienceScore:            newSal,
		DistillSupportCount:      p.DistillSupportCount,
		Merged:                   true,
	}, nil
}

func distillRawText(stmt string, support int, episodeIDs []string) string {
	src := "(inline)"
	if len(episodeIDs) > 0 {
		src = strings.Join(episodeIDs, ", ")
	}
	return fmt.Sprintf("Distilled candidate (support=%d): %s\nAdvisory sources: %s", support, stmt, src)
}

func containsStr(ss []string, x string) bool {
	for _, a := range ss {
		if a == x {
			return true
		}
	}
	return false
}

func unionTags(a, b []string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(xs []string) {
		for _, t := range xs {
			t = strings.TrimSpace(strings.ToLower(t))
			if t == "" {
				continue
			}
			if _, ok := seen[t]; ok {
				continue
			}
			seen[t] = struct{}{}
			out = append(out, t)
		}
	}
	add(a)
	add(b)
	return out
}

func distilledTags(inherited []string, entities []string, episodeSource string) []string {
	seen := make(map[string]struct{})
	var out []string
	add := func(s string) {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	add("distilled-from-advisory")
	if strings.TrimSpace(episodeSource) == "mcp" {
		add("origin:mcp")
	}
	for _, t := range inherited {
		add(t)
	}
	for _, e := range entities {
		add("entity:" + e)
	}
	return out
}

// ErrDistillationDisabled when distillation.enabled is false.
var ErrDistillationDisabled = errors.New("distillation: feature disabled")

// ErrEpisodeNotFound when episode_id does not match a row.
var ErrEpisodeNotFound = errors.New("distillation: advisory episode not found")
