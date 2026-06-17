package benchmark

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// LiveHybridMemoryStub uses a real HTTP embedder for benchmark evaluation (opt-in only).
type LiveHybridMemoryStub struct {
	All          []memory.MemoryObject
	LabelToID    map[string]uuid.UUID
	IDToLabel    map[uuid.UUID]string
	Embedder     memory.Embedder
	Profile      memory.EmbeddingConfigProfile
	LabelVector  map[string][]float32
	LabelMeta    map[string]memory.EmbeddingMeta
	StaleLabels  map[string]memory.StalenessReason

	mu             sync.Mutex
	EmbedLatency   []time.Duration
	FallbackCount  int
	StaleSkipCount int
	MissingCount   int
	InitErrors     []string
}

// NewLiveHybridMemoryStub embeds fixture memories with a real embedder at init.
func NewLiveHybridMemoryStub(lc *LoadedCorpus, embedder memory.Embedder, profile memory.EmbeddingConfigProfile) (*LiveHybridMemoryStub, error) {
	if embedder == nil || embedder.Dimensions() <= 0 {
		return nil, fmt.Errorf("live hybrid: embedder not configured")
	}
	h := &LiveHybridMemoryStub{
		All:         lc.Objects,
		LabelToID:   lc.LabelToID,
		IDToLabel:   lc.IDToLabel,
		Embedder:    embedder,
		Profile:     profile,
		LabelVector: map[string][]float32{},
		LabelMeta:   map[string]memory.EmbeddingMeta{},
		StaleLabels: map[string]memory.StalenessReason{},
	}
	ctx := context.Background()
	for id, fix := range lc.IDToFixture {
		lbl := fix.Label
		kind := api.MemoryKind(strings.TrimSpace(fix.Kind))
		if kind == "" {
			kind = api.MemoryKindPattern
		}
		text := memory.EmbeddingTextForMemory(kind, "", fix.Statement)
		obj := memory.MemoryObject{ID: id, Kind: kind, Statement: fix.Statement}
		hash := memory.ComputeEmbeddingSourceHash(kind, "", fix.Statement)
		start := time.Now()
		vec, err := embedder.Embed(ctx, text)
		h.mu.Lock()
		h.EmbedLatency = append(h.EmbedLatency, time.Since(start))
		h.mu.Unlock()
		if err != nil {
			h.InitErrors = append(h.InitErrors, fmt.Sprintf("%s: %v", lbl, err))
			h.MissingCount++
			continue
		}
		if len(vec) == 0 {
			h.MissingCount++
			continue
		}
		model, provider := profile.Model, profile.Provider
		if he, ok := embedder.(*memory.HTTPEmbedder); ok {
			model = he.ModelName()
			provider = he.ProviderName()
		}
		now := time.Now().UTC()
		h.LabelVector[lbl] = vec
		h.LabelMeta[lbl] = memory.EmbeddingMeta{
			Model:      model,
			Provider:   provider,
			Dimension:  len(vec),
			SourceHash: hash,
			Status:     memory.EmbeddingStatusValid,
			CreatedAt:  &now,
			UpdatedAt:  &now,
		}
		_ = obj
	}
	if len(h.LabelVector) == 0 {
		return nil, fmt.Errorf("live hybrid: no fixture embeddings produced (%d errors)", len(h.InitErrors))
	}
	return h, nil
}

func (h *LiveHybridMemoryStub) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	stub := &MemoryStub{All: h.All}
	return stub.Search(ctx, req)
}

func (h *LiveHybridMemoryStub) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	stub := &MemoryStub{All: h.All}
	return stub.SearchMemories(ctx, req)
}

func (h *LiveHybridMemoryStub) EmbedQueryText(ctx context.Context, text string) ([]float32, string, error) {
	start := time.Now()
	vec, err := h.Embedder.Embed(ctx, strings.TrimSpace(text))
	h.mu.Lock()
	h.EmbedLatency = append(h.EmbedLatency, time.Since(start))
	h.mu.Unlock()
	if err != nil {
		h.FallbackCount++
		return nil, memory.SemanticFallbackEmbeddingFailed, err
	}
	if len(vec) == 0 {
		h.FallbackCount++
		return nil, memory.SemanticFallbackEmbeddingFailed, nil
	}
	if h.Profile.Dimension > 0 && len(vec) != h.Profile.Dimension {
		h.FallbackCount++
		return nil, memory.SemanticFallbackDimensionMismatch, fmt.Errorf("dim %d want %d", len(vec), h.Profile.Dimension)
	}
	return vec, "", nil
}

func (h *LiveHybridMemoryStub) SearchSimilarCandidates(_ context.Context, query []float32, req memory.SearchRequest, limit int, minCosine float64) ([]memory.MemoryObject, map[uuid.UUID]float64, error) {
	if len(query) == 0 {
		h.FallbackCount++
		return nil, nil, nil
	}
	statuses := req.Statuses
	if len(statuses) == 0 {
		st := req.Status
		if st == "" {
			st = "active"
		}
		statuses = []string{st}
	}
	allowed := map[string]struct{}{}
	for _, st := range statuses {
		allowed[st] = struct{}{}
	}
	type cand struct {
		obj memory.MemoryObject
		sim float64
	}
	var cands []cand
	for _, o := range h.All {
		st := string(o.Status)
		if st == "" {
			st = "active"
		}
		if _, ok := allowed[st]; !ok {
			continue
		}
		if len(req.Tags) > 0 && !tagOverlap(o.Tags, req.Tags) {
			continue
		}
		lbl := h.IDToLabel[o.ID]
		if reason, stale := h.StaleLabels[lbl]; stale {
			h.StaleSkipCount++
			_ = reason
			continue
		}
		meta := h.LabelMeta[lbl]
		if reason := memory.CheckEmbeddingStaleness(o, meta, h.Profile); reason != memory.StalenessNone {
			h.StaleSkipCount++
			continue
		}
		lvec := h.LabelVector[lbl]
		if lvec == nil {
			h.MissingCount++
			continue
		}
		sim := cosineSimilarity(query, lvec)
		if sim+1e-9 < minCosine {
			continue
		}
		cands = append(cands, cand{obj: o, sim: sim})
	}
	sort.Slice(cands, func(i, j int) bool {
		if cands[i].sim != cands[j].sim {
			return cands[i].sim > cands[j].sim
		}
		return cands[i].obj.ID.String() < cands[j].obj.ID.String()
	})
	if limit <= 0 {
		limit = 50
	}
	if len(cands) > limit {
		cands = cands[:limit]
	}
	out := make([]memory.MemoryObject, len(cands))
	sims := map[uuid.UUID]float64{}
	for i, c := range cands {
		out[i] = c.obj
		sims[c.obj.ID] = c.sim
	}
	return out, sims, nil
}

// LiveEmbedStats summarizes live embedder benchmark telemetry.
type LiveEmbedStats struct {
	P50LatencyMs   float64 `json:"p50_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	FallbackCount  int     `json:"fallback_count"`
	StaleSkipCount int     `json:"stale_skip_count"`
	MissingCount   int     `json:"missing_embedding_count"`
}

func (h *LiveHybridMemoryStub) Stats() LiveEmbedStats {
	h.mu.Lock()
	defer h.mu.Unlock()
	st := LiveEmbedStats{
		FallbackCount:  h.FallbackCount,
		StaleSkipCount: h.StaleSkipCount,
		MissingCount:   h.MissingCount,
	}
	if len(h.EmbedLatency) == 0 {
		return st
	}
	lat := append([]time.Duration(nil), h.EmbedLatency...)
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	st.P50LatencyMs = float64(lat[len(lat)*50/100]) / float64(time.Millisecond)
	st.P95LatencyMs = float64(lat[len(lat)*95/100]) / float64(time.Millisecond)
	return st
}

// MarkLabelStale marks a fixture label stale (test helper).
func (h *LiveHybridMemoryStub) MarkLabelStale(lbl string, reason memory.StalenessReason) {
	h.StaleLabels[lbl] = reason
	delete(h.LabelVector, lbl)
}
