package recall

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// semanticHybridTestSearcher drives compile tests: empty lexical bridge, semantic path supplies candidates.
type semanticHybridTestSearcher struct {
	baseSearch       []memory.MemoryObject
	semanticObjs     []memory.MemoryObject
	semanticSims     map[uuid.UUID]float64
	embedErr         error
	searchSimilarErr error
}

func (s *semanticHybridTestSearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	return s.baseSearch, nil
}

func (s *semanticHybridTestSearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	return nil, nil
}

func (s *semanticHybridTestSearcher) EmbedQueryText(ctx context.Context, text string) ([]float32, string, error) {
	if s.embedErr != nil {
		return nil, "", s.embedErr
	}
	return []float32{1, 0, 0}, "", nil
}

func (s *semanticHybridTestSearcher) SearchSimilarCandidates(ctx context.Context, query []float32, req memory.SearchRequest, limit int, minCosine float64) ([]memory.MemoryObject, map[uuid.UUID]float64, error) {
	if s.searchSimilarErr != nil {
		return nil, nil, s.searchSimilarErr
	}
	return s.semanticObjs, s.semanticSims, nil
}

func patternMem(id uuid.UUID, authority int, statement, canonical string) memory.MemoryObject {
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	return memory.MemoryObject{
		ID:                 id,
		Kind:               api.MemoryKindPattern,
		Authority:          authority,
		Applicability:      api.ApplicabilityGoverning,
		Statement:          statement,
		StatementCanonical: canonical,
		Status:             api.StatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
}

func TestCompile_semanticParaphraseRetrievesMemory(t *testing.T) {
	ctx := context.Background()
	idemID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	idem := patternMem(idemID, 5,
		"use idempotency keys for retry-safe webhook ingestion",
		"use idempotency keys for retry-safe webhook ingestion",
	)
	searcher := &semanticHybridTestSearcher{
		baseSearch:   nil,
		semanticObjs: []memory.MemoryObject{idem},
		semanticSims: map[uuid.UUID]float64{idemID: 0.92},
	}
	w := DefaultRankingWeights()
	c := &Compiler{
		Memory:  searcher,
		Ranking: &w,
		Semantic: &SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       50,
			MinCosineSimilarity: 0.35,
		},
	}
	bundle, err := c.Compile(ctx, CompileRequest{
		RetrievalQuery: "avoid duplicate webhook charges",
		MaxPerKind:     10,
		MaxTotal:       50,
		Mode:           "continuity",
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	found := false
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == idemID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected idempotency pattern in bundle, got patterns=%v", bundle.ApplicablePatterns)
	}
}

func TestCompile_semanticBeatsLowLexicalOverlap(t *testing.T) {
	ctx := context.Background()
	memID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	// Deliberately unrelated wording vs query — lexical Jaccard should be near zero.
	mem := patternMem(memID, 5,
		"prefer exponential backoff with jitter between queue consumers",
		"prefer exponential backoff with jitter between queue consumers",
	)
	searcher := &semanticHybridTestSearcher{
		baseSearch:   nil,
		semanticObjs: []memory.MemoryObject{mem},
		semanticSims: map[uuid.UUID]float64{memID: 0.9},
	}
	w := DefaultRankingWeights()
	c := &Compiler{
		Memory:  searcher,
		Ranking: &w,
		Semantic: &SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       50,
			MinCosineSimilarity: 0.35,
		},
	}
	bundle, err := c.Compile(ctx, CompileRequest{
		RetrievalQuery: "kubernetes pod crash loop backoff policy",
		MaxPerKind:     10,
		MaxTotal:       50,
		Mode:           "continuity",
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	found := false
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == memID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected semantically retrieved pattern despite low lexical overlap")
	}
}

func TestCompile_authoritySortDominatesSemanticStrength(t *testing.T) {
	ctx := context.Background()
	weakSemID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	strongSemID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	highAuth := patternMem(weakSemID, 10,
		"alpha",
		"alpha",
	)
	lowAuth := patternMem(strongSemID, 1,
		"beta",
		"beta",
	)
	searcher := &semanticHybridTestSearcher{
		baseSearch:   []memory.MemoryObject{highAuth, lowAuth},
		semanticObjs: []memory.MemoryObject{highAuth, lowAuth},
		semanticSims: map[uuid.UUID]float64{
			weakSemID:   0.25,
			strongSemID: 0.99,
		},
	}
	w := DefaultRankingWeights()
	c := &Compiler{
		Memory:  searcher,
		Ranking: &w,
		Semantic: &SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       50,
			MinCosineSimilarity: 0.35,
		},
	}
	bundle, err := c.Compile(ctx, CompileRequest{
		RetrievalQuery: "gamma",
		MaxPerKind:     10,
		MaxTotal:       50,
		Mode:           "continuity",
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if len(bundle.ApplicablePatterns) < 2 {
		t.Fatalf("want 2 patterns, got %d", len(bundle.ApplicablePatterns))
	}
	// sortScoredMemoriesStable: higher authority first regardless of softer score.
	if bundle.ApplicablePatterns[0].ID != weakSemID.String() {
		t.Fatalf("expected high-authority pattern first, got first=%q second=%q",
			bundle.ApplicablePatterns[0].ID, bundle.ApplicablePatterns[1].ID)
	}
}
