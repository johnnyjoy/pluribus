package ingest

import (
	"database/sql"
	"math"
	"testing"
	"time"
)

func TestComputePriorityScore_firstOccurrence(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 17, 12, 0, 0, 0, time.UTC)
	sc := ComputePriorityScore(0.8, 0, sql.NullTime{}, now, 1.0, DefaultPriorityWeights())
	w := DefaultPriorityWeights()
	want := w.WSignal*0.8 + w.WFreq*0 + w.WRecency*1.0 + w.WAgreement*1.0
	if math.Abs(sc-want) > 1e-9 {
		t.Fatalf("got %v want %v", sc, want)
	}
}

func TestComputePriorityScore_withLastSeen(t *testing.T) {
	t.Parallel()
	last := time.Date(2025, 3, 10, 12, 0, 0, 0, time.UTC)
	now := time.Date(2025, 3, 17, 12, 0, 0, 0, time.UTC)
	nt := sql.NullTime{Time: last, Valid: true}
	h := now.Sub(last).Hours()
	rec := math.Exp(-h / 168.0)
	w := DefaultPriorityWeights()
	sc := ComputePriorityScore(0.5, 3, nt, now, 0.5, w)
	freq := math.Min(1.0, 3.0/10.0)
	want := w.WSignal*0.5 + w.WFreq*freq + w.WRecency*rec + w.WAgreement*0.5
	if math.Abs(sc-want) > 1e-9 {
		t.Fatalf("got %v want %v", sc, want)
	}
}
