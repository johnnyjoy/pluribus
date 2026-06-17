package benchmark

import (
	"context"
	"hash/fnv"
	"math"
	"sort"
	"strings"

	"control-plane/internal/memory"

	"github.com/google/uuid"
)

const benchmarkEmbedDim = 16

// HybridMemoryStub supports deterministic hybrid recall benchmarks without external embedders.
type HybridMemoryStub struct {
	All         []memory.MemoryObject
	LabelToID   map[string]uuid.UUID
	IDToLabel   map[uuid.UUID]string
	LabelVector map[string][]float32
}

// NewHybridMemoryStub builds a stub from a loaded corpus with deterministic label vectors.
func NewHybridMemoryStub(lc *LoadedCorpus) *HybridMemoryStub {
	h := &HybridMemoryStub{
		All:         lc.Objects,
		LabelToID:   lc.LabelToID,
		IDToLabel:   lc.IDToLabel,
		LabelVector: map[string][]float32{},
	}
	for id, fix := range lc.IDToFixture {
		text := strings.TrimSpace(fix.Statement)
		if text == "" {
			text = fix.Label
		}
		lbl := fix.Label
		h.LabelVector[lbl] = DeterministicEmbed(text, benchmarkEmbedDim)
		_ = id
	}
	return h
}

// DeterministicEmbed returns a unit-normalized vector from text (benchmark-only; not a production model).
func DeterministicEmbed(text string, dim int) []float32 {
	if dim <= 0 {
		dim = benchmarkEmbedDim
	}
	vec := make([]float32, dim)
	words := strings.Fields(strings.ToLower(strings.TrimSpace(text)))
	if len(words) == 0 {
		words = []string{text}
	}
	for _, w := range words {
		h := fnv.New64a()
		_, _ = h.Write([]byte(w))
		sum := h.Sum64()
		for i := 0; i < dim; i++ {
			bit := (sum >> (i % 64)) & 1
			if bit == 1 {
				vec[i] += 1
			} else {
				vec[i] -= 0.5
			}
		}
	}
	var norm float64
	for _, v := range vec {
		norm += float64(v) * float64(v)
	}
	if norm > 0 {
		n := float32(math.Sqrt(norm))
		for i := range vec {
			vec[i] /= n
		}
	}
	return vec
}

func cosineSimilarity(a, b []float32) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += float64(a[i]) * float64(b[i])
		na += float64(a[i]) * float64(a[i])
		nb += float64(b[i]) * float64(b[i])
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func (h *HybridMemoryStub) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	stub := &MemoryStub{All: h.All}
	return stub.Search(ctx, req)
}

func (h *HybridMemoryStub) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	stub := &MemoryStub{All: h.All}
	return stub.SearchMemories(ctx, req)
}

func (h *HybridMemoryStub) EmbedQueryText(_ context.Context, text string) ([]float32, string, error) {
	return DeterministicEmbed(text, benchmarkEmbedDim), "", nil
}

func (h *HybridMemoryStub) SearchSimilarCandidates(_ context.Context, query []float32, req memory.SearchRequest, limit int, minCosine float64) ([]memory.MemoryObject, map[uuid.UUID]float64, error) {
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
		lvec := h.LabelVector[lbl]
		if lvec == nil {
			lvec = DeterministicEmbed(lbl, benchmarkEmbedDim)
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
