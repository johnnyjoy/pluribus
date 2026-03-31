package similarity

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"
)

// Config holds runtime policy for the similarity service (from app.Config.Similarity).
type Config struct {
	Enabled         bool
	MaxSummaryBytes int
	MaxEpisodesScan int
	MaxResults      int
	// MinResemblance is the minimum combined lexical (+ optional tag) score in [0,1] to include a row.
	MinResemblance float64
	// McpDedupEnabled when true, duplicate MCP source episodes with same summary + correlation session within McpDedupWindow return the existing row (no insert; no auto-distill replay).
	McpDedupEnabled bool
	// McpDedupWindow bounds duplicate detection for source=mcp (default 120s when zero and McpDedupEnabled).
	McpDedupWindow time.Duration
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
	tags := append([]string(nil), req.Tags...)
	corr := strings.TrimSpace(req.CorrelationID)
	if corr != "" {
		tags = append(tags, "mcp:session:"+corr)
	}
	if src == "mcp" && s.Config != nil && s.Config.McpDedupEnabled && s.Repo != nil {
		win := s.Config.McpDedupWindow
		if win <= 0 {
			win = 120 * time.Second
		}
		dup, err := s.Repo.FindMcpDuplicateInWindow(ctx, req.Summary, corr, win)
		if err != nil {
			return nil, err
		}
		if dup != nil {
			dup.Deduplicated = true
			return dup, nil
		}
	}
	rec := &Record{
		SummaryText:     req.Summary,
		Source:          src,
		Tags:            tags,
		RelatedMemoryID: req.RelatedMemoryID,
		OccurredAt:      req.OccurredAt,
		Entities:        normalizeEntityList(req.Entities),
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

	if err := validateSimilarTimeWindow(req.OccurredAfter, req.OccurredBefore); err != nil {
		return nil, err
	}

	candidates, err := s.Repo.ListCandidates(ctx, scan, req.OccurredAfter, req.OccurredBefore)
	if err != nil {
		return nil, err
	}

	entityFilters := mergeEntityFilters(req.Entity, req.Entities)

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
		if len(entityFilters) > 0 && !entityOverlap(entityFilters, rec.Entities) {
			continue
		}
		base, sig := resemblanceScore(q, rec.SummaryText, req.Tags, rec.Tags)
		entJ := tagJaccard(entityFilters, rec.Entities)
		if len(entityFilters) > 0 && len(rec.Entities) > 0 && entJ > 0 {
			sig = append(sig, "entity_overlap")
		}
		score := base + 0.12*entJ
		if tp, ok := timeProximityBoost(effectiveEpisodeTime(rec), req.OccurredAfter, req.OccurredBefore); ok {
			score += tp
			sig = append(sig, "time_proximity")
		}
		if score > 1 {
			score = 1
		}
		if req.OccurredAfter != nil || req.OccurredBefore != nil {
			sig = append(sig, "time_window_filter")
		}
		if score < minScore {
			continue
		}
		out = append(out, scored{rec: rec, sim: score, signals: sig})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].sim != out[j].sim {
			return out[i].sim > out[j].sim
		}
		return effectiveEpisodeTime(out[i].rec).After(effectiveEpisodeTime(out[j].rec))
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
			OccurredAt:         sc.rec.OccurredAt,
			Entities:           sc.rec.Entities,
		}
		if sc.rec.RelatedMemoryID != nil {
			sid := sc.rec.RelatedMemoryID.String()
			item.RelatedMemoryID = &sid
		}
		resp.AdvisorySimilarCases = append(resp.AdvisorySimilarCases, item)
	}
	return resp, nil
}

func effectiveEpisodeTime(rec Record) time.Time {
	if rec.OccurredAt != nil {
		return *rec.OccurredAt
	}
	return rec.CreatedAt
}

// validateSimilarTimeWindow rejects an empty or inverted window (after strictly after before).
func validateSimilarTimeWindow(after, before *time.Time) error {
	if after == nil || before == nil {
		return nil
	}
	if after.After(*before) {
		return errors.New("similarity: occurred_after must be on or before occurred_before")
	}
	return nil
}

// timeProximityBoost adds a small score when both bounds are set, favoring episodes near the
// midpoint of [after, before] (advisory tie-breaker; not canonical authority).
func timeProximityBoost(eff time.Time, after, before *time.Time) (float64, bool) {
	if after == nil || before == nil {
		return 0, false
	}
	span := before.Sub(*after)
	if span < 0 {
		return 0, false
	}
	const maxBoost = 0.03
	if span == 0 {
		if eff.Equal(*after) || eff.Equal(*before) {
			return maxBoost, true
		}
		return 0, false
	}
	center := after.Add(span / 2)
	half := span / 2
	if half <= 0 {
		return 0, false
	}
	d := eff.Sub(center)
	if d < 0 {
		d = -d
	}
	frac := float64(d) / float64(half)
	if frac > 1 {
		frac = 1
	}
	return (1 - frac) * maxBoost, true
}

func normalizeEntityList(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	const maxEnt = 64
	seen := make(map[string]struct{})
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			continue
		}
		if len(s) > 128 {
			s = s[:128]
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
		if len(out) >= maxEnt {
			break
		}
	}
	return out
}

func mergeEntityFilters(entity string, entities []string) []string {
	var raw []string
	if strings.TrimSpace(entity) != "" {
		raw = append(raw, entity)
	}
	raw = append(raw, entities...)
	return normalizeEntityList(raw)
}

// entityOverlap is true if any normalized filter entity appears in episode entities.
func entityOverlap(filter []string, episode []string) bool {
	if len(filter) == 0 {
		return true
	}
	set := tagSet(episode)
	for _, f := range filter {
		if _, ok := set[f]; ok {
			return true
		}
	}
	return false
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
