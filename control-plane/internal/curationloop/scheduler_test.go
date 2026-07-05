package curationloop

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

type fakeExpirer struct {
	calls atomic.Int32
	err   error
}

func (f *fakeExpirer) ExpireMemories(ctx context.Context, asOf time.Time) (int, error) {
	f.calls.Add(1)
	return 3, f.err
}

type fakePromoter struct {
	calls atomic.Int32
	err   error
}

func (f *fakePromoter) AutoPromoteBatchCount(ctx context.Context) (int, int, error) {
	f.calls.Add(1)
	return 1, 2, f.err
}

type fakeEmbedBackfill struct {
	calls atomic.Int32
	err   error
}

func (f *fakeEmbedBackfill) BackfillEmbeddingBatch(ctx context.Context, batchSize int) (embedded, remaining int, err error) {
	f.calls.Add(1)
	if f.err != nil {
		return 0, 0, f.err
	}
	return 2, 5, nil
}

type fakeRejectedPruner struct {
	calls atomic.Int32
	err   error
}

func (f *fakeRejectedPruner) PruneRejectedBatch(ctx context.Context, olderThanHours, limit int) (int, error) {
	f.calls.Add(1)
	if f.err != nil {
		return 0, f.err
	}
	return 4, nil
}

func TestRunOnce_callsPruneRejected(t *testing.T) {
	pr := &fakeRejectedPruner{}
	s := &Scheduler{Interval: time.Hour, Rejected: pr}
	s.RunOnce(context.Background())
	if pr.calls.Load() != 1 {
		t.Fatalf("expected prune rejected call, got %d", pr.calls.Load())
	}
}

func TestRunOnce_callsEmbedBackfill(t *testing.T) {
	emb := &fakeEmbedBackfill{}
	s := &Scheduler{Interval: time.Hour, Embeddings: emb, EmbedBackfillBatchSize: 10}
	s.RunOnce(context.Background())
	if emb.calls.Load() != 1 {
		t.Fatalf("expected embed backfill call, got %d", emb.calls.Load())
	}
}

func TestRunOnce_embedFailureDoesNotBlockExpire(t *testing.T) {
	exp := &fakeExpirer{}
	emb := &fakeEmbedBackfill{err: errors.New("embedder down")}
	s := &Scheduler{Interval: time.Hour, Memory: exp, Embeddings: emb}
	s.RunOnce(context.Background())
	if exp.calls.Load() != 1 {
		t.Fatalf("expire must still run after embed failure")
	}
}

func TestRunOnce_callsExpireAndPromote(t *testing.T) {
	exp := &fakeExpirer{}
	pro := &fakePromoter{}
	s := &Scheduler{Interval: time.Hour, Memory: exp, Promoter: pro}
	s.RunOnce(context.Background())
	if exp.calls.Load() != 1 || pro.calls.Load() != 1 {
		t.Fatalf("expected one call each, got expire=%d promote=%d", exp.calls.Load(), pro.calls.Load())
	}
}

func TestRunOnce_expireFailureDoesNotBlockPromote(t *testing.T) {
	exp := &fakeExpirer{err: errors.New("db down")}
	pro := &fakePromoter{}
	s := &Scheduler{Interval: time.Hour, Memory: exp, Promoter: pro}
	s.RunOnce(context.Background())
	if pro.calls.Load() != 1 {
		t.Fatalf("promote must still run after expire failure, got %d calls", pro.calls.Load())
	}
}

func TestRun_stopsOnContextCancel(t *testing.T) {
	exp := &fakeExpirer{}
	s := &Scheduler{Interval: 5 * time.Millisecond, Memory: exp}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.Run(ctx)
		close(done)
	}()
	// Let at least one tick pass, then cancel.
	time.Sleep(25 * time.Millisecond)
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Run did not stop after context cancel")
	}
	if exp.calls.Load() < 1 {
		t.Fatalf("expected at least one pass, got %d", exp.calls.Load())
	}
}

func TestRun_zeroIntervalNoops(t *testing.T) {
	s := &Scheduler{Interval: 0, Memory: &fakeExpirer{}}
	done := make(chan struct{})
	go func() {
		s.Run(context.Background())
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run with zero interval must return immediately")
	}
}
