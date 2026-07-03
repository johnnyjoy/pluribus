// Package curationloop runs the background auto-curation scheduler (Phase 3):
// a config-gated in-process loop that periodically expires stale/probationary
// memories and auto-promotes eligible candidates, so the pool is curated
// continuously instead of only when an operator remembers to call the batch
// endpoints by hand.
package curationloop

import (
	"context"
	"log/slog"
	"time"
)

// MemoryExpirer archives TTL-expired rows and time-decayed probationary rows.
// Satisfied by *memory.Service.
type MemoryExpirer interface {
	ExpireMemories(ctx context.Context, asOf time.Time) (int, error)
}

// AutoPromoter materializes pending candidates that pass promotion thresholds.
// Satisfied by *curation.Service (only wired when promotion.auto_promote is on).
type AutoPromoter interface {
	AutoPromoteBatchCount(ctx context.Context) (promoted, skipped int, err error)
}

// ChorePassRunner opens agent-facing curation chores (contradiction review,
// quarantine review, embedding near-duplicate pairs). Satisfied by *chores.Service.
type ChorePassRunner interface {
	RunChorePass(ctx context.Context) (opened int, err error)
}

// EmbeddingBackfiller embeds a batch of rows missing or stale vectors.
// Satisfied by *memory.Service via router adapter.
type EmbeddingBackfiller interface {
	BackfillEmbeddingBatch(ctx context.Context, batchSize int) (embedded, remaining int, err error)
}

// Scheduler runs the curation loop. Zero-value fields are skipped.
type Scheduler struct {
	// Interval between runs (required; caller applies the config default).
	Interval time.Duration
	// InitialDelay before the first run, so startup traffic settles first.
	InitialDelay time.Duration
	Memory       MemoryExpirer
	Promoter     AutoPromoter
	Chores       ChorePassRunner
	Embeddings   EmbeddingBackfiller
	// EmbedBackfillBatchSize caps rows embedded per pass (default 50).
	EmbedBackfillBatchSize int
}

// Run blocks until ctx is done, executing RunOnce every Interval.
// Intended to be started as a goroutine at server boot.
func (s *Scheduler) Run(ctx context.Context) {
	if s == nil || s.Interval <= 0 {
		return
	}
	if s.InitialDelay > 0 {
		select {
		case <-ctx.Done():
			return
		case <-time.After(s.InitialDelay):
		}
	}
	slog.Info("[CURATION LOOP] started", "interval", s.Interval.String())
	ticker := time.NewTicker(s.Interval)
	defer ticker.Stop()
	s.RunOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			slog.Info("[CURATION LOOP] stopped")
			return
		case <-ticker.C:
			s.RunOnce(ctx)
		}
	}
}

// RunOnce executes one curation pass: expire, then auto-promote.
// Each step is independent; a failing step never blocks the others.
func (s *Scheduler) RunOnce(ctx context.Context) {
	if s == nil {
		return
	}
	start := time.Now()
	archived := 0
	if s.Memory != nil {
		n, err := s.Memory.ExpireMemories(ctx, time.Now())
		if err != nil {
			slog.Warn("[CURATION LOOP] expire failed", "error", err.Error())
		} else {
			archived = n
		}
	}
	promoted, skipped := 0, 0
	if s.Promoter != nil {
		p, sk, err := s.Promoter.AutoPromoteBatchCount(ctx)
		if err != nil {
			slog.Warn("[CURATION LOOP] auto-promote failed", "error", err.Error())
		} else {
			promoted, skipped = p, sk
		}
	}
	choresOpened := 0
	if s.Chores != nil {
		n, err := s.Chores.RunChorePass(ctx)
		if err != nil {
			slog.Warn("[CURATION LOOP] chore pass failed", "error", err.Error())
		} else {
			choresOpened = n
		}
	}
	embedBatch := s.EmbedBackfillBatchSize
	if embedBatch <= 0 {
		embedBatch = 50
	}
	embedded, embedRemaining := 0, 0
	if s.Embeddings != nil {
		n, rem, err := s.Embeddings.BackfillEmbeddingBatch(ctx, embedBatch)
		if err != nil {
			slog.Warn("[CURATION LOOP] embed backfill failed", "error", err.Error())
		} else {
			embedded, embedRemaining = n, rem
		}
	}
	slog.Info("[CURATION LOOP] pass complete",
		"archived", archived, "promoted", promoted, "promote_skipped", skipped,
		"chores_opened", choresOpened,
		"embeddings_backfilled", embedded, "embeddings_remaining", embedRemaining,
		"elapsed_ms", time.Since(start).Milliseconds())
}
