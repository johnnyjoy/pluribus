package chores

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
)

// syncLimit bounds how many source rows each sync pass considers.
const syncLimit = 50

// contradictionEvidence is stored on contradiction chores so the resolver can
// update the originating contradiction_records row.
type contradictionEvidence struct {
	ContradictionRecordID uuid.UUID `json:"contradiction_record_id"`
}

// quarantineEvidence carries the quarantine reason for review context.
type quarantineEvidence struct {
	Reason string `json:"reason,omitempty"`
}

// duplicateEvidence carries the embedding similarity that flagged the pair.
type duplicateEvidence struct {
	CosineSimilarity float64 `json:"cosine_similarity"`
}

// SyncReviewChores opens chores for unresolved contradictions and quarantined
// memories. Idempotent: a (type, subject, related) triple is chore'd at most
// once ever. Returns the number of chores newly opened.
func (s *Service) SyncReviewChores(ctx context.Context) (int, error) {
	opened := 0

	rows, err := s.Repo.DB.QueryContext(ctx,
		`SELECT id, memory_id, conflict_with_id FROM contradiction_records
		 WHERE resolution_state = 'unresolved'
		 ORDER BY created_at ASC LIMIT $1`, syncLimit)
	if err != nil {
		return opened, err
	}
	type contraRow struct {
		recID, memID, conflictID uuid.UUID
	}
	var contras []contraRow
	for rows.Next() {
		var c contraRow
		if err := rows.Scan(&c.recID, &c.memID, &c.conflictID); err != nil {
			rows.Close()
			return opened, err
		}
		contras = append(contras, c)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return opened, err
	}
	for _, c := range contras {
		related := c.conflictID
		created, err := s.Repo.EnsureChore(ctx, TypeContradiction, c.memID, &related,
			contradictionEvidence{ContradictionRecordID: c.recID})
		if err != nil {
			slog.Warn("[CHORES] contradiction sync failed", "error", err.Error())
			continue
		}
		if created {
			opened++
		}
	}

	qrows, err := s.Repo.DB.QueryContext(ctx,
		`SELECT id, COALESCE(status_reason, '') FROM memories
		 WHERE status = 'quarantined'
		 ORDER BY updated_at ASC LIMIT $1`, syncLimit)
	if err != nil {
		return opened, err
	}
	type quarRow struct {
		id     uuid.UUID
		reason string
	}
	var quars []quarRow
	for qrows.Next() {
		var q quarRow
		if err := qrows.Scan(&q.id, &q.reason); err != nil {
			qrows.Close()
			return opened, err
		}
		quars = append(quars, q)
	}
	qrows.Close()
	if err := qrows.Err(); err != nil {
		return opened, err
	}
	for _, q := range quars {
		created, err := s.Repo.EnsureChore(ctx, TypeQuarantineReview, q.id, nil,
			quarantineEvidence{Reason: q.reason})
		if err != nil {
			slog.Warn("[CHORES] quarantine sync failed", "error", err.Error())
			continue
		}
		if created {
			opened++
		}
	}
	return opened, nil
}

// ScanNearDuplicates opens duplicate_pair chores for active memory pairs whose
// embedding cosine similarity is at or above minSimilarity, considering rows
// created in the last windowDays. Pairs are canonicalized (subject < related
// by id) so each pair is chore'd at most once ever; the pair already being
// related (supersedes/contradicts/...) also suppresses the chore.
func (s *Service) ScanNearDuplicates(ctx context.Context, minSimilarity float64, windowDays, limit int) (int, error) {
	if minSimilarity <= 0 || minSimilarity >= 1 {
		minSimilarity = 0.92
	}
	if windowDays <= 0 {
		windowDays = 14
	}
	if limit <= 0 {
		limit = 20
	}
	maxDistance := 1 - minSimilarity
	rows, err := s.Repo.DB.QueryContext(ctx,
		`SELECT LEAST(a.id, b.id), GREATEST(a.id, b.id), 1 - (a.embedding <=> b.embedding)
		 FROM memories a
		 JOIN memories b ON a.id < b.id
		 WHERE a.status = 'active' AND b.status = 'active'
		   AND a.embedding IS NOT NULL AND b.embedding IS NOT NULL
		   AND a.created_at >= now() - ($1 || ' days')::interval
		   AND (a.embedding <=> b.embedding) <= $2
		   AND NOT EXISTS (
		     SELECT 1 FROM memory_relationships r
		     WHERE (r.from_memory_id = a.id AND r.to_memory_id = b.id)
		        OR (r.from_memory_id = b.id AND r.to_memory_id = a.id)
		   )
		 ORDER BY (a.embedding <=> b.embedding) ASC
		 LIMIT $3`,
		windowDays, maxDistance, limit)
	if err != nil {
		return 0, err
	}
	type pair struct {
		subject, related uuid.UUID
		similarity       float64
	}
	var pairs []pair
	for rows.Next() {
		var p pair
		if err := rows.Scan(&p.subject, &p.related, &p.similarity); err != nil {
			rows.Close()
			return 0, err
		}
		pairs = append(pairs, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}
	opened := 0
	for _, p := range pairs {
		related := p.related
		created, err := s.Repo.EnsureChore(ctx, TypeDuplicatePair, p.subject, &related,
			duplicateEvidence{CosineSimilarity: p.similarity})
		if err != nil {
			slog.Warn("[CHORES] near-dup chore create failed", "error", err.Error())
			continue
		}
		if created {
			opened++
		}
	}
	return opened, nil
}
