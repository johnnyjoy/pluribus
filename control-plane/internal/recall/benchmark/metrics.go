package benchmark

import (
	"context"
	"fmt"
	"strings"

	"control-plane/internal/memory"
	"control-plane/internal/recall"

	"github.com/google/uuid"
)

// RankedHit is one recalled memory with diagnostic fields.
type RankedHit struct {
	Rank        int     `json:"rank"`
	Label       string  `json:"label"`
	MemoryID    string  `json:"memory_id"`
	Domain      string  `json:"domain"`
	Kind        string  `json:"kind"`
	Authority   int     `json:"authority"`
	Score       float64 `json:"score"`
	Reason      string  `json:"reason,omitempty"`
	Bucket      string  `json:"bundle_bucket,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Forbidden   bool    `json:"forbidden,omitempty"`
	Expected    bool    `json:"expected,omitempty"`
	Acceptable  bool    `json:"acceptable,omitempty"`
	RelevanceScore      float64 `json:"relevance_score,omitempty"`
	LexicalScore        float64 `json:"lexical_score,omitempty"`
	SituationalScore    float64 `json:"situational_score,omitempty"`
	TagMatchScore       float64 `json:"tag_match_score,omitempty"`
	AuthorityMultiplier float64 `json:"authority_multiplier,omitempty"`
	WrongDomainPenalty  float64 `json:"wrong_domain_penalty,omitempty"`
	GenericTermPenalty  float64 `json:"generic_term_penalty,omitempty"`
	MatchedTerms        []string `json:"matched_terms,omitempty"`
}

// CaseMetrics holds computed metrics for one case.
type CaseMetrics struct {
	RecallAtK           float64 `json:"recall_at_k"`
	PrecisionAtK        float64 `json:"precision_at_k"`
	ForbiddenHitCount   int     `json:"forbidden_hit_count"`
	ForbiddenHitRate    float64 `json:"forbidden_hit_rate"`
	ExpectedMissing     []string `json:"expected_missing"`
	ForbiddenReturned   []string `json:"forbidden_returned"`
	AverageRankExpected float64 `json:"average_rank_of_expected"`
	MRR                 float64 `json:"mrr"`
	PrimaryExpectedRank int     `json:"primary_expected_rank,omitempty"`
	Violations          ViolationMetrics `json:"violations"`
}

// CaseResult is the outcome of one benchmark case.
type CaseResult struct {
	Case       FixtureCase  `json:"case"`
	Hits       []RankedHit  `json:"hits"`
	Metrics    CaseMetrics  `json:"metrics"`
	Passed     bool         `json:"passed"`
	FailReason string       `json:"fail_reason,omitempty"`
	Semantic   *SemanticStatus `json:"semantic,omitempty"`
	LiveStats  *LiveEmbedStats `json:"live_embedder,omitempty"`
}

// SemanticStatus records retrieval mode for honesty reporting.
type SemanticStatus struct {
	Attempted      bool   `json:"attempted"`
	Path           string `json:"path"`
	FallbackReason string `json:"fallback_reason,omitempty"`
	EmbedderAvailable bool `json:"embedder_available"`
}

// RunConfig controls benchmark execution.
type RunConfig struct {
	MaxPerKind    int
	MaxTotal      int
	Gate          bool
	RetrievalMode string // lexical | hybrid | hybrid_live
	HybridGate    bool   // enforce zero lifecycle/date/utility/mode violations
	LiveEmbedder  memory.Embedder
	LiveProfile   memory.EmbeddingConfigProfile
	LiveEnv       memory.LiveEmbedderEnvConfig
}

const (
	RetrievalModeLexical    = "lexical"
	RetrievalModeHybrid     = "hybrid"
	RetrievalModeHybridLive = "hybrid_live"
)

// DefaultRunConfig returns standard compile limits for benchmarks.
func DefaultRunConfig() RunConfig {
	return RunConfig{MaxPerKind: 10, MaxTotal: 40, Gate: false, RetrievalMode: RetrievalModeLexical}
}

// RunCase executes recall for one fixture case.
func RunCase(ctx context.Context, lc *LoadedCorpus, c FixtureCase, cfg RunConfig) (*CaseResult, error) {
	if cfg.MaxPerKind <= 0 {
		cfg.MaxPerKind = 10
	}
	if cfg.MaxTotal <= 0 {
		cfg.MaxTotal = 40
	}
	w := recall.DefaultRankingWeights()
	var liveStub *LiveHybridMemoryStub
	compiler := &recall.Compiler{
		Ranking: &w,
	}
	mode := strings.TrimSpace(cfg.RetrievalMode)
	if mode == "" {
		mode = RetrievalModeLexical
	}
	if mode == RetrievalModeHybrid {
		compiler.Memory = NewHybridMemoryStub(lc)
		compiler.Semantic = &recall.SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       50,
			MinCosineSimilarity: 0.25,
		}
	} else if mode == RetrievalModeHybridLive {
		if cfg.LiveEmbedder == nil {
			return nil, fmt.Errorf("hybrid_live requires explicit live embedder config")
		}
		st, err := NewLiveHybridMemoryStub(lc, cfg.LiveEmbedder, cfg.LiveProfile)
		if err != nil {
			return nil, err
		}
		liveStub = st
		compiler.Memory = liveStub
		compiler.Semantic = &recall.SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       50,
			MinCosineSimilarity: 0.25,
		}
	} else {
		compiler.Memory = &MemoryStub{All: lc.Objects}
		compiler.Semantic = &recall.SemanticRecallConfig{Enabled: false}
	}
	if util := buildUtilityStub(lc); util != nil {
		compiler.Utility = util
		compiler.UtilityWeight = 0.3
	}
	if contra := buildContradictionStub(lc); contra != nil {
		compiler.Contradiction = contra
	}
	req := recall.CompileRequest{
		RetrievalQuery: c.Query,
		Tags:           c.Tags,
		RepoRoot:       c.RepoRootHint,
		MaxPerKind:     cfg.MaxPerKind,
		MaxTotal:       cfg.MaxTotal,
		Mode:           "continuity",
		RecallMode:     c.RecallMode,
		IncludeStatus:  append([]string(nil), c.IncludeStatus...),
		OccurredAfter:  c.OccurredAfter,
		OccurredBefore: c.OccurredBefore,
	}
	bundle, err := compiler.Compile(ctx, req)
	if err != nil {
		return nil, err
	}
	flat := recall.FlattenBundleByScore(bundle)
	k := c.K
	if k <= 0 {
		k = 10
	}
	if len(flat) > k {
		flat = flat[:k]
	}
	expected := labelSet(c.ExpectedLabels)
	acceptable := labelSet(c.AcceptableLabels)
	forbidden := labelSet(c.ForbiddenLabels)

	hits := make([]RankedHit, 0, len(flat))
	for i, it := range flat {
		id, _ := uuid.Parse(it.ID)
		lbl := lc.IDToLabel[id]
		dom := lc.IDToDomain[id]
		fix, _ := lc.IDToFixture[id]
		score, reason := float64(0), ""
		if it.Justification != nil {
			score = it.Justification.Score
			reason = it.Justification.Reason
		}
		h := RankedHit{
			Rank:      i + 1,
			Label:     lbl,
			MemoryID:  it.ID,
			Domain:    dom,
			Kind:      it.Kind,
			Authority: it.Authority,
			Score:     score,
			Reason:    reason,
			Bucket:    bucketOf(it),
			Tags:      fix.Tags,
			Forbidden: forbidden[lbl],
			Expected:  expected[lbl],
			Acceptable: acceptable[lbl],
		}
		if it.Justification != nil && it.Justification.Components != nil {
			c := it.Justification.Components
			h.RelevanceScore = c.RelevanceScore
			h.LexicalScore = c.LexicalScore
			h.SituationalScore = c.SituationalScore
			h.TagMatchScore = c.TagMatchScore
			h.AuthorityMultiplier = c.AuthorityMultiplier
			h.WrongDomainPenalty = c.WrongDomainPenalty
			h.GenericTermPenalty = c.GenericTermPenalty
			h.MatchedTerms = c.MatchedTerms
		}
		hits = append(hits, h)
	}
	metrics := ComputeMetrics(c, hits)
	metrics.Violations = ComputeViolations(c, lc, hits)
	res := &CaseResult{
		Case:    c,
		Hits:    hits,
		Metrics: metrics,
	}
	if bundle.SemanticRetrieval != nil {
		res.Semantic = &SemanticStatus{
			Attempted:      bundle.SemanticRetrieval.Attempted,
			Path:           bundle.SemanticRetrieval.Path,
			FallbackReason: bundle.SemanticRetrieval.FallbackReason,
			EmbedderAvailable: bundle.SemanticRetrieval.Path == "semantic_hybrid",
		}
	} else {
		res.Semantic = &SemanticStatus{Path: "lexical_only", EmbedderAvailable: false}
	}
	if liveStub != nil {
		st := liveStub.Stats()
		res.LiveStats = &st
	}
	if cfg.Gate || cfg.HybridGate {
		res.Passed, res.FailReason = GateCase(c, metrics, len(hits))
		if cfg.HybridGate && res.Passed {
			if vr := GateViolations(metrics.Violations); vr != "" {
				res.Passed = false
				res.FailReason = vr
			}
		}
	} else {
		res.Passed = true
	}
	return res, nil
}

func bucketOf(it recall.MemoryItem) string {
	return it.Kind
}

func labelSet(labels []string) map[string]bool {
	m := map[string]bool{}
	for _, l := range labels {
		m[l] = true
	}
	return m
}

// ComputeMetrics calculates Recall@K, Precision@K, forbidden hits.
func ComputeMetrics(c FixtureCase, hits []RankedHit) CaseMetrics {
	expected := append([]string{}, c.ExpectedLabels...)
	expectedSet := labelSet(expected)
	acceptableSet := labelSet(c.AcceptableLabels)
	forbiddenSet := labelSet(c.ForbiddenLabels)

	hitLabels := map[string]int{}
	for _, h := range hits {
		hitLabels[h.Label] = h.Rank
	}

	var missing []string
	hitsExpected := 0
	var rankSum float64
	var rankCount int
	primaryRank := 0
	if len(c.ExpectedLabels) > 0 {
		if r, ok := hitLabels[c.ExpectedLabels[0]]; ok {
			primaryRank = r
		}
	}
	for _, lbl := range c.ExpectedLabels {
		if r, ok := hitLabels[lbl]; ok {
			hitsExpected++
			rankSum += float64(r)
			rankCount++
		} else {
			missing = append(missing, lbl)
		}
	}
	denomExpected := len(c.ExpectedLabels)
	recallAtK := 0.0
	if denomExpected > 0 {
		recallAtK = float64(hitsExpected) / float64(denomExpected)
	}
	goodInTopK := 0
	var forbiddenReturned []string
	forbiddenCount := 0
	for _, h := range hits {
		if expectedSet[h.Label] || acceptableSet[h.Label] {
			goodInTopK++
		}
		if forbiddenSet[h.Label] {
			forbiddenCount++
			forbiddenReturned = append(forbiddenReturned, h.Label)
		}
	}
	precisionAtK := 0.0
	if len(hits) > 0 {
		precisionAtK = float64(goodInTopK) / float64(len(hits))
	}
	forbiddenRate := 0.0
	if len(hits) > 0 {
		forbiddenRate = float64(forbiddenCount) / float64(len(hits))
	}
	avgRank := 0.0
	if rankCount > 0 {
		avgRank = rankSum / float64(rankCount)
	}
	mrr := 0.0
	if primaryRank > 0 {
		mrr = 1.0 / float64(primaryRank)
	}
	return CaseMetrics{
		RecallAtK:           recallAtK,
		PrecisionAtK:        precisionAtK,
		ForbiddenHitCount:   forbiddenCount,
		ForbiddenHitRate:    forbiddenRate,
		ExpectedMissing:     missing,
		ForbiddenReturned:   forbiddenReturned,
		AverageRankExpected: avgRank,
		MRR:                 mrr,
		PrimaryExpectedRank: primaryRank,
	}
}

// GateCase returns pass/fail against case thresholds.
func GateCase(c FixtureCase, m CaseMetrics, hitCount int) (bool, string) {
	if c.RequireResults && hitCount == 0 {
		return false, "no recall results"
	}
	if m.RecallAtK+1e-9 < c.MinimumRecallAtK {
		return false, fmt.Sprintf("recall_at_%d=%.2f required=%.2f", c.K, m.RecallAtK, c.MinimumRecallAtK)
	}
	if m.PrecisionAtK+1e-9 < c.MinimumPrecisionAtK {
		return false, fmt.Sprintf("precision_at_%d=%.2f required=%.2f", c.K, m.PrecisionAtK, c.MinimumPrecisionAtK)
	}
	if m.ForbiddenHitCount > c.MaximumForbiddenHits {
		return false, fmt.Sprintf("forbidden_hit_count=%d max=%d", m.ForbiddenHitCount, c.MaximumForbiddenHits)
	}
	return true, ""
}

// GateViolations returns fail reason when hybrid safety violations exist.
func GateViolations(v ViolationMetrics) string {
	if v.LifecycleViolationCount > 0 {
		return fmt.Sprintf("lifecycle_violation_count=%d", v.LifecycleViolationCount)
	}
	if v.DateBoundViolationCount > 0 {
		return fmt.Sprintf("date_bound_violation_count=%d", v.DateBoundViolationCount)
	}
	if v.UtilityViolationCount > 0 {
		return fmt.Sprintf("utility_violation_count=%d", v.UtilityViolationCount)
	}
	if v.ModeViolationCount > 0 {
		return fmt.Sprintf("mode_violation_count=%d", v.ModeViolationCount)
	}
	return ""
}

// RunAll executes every case in the corpus.
func RunAll(ctx context.Context, lc *LoadedCorpus, cfg RunConfig) ([]CaseResult, error) {
	var out []CaseResult
	for _, c := range lc.Cases {
		if c.HybridOnly && cfg.RetrievalMode != RetrievalModeHybrid {
			continue
		}
		r, err := RunCase(ctx, lc, c, cfg)
		if err != nil {
			return nil, fmt.Errorf("case %s: %w", c.ID, err)
		}
		out = append(out, *r)
	}
	return out, nil
}

// DomainConfusionMatrix builds expected_domain x returned_domain counts from case results.
func DomainConfusionMatrix(results []CaseResult) map[string]map[string]int {
	matrix := map[string]map[string]int{}
	for _, r := range results {
		exp := strings.TrimSpace(r.Case.ExpectedDomain)
		if exp == "" {
			exp = inferDomainFromLabels(r.Case.ExpectedLabels)
		}
		if matrix[exp] == nil {
			matrix[exp] = map[string]int{}
		}
		for _, h := range r.Hits {
			matrix[exp][h.Domain]++
		}
	}
	return matrix
}

func inferDomainFromLabels(labels []string) string {
	if len(labels) == 0 {
		return "unknown"
	}
	parts := strings.SplitN(labels[0], "_", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return "unknown"
}
