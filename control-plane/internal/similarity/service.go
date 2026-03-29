package similarity

import (
	"context"
	"errors"
	"sort"
	"strings"
)

// Config holds runtime policy for the similarity service (from app.Config.Similarity).
type Config struct {
	Enabled         bool
	MaxSummaryBytes int
	MaxEpisodesScan int
	MaxResults      int
	// MinResemblance is the minimum combined lexical (+ optional tag) score in [0,1] to include a row.
	MinResemblance float64
}

// Service provides create + similar-case retrieval using self-contained lexical/metadata signals.
type Service struct {
	Repo   *Repo
	Config *Config
}

// Create stores a compact advisory episode (lexical/metadata only).
func (s *Service) Create(ctx context.Context, req CreateRequest) (*Record, error) {
	if s == nil || s.Config == nil || !s.Config.Enabled {
		return nil, ErrSimilarityDisabled
	}
	req.Summary = strings.TrimSpace(req.Summary)
	if req.Summary == "" {
		return nil, errors.New("similarity: summary required")
	}
	maxB := s.Config.MaxSummaryBytes
	if maxB <= 0 {
		maxB = 2048
	}
	if len(req.Summary) > maxB {
		req.Summary = req.Summary[:maxB]
	}
	src := strings.TrimSpace(req.Source)
	if src == "" {
		src = "manual"
	}
	if _, ok := ValidSources[src]; !ok {
		return nil, errors.New("similarity: invalid source")
	}
	rec := &Record{
		SummaryText:     req.Summary,
		Source:          src,
		Tags:            req.Tags,
		RelatedMemoryID: req.RelatedMemoryID,
	}
	if err := s.Repo.Create(ctx, rec); err != nil {
		return nil, err
	}
	return rec, nil
}

// FindSimilar returns top advisory episodes by lexical + tag resemblance (subordinate; not canon).
func (s *Service) FindSimilar(ctx context.Context, req SimilarRequest) (*SimilarResponse, error) {
	if s == nil || s.Config == nil || !s.Config.Enabled {
		return &SimilarResponse{AdvisorySimilarCases: nil}, nil
	}
	q := strings.TrimSpace(req.Query)
	if q == "" {
		return nil, errors.New("similarity: query required")
	}
	maxK := req.MaxResults
	if maxK <= 0 && s.Config.MaxResults > 0 {
		maxK = s.Config.MaxResults
	}
	if maxK <= 0 {
		maxK = 5
	}
	scan := s.Config.MaxEpisodesScan
	if scan <= 0 {
		scan = 500
	}
	minScore := s.Config.MinResemblance
	if minScore <= 0 {
		minScore = 0.08
	}

	candidates, err := s.Repo.ListRecent(ctx, scan)
	if err != nil {
		return nil, err
	}

	type scored struct {
		rec     Record
		sim     float64
		signals []string
	}
	var out []scored
	for _, rec := range candidates {
		if len(req.Tags) > 0 && !tagOverlap(req.Tags, rec.Tags) {
			continue
		}
		score, sig := resemblanceScore(q, rec.SummaryText, req.Tags, rec.Tags)
		if score < minScore {
			continue
		}
		out = append(out, scored{rec: rec, sim: score, signals: sig})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].sim != out[j].sim {
			return out[i].sim > out[j].sim
		}
		return out[i].rec.CreatedAt.After(out[j].rec.CreatedAt)
	})
	if len(out) > maxK {
		out = out[:maxK]
	}
	resp := &SimilarResponse{AdvisorySimilarCases: make([]AdvisorySimilarCase, 0, len(out))}
	for _, sc := range out {
		item := AdvisorySimilarCase{
			ID:                 sc.rec.ID.String(),
			Summary:            sc.rec.SummaryText,
			Source:             sc.rec.Source,
			Tags:               sc.rec.Tags,
			ResemblanceScore:   sc.sim,
			ResemblanceSignals: append([]string(nil), sc.signals...),
			Advisory:           true,
			CreatedAt:          sc.rec.CreatedAt,
		}
		if sc.rec.RelatedMemoryID != nil {
			sid := sc.rec.RelatedMemoryID.String()
			item.RelatedMemoryID = &sid
		}
		resp.AdvisorySimilarCases = append(resp.AdvisorySimilarCases, item)
	}
	return resp, nil
}

func tagOverlap(filter []string, episode []string) bool {
	set := make(map[string]struct{}, len(episode))
	for _, t := range episode {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			set[t] = struct{}{}
		}
	}
	for _, t := range filter {
		t = strings.TrimSpace(strings.ToLower(t))
		if t == "" {
			continue
		}
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}
