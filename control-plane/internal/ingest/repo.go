package ingest

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"

	"github.com/google/uuid"
)

// Repo persists ingestion_records and canonical_fact_extractions.
type Repo struct {
	DB *sql.DB
}

// TrustWeightByTempContributorID returns trust_weight for this temporary ingest sender label, or 1.0 when no profile exists.
func (r *Repo) TrustWeightByTempContributorID(ctx context.Context, tempContributorID string) (float64, error) {
	var w sql.NullFloat64
	err := r.DB.QueryRowContext(ctx,
		`SELECT trust_weight FROM temp_contributor_profiles WHERE temp_contributor_id = $1`,
		tempContributorID,
	).Scan(&w)
	if errors.Is(err, sql.ErrNoRows) {
		return 1.0, nil
	}
	if err != nil {
		return 0, err
	}
	if !w.Valid {
		return 1.0, nil
	}
	return w.Float64, nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...interface{}) *sql.Row
}

type execer interface {
	ExecContext(context.Context, string, ...interface{}) (sql.Result, error)
}

type rowsQuerier interface {
	QueryContext(context.Context, string, ...interface{}) (*sql.Rows, error)
}

type canonicalTriple struct {
	PredicateNorm  string
	ObjectNorm     string
	NormalizedHash string
}

// selectCanonicalCandidates merges subject+predicate and subject-wide lookups (deduped by hash, ordered).
func selectCanonicalCandidates(ctx context.Context, q rowsQuerier, subjectNorm, predicateNorm string, limit int) ([]canonicalTriple, error) {
	if limit <= 0 {
		limit = 500
	}
	byHash := make(map[string]canonicalTriple)
	load := func(rows *sql.Rows) error {
		defer rows.Close()
		for rows.Next() {
			var t canonicalTriple
			if err := rows.Scan(&t.PredicateNorm, &t.ObjectNorm, &t.NormalizedHash); err != nil {
				return err
			}
			byHash[t.NormalizedHash] = t
		}
		return rows.Err()
	}
	r1, err := q.QueryContext(ctx,
		`SELECT predicate_norm, object_norm, normalized_hash FROM canonical_fact_extractions
		 WHERE subject_norm = $1 AND predicate_norm = $2
		 ORDER BY normalized_hash ASC LIMIT $3`,
		subjectNorm, predicateNorm, limit,
	)
	if err != nil {
		return nil, err
	}
	if err := load(r1); err != nil {
		return nil, err
	}
	r2, err := q.QueryContext(ctx,
		`SELECT predicate_norm, object_norm, normalized_hash FROM canonical_fact_extractions
		 WHERE subject_norm = $1
		 ORDER BY normalized_hash ASC LIMIT $2`,
		subjectNorm, limit,
	)
	if err != nil {
		return nil, err
	}
	if err := load(r2); err != nil {
		return nil, err
	}
	out := make([]canonicalTriple, 0, len(byHash))
	for _, t := range byHash {
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].NormalizedHash < out[j].NormalizedHash
	})
	return out, nil
}

// insertCanonicalFactLineageBatch persists GMCL lineage rows (no-op when empty).
func insertCanonicalFactLineageBatch(ctx context.Context, e execer, ingestionID uuid.UUID, events []LineageEvent) error {
	if len(events) == 0 {
		return nil
	}
	for _, ev := range events {
		meta := ev.Meta
		if meta == nil {
			meta = map[string]interface{}{}
		}
		mb, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		var parent interface{}
		if ev.ParentHash != "" {
			parent = ev.ParentHash
		}
		_, err = e.ExecContext(ctx,
			`INSERT INTO canonical_fact_lineage (ingestion_id, fact_hash, parent_hash, root_hash, merge_type, source, meta)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			ingestionID, ev.FactHash, parent, ev.RootHash, ev.MergeType, ev.Source, mb,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

func insertIngestion(ctx context.Context, q queryRower, tempContributorID, status string, rejectedReason *string, payload json.RawMessage, contextHash *string) (uuid.UUID, error) {
	var id uuid.UUID
	var rr interface{}
	if rejectedReason != nil {
		rr = *rejectedReason
	}
	var ch interface{}
	if contextHash != nil && *contextHash != "" {
		ch = *contextHash
	}
	err := q.QueryRowContext(ctx,
		`INSERT INTO ingestion_records (temp_contributor_id, status, rejected_reason, payload_raw, context_window_hash)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		tempContributorID, status, rr, []byte(payload), ch,
	).Scan(&id)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

// Insert creates an ingestion row and returns its id.
func (r *Repo) Insert(ctx context.Context, tempContributorID, status string, rejectedReason *string, payload json.RawMessage, contextHash *string) (uuid.UUID, error) {
	return insertIngestion(ctx, r.DB, tempContributorID, status, rejectedReason, payload, contextHash)
}

// IngestionRecordStatus is a minimal projection for commit/promotion checks (M7).
type IngestionRecordStatus struct {
	Status string
}

// GetIngestionStatus returns status for an ingestion row, or nil if not found.
func (r *Repo) GetIngestionStatus(ctx context.Context, id uuid.UUID) (*IngestionRecordStatus, error) {
	var st string
	err := r.DB.QueryRowContext(ctx,
		`SELECT status FROM ingestion_records WHERE id = $1`,
		id,
	).Scan(&st)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &IngestionRecordStatus{Status: st}, nil
}

// ListCanonicalRowsByIngestionID loads persisted canonical rows for operator promotion (M7).
func (r *Repo) ListCanonicalRowsByIngestionID(ctx context.Context, ingestionID uuid.UUID) ([]CanonicalFactRow, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT ingestion_id, subject_norm, predicate_norm, object_norm, confidence, provenance, normalized_hash, source_index, priority_score
		 FROM canonical_fact_extractions WHERE ingestion_id = $1 ORDER BY source_index ASC`,
		ingestionID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []CanonicalFactRow
	for rows.Next() {
		var row CanonicalFactRow
		var prov []byte
		if err := rows.Scan(
			&row.IngestionID, &row.SubjectNorm, &row.PredicateNorm, &row.ObjectNorm,
			&row.Confidence, &prov, &row.NormalizedHash, &row.SourceIndex, &row.PriorityScore,
		); err != nil {
			return nil, err
		}
		row.Provenance = prov
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// SelectCanonicalConfidenceStats returns MAX(confidence) and row count for a normalized hash.
// Used inside a transaction so prior inserts in the same ingest are visible (M3 reinforce).
func SelectCanonicalConfidenceStats(ctx context.Context, q queryRower, normalizedHash string) (maxConf float64, count int64, err error) {
	err = q.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(confidence), 0), COUNT(*) FROM canonical_fact_extractions WHERE normalized_hash = $1`,
		normalizedHash,
	).Scan(&maxConf, &count)
	return maxConf, count, err
}

// SelectMaxCreatedAtForHash returns MAX(created_at) for prior extractions of this hash (M6 recency).
func SelectMaxCreatedAtForHash(ctx context.Context, q queryRower, normalizedHash string) (sql.NullTime, error) {
	var t sql.NullTime
	err := q.QueryRowContext(ctx,
		`SELECT MAX(created_at) FROM canonical_fact_extractions WHERE normalized_hash = $1`,
		normalizedHash,
	).Scan(&t)
	if err != nil {
		return sql.NullTime{}, err
	}
	return t, nil
}

// insertCanonicalFact persists one extraction row (M2/M3/M6).
func insertCanonicalFact(ctx context.Context, e execer, row CanonicalFactRow) error {
	_, err := e.ExecContext(ctx,
		`INSERT INTO canonical_fact_extractions (ingestion_id, subject_norm, predicate_norm, object_norm, confidence, provenance, normalized_hash, source_index, priority_score)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		row.IngestionID, row.SubjectNorm, row.PredicateNorm, row.ObjectNorm,
		row.Confidence, []byte(row.Provenance), row.NormalizedHash, row.SourceIndex, row.PriorityScore,
	)
	return err
}
