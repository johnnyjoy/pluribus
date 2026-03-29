package recall

import (
	"context"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"control-plane/internal/evidence"

	"github.com/google/uuid"
)

type fakeEvidenceLister struct {
	byMemory map[uuid.UUID][]evidence.Record
}

func (f *fakeEvidenceLister) ListEvidenceForMemory(ctx context.Context, memoryID uuid.UUID) ([]evidence.Record, error) {
	if f.byMemory == nil {
		return nil, nil
	}
	return f.byMemory[memoryID], nil
}

func TestApplyEvidenceInBundleDefaults(t *testing.T) {
	c := &EvidenceInBundleConfig{Enabled: true}
	ApplyEvidenceInBundleDefaults(c)
	if c.MaxPerMemory != 2 || c.MaxPerBundle != 12 || c.SummaryMaxChars != 256 {
		t.Fatalf("defaults: %+v", c)
	}
}

func TestHydrateSupportingEvidence_empty(t *testing.T) {
	ctx := context.Background()
	bundle := &RecallBundle{
		GoverningConstraints: []MemoryItem{{ID: uuid.New().String(), Kind: "constraint", Statement: "x", Authority: 5}},
	}
	cfg := &EvidenceInBundleConfig{Enabled: true, MaxPerMemory: 2, MaxPerBundle: 12, SummaryMaxChars: 256}
	l := &fakeEvidenceLister{}
	if err := HydrateSupportingEvidence(ctx, l, cfg, bundle); err != nil {
		t.Fatal(err)
	}
	if len(bundle.GoverningConstraints[0].SupportingEvidence) != 0 {
		t.Fatal("expected no evidence")
	}
}

func TestHydrateSupportingEvidence_sortedAndBounded(t *testing.T) {
	ctx := context.Background()
	memID := uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	e1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	e2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	e3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	t0 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)

	l := &fakeEvidenceLister{}
	l.byMemory = map[uuid.UUID][]evidence.Record{
		memID: {
			{ID: e1, Kind: evidence.KindObservation, Path: "/tmp/old.log", Digest: "ab", CreatedAt: t0},
			{ID: e2, Kind: evidence.KindTest, Path: "/tmp/good_test.go", Digest: "cd", CreatedAt: t1},
			{ID: e3, Kind: evidence.KindObservation, Path: "/tmp/new.log", Digest: "ef", CreatedAt: t1},
		},
	}

	bundle := &RecallBundle{
		GoverningConstraints: []MemoryItem{{ID: memID.String(), Kind: "constraint", Statement: "rule", Authority: 5}},
	}
	cfg := &EvidenceInBundleConfig{Enabled: true, MaxPerMemory: 2, MaxPerBundle: 12, SummaryMaxChars: 256}
	if err := HydrateSupportingEvidence(ctx, l, cfg, bundle); err != nil {
		t.Fatal(err)
	}
	se := bundle.GoverningConstraints[0].SupportingEvidence
	if len(se) != 2 {
		t.Fatalf("want 2 refs, got %d", len(se))
	}
	// KindTest scores higher than KindObservation — first ref should be test
	if se[0].Kind != evidence.KindTest {
		t.Errorf("first kind = %q, want test", se[0].Kind)
	}
}

func TestHydrateSupportingEvidence_bundleCap(t *testing.T) {
	ctx := context.Background()
	cfg := &EvidenceInBundleConfig{Enabled: true, MaxPerMemory: 2, MaxPerBundle: 2, SummaryMaxChars: 80}
	l := &fakeEvidenceLister{byMemory: map[uuid.UUID][]evidence.Record{}}
	m1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	m2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	m3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	t0 := time.Now()
	for _, id := range []uuid.UUID{m1, m2, m3} {
		l.byMemory[id] = []evidence.Record{
			{ID: uuid.New(), Kind: evidence.KindTest, Path: "/tmp/a", Digest: "aa", CreatedAt: t0},
			{ID: uuid.New(), Kind: evidence.KindTest, Path: "/tmp/b", Digest: "bb", CreatedAt: t0},
		}
	}
	bundle := &RecallBundle{
		GoverningConstraints: []MemoryItem{
			{ID: m1.String(), Kind: "constraint", Statement: "a", Authority: 5},
			{ID: m2.String(), Kind: "constraint", Statement: "b", Authority: 5},
			{ID: m3.String(), Kind: "constraint", Statement: "c", Authority: 5},
		},
	}
	if err := HydrateSupportingEvidence(ctx, l, cfg, bundle); err != nil {
		t.Fatal(err)
	}
	if !bundle.EvidenceBudgetApplied {
		t.Fatal("expected EvidenceBudgetApplied")
	}
	if len(bundle.GoverningConstraints[0].SupportingEvidence) != 2 {
		t.Fatalf("want 2 on first item, got %d", len(bundle.GoverningConstraints[0].SupportingEvidence))
	}
	if len(bundle.GoverningConstraints[1].SupportingEvidence) != 0 {
		t.Fatal("second item should be skipped when bundle cap exhausted")
	}
}

func TestRecordToEvidenceRef_truncates(t *testing.T) {
	r := evidence.Record{
		ID:        uuid.New(),
		Kind:      evidence.KindLog,
		Path:      "/var/logs/" + strings.Repeat("x", 200) + "/app.log",
		Digest:    "deadbeef",
		CreatedAt: time.Now(),
	}
	ref := recordToEvidenceRef(r, 40)
	if utf8.RuneCountInString(ref.Summary) > 41 {
		t.Fatalf("summary too long: %d runes", utf8.RuneCountInString(ref.Summary))
	}
}
