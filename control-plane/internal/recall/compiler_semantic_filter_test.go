package recall

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type semanticFilterTestSearcher struct {
	semanticObjs []memory.MemoryObject
	semanticSims map[uuid.UUID]float64
}

func (s *semanticFilterTestSearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	return nil, nil
}

func (s *semanticFilterTestSearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	return nil, nil
}

func (s *semanticFilterTestSearcher) EmbedQueryText(ctx context.Context, text string) ([]float32, string, error) {
	return []float32{1, 0, 0}, "", nil
}

func (s *semanticFilterTestSearcher) SearchSimilarCandidates(ctx context.Context, query []float32, req memory.SearchRequest, limit int, minCosine float64) ([]memory.MemoryObject, map[uuid.UUID]float64, error) {
	var out []memory.MemoryObject
	for _, o := range s.semanticObjs {
		st := string(o.Status)
		if st == "" {
			st = "active"
		}
		if len(req.Statuses) > 0 {
			ok := false
			for _, want := range req.Statuses {
				if st == want {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		} else if req.Status != "" && st != req.Status {
			continue
		}
		out = append(out, o)
	}
	return out, s.semanticSims, nil
}

func memWithStatus(id uuid.UUID, status api.Status, statement string) memory.MemoryObject {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return memory.MemoryObject{
		ID:        id,
		Kind:      api.MemoryKindPattern,
		Statement: statement,
		Status:    status,
		Authority: 5,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func semanticCompiler(searcher *semanticFilterTestSearcher) *Compiler {
	w := DefaultRankingWeights()
	return &Compiler{
		Memory:  searcher,
		Ranking: &w,
		Semantic: &SemanticRecallConfig{
			Enabled:             true,
			MaxCandidates:       50,
			MinCosineSimilarity: 0.1,
		},
	}
}

func TestSemanticCurrentModeExcludesArchived(t *testing.T) {
	archivedID := uuid.MustParse("a0000001-0000-0000-0000-000000000001")
	searcher := &semanticFilterTestSearcher{
		semanticObjs: []memory.MemoryObject{memWithStatus(archivedID, api.StatusArchived, "archived guidance")},
		semanticSims: map[uuid.UUID]float64{archivedID: 0.99},
	}
	bundle, err := semanticCompiler(searcher).Compile(context.Background(), CompileRequest{
		RetrievalQuery: "guidance",
		RecallMode:     "current",
		MaxPerKind:     10,
		MaxTotal:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == archivedID.String() {
			t.Fatal("archived memory leaked into current mode")
		}
	}
}

func TestSemanticCurrentModeExcludesSuperseded(t *testing.T) {
	supID := uuid.MustParse("a0000002-0000-0000-0000-000000000002")
	searcher := &semanticFilterTestSearcher{
		semanticObjs: []memory.MemoryObject{memWithStatus(supID, api.StatusSuperseded, "superseded guidance")},
		semanticSims: map[uuid.UUID]float64{supID: 0.99},
	}
	bundle, err := semanticCompiler(searcher).Compile(context.Background(), CompileRequest{
		RetrievalQuery: "guidance",
		RecallMode:     "current",
		MaxPerKind:     10,
		MaxTotal:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == supID.String() {
			t.Fatal("superseded memory leaked into current mode")
		}
	}
}

func TestSemanticHistoricalModeIncludesArchived(t *testing.T) {
	archivedID := uuid.MustParse("a0000003-0000-0000-0000-000000000003")
	searcher := &semanticFilterTestSearcher{
		semanticObjs: []memory.MemoryObject{memWithStatus(archivedID, api.StatusArchived, "historical archived guidance")},
		semanticSims: map[uuid.UUID]float64{archivedID: 0.99},
	}
	bundle, err := semanticCompiler(searcher).Compile(context.Background(), CompileRequest{
		RetrievalQuery: "historical archived guidance",
		RecallMode:     "historical",
		MaxPerKind:     10,
		MaxTotal:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == archivedID.String() {
			found = true
		}
	}
	if !found {
		t.Fatal("expected archived memory in historical mode")
	}
}

func TestSemanticHistoricalModeIncludesSuperseded(t *testing.T) {
	supID := uuid.MustParse("a0000004-0000-0000-0000-000000000004")
	searcher := &semanticFilterTestSearcher{
		semanticObjs: []memory.MemoryObject{memWithStatus(supID, api.StatusSuperseded, "historical superseded guidance")},
		semanticSims: map[uuid.UUID]float64{supID: 0.99},
	}
	bundle, err := semanticCompiler(searcher).Compile(context.Background(), CompileRequest{
		RetrievalQuery: "historical superseded guidance",
		RecallMode:     "historical",
		MaxPerKind:     10,
		MaxTotal:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == supID.String() {
			found = true
		}
	}
	if !found {
		t.Fatal("expected superseded memory in historical mode")
	}
}

func TestSemanticIncludeStatusHonored(t *testing.T) {
	archivedID := uuid.MustParse("a0000005-0000-0000-0000-000000000005")
	activeID := uuid.MustParse("a0000006-0000-0000-0000-000000000006")
	searcher := &semanticFilterTestSearcher{
		semanticObjs: []memory.MemoryObject{
			memWithStatus(archivedID, api.StatusArchived, "only archived"),
			memWithStatus(activeID, api.StatusActive, "active noise"),
		},
		semanticSims: map[uuid.UUID]float64{archivedID: 0.99, activeID: 0.98},
	}
	bundle, err := semanticCompiler(searcher).Compile(context.Background(), CompileRequest{
		RetrievalQuery: "only archived",
		IncludeStatus:  []string{"archived"},
		MaxPerKind:     10,
		MaxTotal:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == activeID.String() {
			t.Fatal("active memory returned when include_status is archived only")
		}
	}
}

func TestSemanticDateBoundsFilterVectorCandidates(t *testing.T) {
	insideID := uuid.MustParse("a0000007-0000-0000-0000-000000000007")
	outsideID := uuid.MustParse("a0000008-0000-0000-0000-000000000008")
	inTime := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
	outTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	inside := memWithStatus(insideID, api.StatusArchived, "june event")
	inside.OccurredAt = &inTime
	outside := memWithStatus(outsideID, api.StatusArchived, "later event")
	outside.OccurredAt = &outTime
	searcher := &semanticFilterTestSearcher{
		semanticObjs: []memory.MemoryObject{inside, outside},
		semanticSims: map[uuid.UUID]float64{insideID: 0.8, outsideID: 0.99},
	}
	bundle, err := semanticCompiler(searcher).Compile(context.Background(), CompileRequest{
		RetrievalQuery: "event",
		RecallMode:     "historical",
		OccurredAfter:  "2023-06-15T00:00:00Z",
		OccurredBefore: "2023-06-16T00:00:00Z",
		MaxPerKind:     10,
		MaxTotal:       20,
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range bundle.ApplicablePatterns {
		if p.ID == outsideID.String() {
			t.Fatal("outside date-bound memory returned")
		}
	}
}

func TestSemanticPostMergeSafetyFilter(t *testing.T) {
	archivedID := uuid.MustParse("a0000009-0000-0000-0000-000000000009")
	obj := memWithStatus(archivedID, api.StatusArchived, "post merge filter")
	filtered := applyCandidateSafetyFilter(
		[]memory.MemoryObject{obj},
		RecallModeCurrent,
		[]api.Status{api.StatusActive},
		"",
		nil,
		nil,
	)
	if len(filtered) != 0 {
		t.Fatal("post-merge safety filter should remove archived in current mode")
	}
}
