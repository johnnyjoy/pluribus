package memory

import (
	"context"
	"log/slog"
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
	req.Embedding = vec
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

// SearchSimilarCandidates runs vector similarity search with the same tag/kind filters as Search.
func (s *Service) SearchSimilarCandidates(ctx context.Context, query []float32, req SearchRequest, limit int, minCosine float64) ([]MemoryObject, map[uuid.UUID]float64, error) {
	if s == nil || s.Repo == nil || len(query) == 0 {
		return nil, nil, nil
	}
	maxDist := 1.0 - minCosine
	if maxDist <= 0 || maxDist > 2 {
		maxDist = 0.65
	}
	return s.Repo.SearchSimilar(ctx, query, req, limit, maxDist)
}
