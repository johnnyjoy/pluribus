package recall

import (
	"context"
	"testing"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

type dateFilterFakeSearcher struct {
	all []memory.MemoryObject
}

func (s *dateFilterFakeSearcher) Search(ctx context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	st := req.Status
	if st == "" {
		st = "active"
	}
	var out []memory.MemoryObject
	for _, o := range s.all {
		os := string(o.Status)
		if os == "" {
			os = "active"
		}
		if len(req.Tags) > 0 && !dateFilterTagOverlap(o.Tags, req.Tags) {
			continue
		}
		if os == st {
			out = append(out, o)
		}
	}
	return out, nil
}

func (s *dateFilterFakeSearcher) SearchMemories(ctx context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	return s.Search(ctx, memory.SearchRequest{Tags: req.Tags, Status: req.Status, Max: req.Max})
}

func dateFilterTagOverlap(memTags, reqTags []string) bool {
	set := map[string]struct{}{}
	for _, t := range memTags {
		set[t] = struct{}{}
	}
	for _, t := range reqTags {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}

func bundleContainsMemoryID(bundle *RecallBundle, id uuid.UUID) bool {
	if bundle == nil {
		return false
	}
	idStr := id.String()
	for _, items := range [][]MemoryItem{
		bundle.GoverningConstraints, bundle.Decisions, bundle.KnownFailures,
		bundle.ApplicablePatterns, bundle.Constraints, bundle.Continuity, bundle.Experience,
	} {
		for _, it := range items {
			if it.ID == idStr {
				return true
			}
		}
	}
	return false
}

func TestParseCompileDateBounds_valid(t *testing.T) {
	after, before, err := ParseCompileDateBounds(CompileRequest{
		OccurredAfter:  "2023-06-15T00:00:00-04:00",
		OccurredBefore: "2023-06-16T00:00:00-04:00",
	})
	if err != nil {
		t.Fatalf("ParseCompileDateBounds: %v", err)
	}
	if after == nil || before == nil {
		t.Fatal("expected both bounds")
	}
}

func TestParseCompileDateBounds_invalidAfter(t *testing.T) {
	_, _, err := ParseCompileDateBounds(CompileRequest{OccurredAfter: "not-a-date"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCompileDateBounds_invalidBefore(t *testing.T) {
	_, _, err := ParseCompileDateBounds(CompileRequest{OccurredBefore: "yesterday"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseCompileDateBounds_invertedRange(t *testing.T) {
	_, _, err := ParseCompileDateBounds(CompileRequest{
		OccurredAfter:  "2024-01-02T00:00:00Z",
		OccurredBefore: "2024-01-01T00:00:00Z",
	})
	if err != ErrInvalidDateRange {
		t.Fatalf("got %v want ErrInvalidDateRange", err)
	}
}

func TestFilterByDateBounds_occurredAt(t *testing.T) {
	day := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)
	after := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	before := time.Date(2023, 6, 16, 0, 0, 0, 0, time.UTC)
	in := []memory.MemoryObject{{
		ID:         uuid.New(),
		Statement:  "inside range",
		OccurredAt: &day,
		CreatedAt:  time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
	}}
	out := filterByDateBounds(in, &after, &before)
	if len(out) != 1 {
		t.Fatalf("len=%d want 1", len(out))
	}
}

func TestFilterByDateBounds_createdAtFallback(t *testing.T) {
	created := time.Date(2023, 6, 15, 8, 0, 0, 0, time.UTC)
	after := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	before := time.Date(2023, 6, 16, 0, 0, 0, 0, time.UTC)
	in := []memory.MemoryObject{{
		ID:        uuid.New(),
		Statement: "fallback created_at",
		CreatedAt: created,
	}}
	out := filterByDateBounds(in, &after, &before)
	if len(out) != 1 {
		t.Fatalf("len=%d want 1 (created_at fallback)", len(out))
	}
}

func TestFilterByDateBounds_excludesOutside(t *testing.T) {
	created := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	after := time.Date(2023, 6, 15, 0, 0, 0, 0, time.UTC)
	before := time.Date(2023, 6, 16, 0, 0, 0, 0, time.UTC)
	in := []memory.MemoryObject{{ID: uuid.New(), Statement: "outside", CreatedAt: created}}
	out := filterByDateBounds(in, &after, &before)
	if len(out) != 0 {
		t.Fatalf("len=%d want 0", len(out))
	}
}

func TestHistoricalRecallFiltersByOccurredAt_integration(t *testing.T) {
	occurred := time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)
	created := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	memID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	searcher := &dateFilterFakeSearcher{all: []memory.MemoryObject{{
		ID:         memID,
		Kind:       api.MemoryKindState,
		Statement:  "Phase 9B date-bound historical recall proof statement",
		Status:     api.StatusArchived,
		OccurredAt: &occurred,
		CreatedAt:  created,
		Tags:       []string{"phase9b-date"},
	}}}
	c := &Compiler{Memory: searcher}
	after := "2023-06-15T00:00:00Z"
	before := "2023-06-16T00:00:00Z"
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "what were we doing",
		RecallMode:     "historical",
		Tags:           []string{"phase9b-date"},
		OccurredAfter:  after,
		OccurredBefore: before,
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	found := bundleContainsMemoryID(bundle, memID)
	if !found {
		t.Fatal("expected archived memory inside date window")
	}
}

func TestHistoricalRecallExcludesOutsideDateRange(t *testing.T) {
	occurred := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	memID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	searcher := &dateFilterFakeSearcher{all: []memory.MemoryObject{{
		ID:         memID,
		Statement:  "old archived memory outside window",
		Status:     api.StatusArchived,
		OccurredAt: &occurred,
		CreatedAt:  occurred,
	}}}
	c := &Compiler{Memory: searcher}
	bundle, err := c.Compile(context.Background(), CompileRequest{
		RetrievalQuery: "history",
		RecallMode:     "historical",
		OccurredAfter:  "2023-06-15T00:00:00Z",
		OccurredBefore: "2023-06-16T00:00:00Z",
		MaxPerKind:     5,
	})
	if err != nil {
		t.Fatalf("Compile: %v", err)
	}
	if bundleContainsMemoryID(bundle, memID) {
		t.Fatal("memory outside date range must not appear")
	}
}
