package memory

import (
	"context"
	"log/slog"
	"sort"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) maybeEmbedOnCreate(ctx context.Context, req *CreateRequest) {
	if s == nil || req == nil || s.Semantic == nil || !s.Semantic.RetrievalEnabled() || s.Embedder == nil {
		return
	}
	dim := DefaultEmbeddingDimensions
	if s.Semantic.EmbeddingDimensions > 0 {
		dim = s.Semantic.EmbeddingDimensions
	}
	if s.Embedder.Dimensions() != dim {
		return
	}
	txt := EmbeddingTextForMemory(req.Kind, req.StatementCanonical, req.Statement)
	if strings.TrimSpace(txt) == "" {
		return
	}
	vec, err := s.Embedder.Embed(ctx, txt)
	if err != nil {
		slog.Warn("memory semantic embedding skipped", "err", err)
		return
	}
	if len(vec) != dim {
		slog.Warn("memory semantic embedding dimension mismatch", "got", len(vec), "want", dim)
		return
	}
	model, provider := "text-embedding-3-small", "http"
	if he, ok := s.Embedder.(*HTTPEmbedder); ok {
		model = he.ModelName()
		provider = he.ProviderName()
	}
	wm := NewEmbeddingWriteMeta(req.Kind, req.StatementCanonical, req.Statement, provider, model, len(vec))
	req.Embedding = vec
	req.EmbeddingWrite = &wm
}

// EmbedQueryText embeds retrieval text for recall (semantic candidate query).
// On success returns (vec, "", nil). When embedding is skipped for an expected reason, returns (nil, fallbackReason, nil) with a SemanticFallback* constant.
func (s *Service) EmbedQueryText(ctx context.Context, text string) ([]float32, string, error) {
	if s == nil || s.Semantic == nil || !s.Semantic.RetrievalEnabled() {
		return nil, SemanticFallbackRetrievalDisabled, nil
	}
	if s.Embedder == nil {
		return nil, SemanticFallbackNoEmbedder, nil
	}
	dim := DefaultEmbeddingDimensions
	if s.Semantic.EmbeddingDimensions > 0 {
		dim = s.Semantic.EmbeddingDimensions
	}
	d := s.Embedder.Dimensions()
	if d == 0 {
		return nil, SemanticFallbackNoEmbedder, nil
	}
	if d != dim {
		return nil, SemanticFallbackDimensionMismatch, nil
	}
	t := strings.TrimSpace(text)
	if t == "" {
		return nil, SemanticFallbackEmptyQuery, nil
	}
	vec, err := s.Embedder.Embed(ctx, t)
	if err != nil {
		return nil, "", err
	}
	if len(vec) == 0 {
		return nil, SemanticFallbackEmbeddingFailed, nil
	}
	if len(vec) != dim {
		return nil, SemanticFallbackDimensionMismatch, nil
	}
	return vec, "", nil
}

// SearchSimilarCandidates runs vector similarity search with the same tag/kind/status filters as Search.
// When req.Statuses is set, searches each status and unions results (deduped by ID, best similarity kept).
func (s *Service) SearchSimilarCandidates(ctx context.Context, query []float32, req SearchRequest, limit int, minCosine float64) ([]MemoryObject, map[uuid.UUID]float64, error) {
	if s == nil || s.Repo == nil || len(query) == 0 {
		return nil, nil, nil
	}
	maxDist := 1.0 - minCosine
	if maxDist <= 0 || maxDist > 2 {
		maxDist = 0.65
	}
	if s.Semantic != nil {
		profile := ProfileFromSemanticConfig(s.Semantic)
		req.EmbeddingFilter = &EmbeddingSearchFilter{
			Model:     profile.Model,
			Dimension: profile.Dimension,
		}
	}
	statuses := req.Statuses
	if len(statuses) == 0 {
		st := req.Status
		if st == "" {
			st = "active"
		}
		statuses = []string{st}
	}
	byID := map[uuid.UUID]MemoryObject{}
	sims := map[uuid.UUID]float64{}
	perLimit := limit
	if perLimit <= 0 {
		perLimit = 20
	}
	for _, st := range statuses {
		sub := req
		sub.Status = st
		sub.Statuses = nil
		batch, batchSims, err := s.Repo.SearchSimilar(ctx, query, sub, perLimit, maxDist)
		if err != nil {
			return nil, nil, err
		}
		for _, o := range batch {
			sim := batchSims[o.ID]
			if prev, ok := sims[o.ID]; !ok || sim > prev {
				byID[o.ID] = o
				sims[o.ID] = sim
			}
		}
	}
	if len(byID) == 0 {
		return nil, sims, nil
	}
	type pair struct {
		obj MemoryObject
		sim float64
	}
	pairs := make([]pair, 0, len(byID))
	for id, o := range byID {
		pairs = append(pairs, pair{obj: o, sim: sims[id]})
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].sim != pairs[j].sim {
			return pairs[i].sim > pairs[j].sim
		}
		return pairs[i].obj.ID.String() < pairs[j].obj.ID.String()
	})
	if limit > 0 && len(pairs) > limit {
		pairs = pairs[:limit]
	}
	out := make([]MemoryObject, len(pairs))
	outSims := map[uuid.UUID]float64{}
	for i, p := range pairs {
		out[i] = p.obj
		outSims[p.obj.ID] = p.sim
	}
	return out, outSims, nil
}
