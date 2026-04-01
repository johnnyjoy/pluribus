package similarity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Repo persists advisory_experiences (advisory ingest / reject bucket).
type Repo struct {
	DB *sql.DB
}

// Create inserts an advisory_experiences row.
func (r *Repo) Create(ctx context.Context, rec *Record) error {
	if r == nil || r.DB == nil {
		return errors.New("similarity: repo not configured")
	}
	tags, err := json.Marshal(rec.Tags)
	if err != nil {
		return err
	}
	var rel interface{}
	if rec.RelatedMemoryID != nil {
		rel = *rec.RelatedMemoryID
	}
	ents := rec.Entities
	if ents == nil {
		ents = []string{}
	}
	ent, err := json.Marshal(ents)
	if err != nil {
		return err
	}
	var occ interface{}
	if rec.OccurredAt != nil {
		occ = *rec.OccurredAt
	}
	status := rec.MemoryFormationStatus
	if status == "" {
		status = FormationRejected
	}
	q := `INSERT INTO advisory_experiences (summary_text, source, tags, related_memory_id, occurred_at, entities, memory_formation_status)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6::jsonb, $7)
		RETURNING id, created_at`
	return r.DB.QueryRowContext(ctx, q, rec.SummaryText, rec.Source, tags, rel, occ, ent, status).Scan(&rec.ID, &rec.CreatedAt)
}

// FindMcpDuplicateInWindow returns a recent advisory_experiences row with source=mcp, same summary_text,
// matching correlation session (see mcpDedupCorrelationMatch), and created_at within the window. Nil if none.
func (r *Repo) FindMcpDuplicateInWindow(ctx context.Context, summary string, correlationID string, window time.Duration) (*Record, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("similarity: repo not configured")
	}
	if window <= 0 {
		return nil, nil
	}
	since := time.Now().Add(-window).UTC()
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, summary_text, source, tags, related_memory_id, created_at, occurred_at, entities, memory_formation_status, rejection_reason
		FROM advisory_experiences
		WHERE source = 'mcp'
		  AND summary_text = $1
		  AND created_at > $2
		ORDER BY created_at DESC
		LIMIT 32`, summary, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		rec, err := scanOneAdvisoryRow(rows)
		if err != nil {
			return nil, err
		}
		if mcpDedupCorrelationMatch(correlationID, rec.Tags) {
			return rec, nil
		}
	}
	return nil, rows.Err()
}

// ListCandidates returns rows for episodic similarity, ordered by effective time (newest first).
// Excludes rejected ingest rows (reject bucket retained for inspection only).
func (r *Repo) ListCandidates(ctx context.Context, limit int, occurredAfter, occurredBefore *time.Time) ([]Record, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("similarity: repo not configured")
	}
	if limit <= 0 {
		limit = 500
	}
	var after, before interface{}
	if occurredAfter != nil {
		after = *occurredAfter
	}
	if occurredBefore != nil {
		before = *occurredBefore
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, summary_text, source, tags, related_memory_id, created_at, occurred_at, entities, memory_formation_status, rejection_reason
		FROM advisory_experiences
		WHERE memory_formation_status <> $4
		  AND ($1::timestamptz IS NULL OR COALESCE(occurred_at, created_at) >= $1)
		  AND ($2::timestamptz IS NULL OR COALESCE(occurred_at, created_at) <= $2)
		ORDER BY COALESCE(occurred_at, created_at) DESC
		LIMIT $3`, after, before, limit, FormationRejected)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdvisoryRows(rows)
}

// GetByID returns one advisory experience or nil if not found.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("similarity: repo not configured")
	}
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, summary_text, source, tags, related_memory_id, created_at, occurred_at, entities, memory_formation_status, rejection_reason
		FROM advisory_experiences WHERE id = $1`, id)
	rec, err := scanOneAdvisoryRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rec, nil
}

type sqlRowScanner interface {
	Scan(dest ...any) error
}

func scanOneAdvisoryRow(row sqlRowScanner) (*Record, error) {
	var rec Record
	var tags, ent []byte
	var rel sql.NullString
	var occ sql.NullTime
	var reject sql.NullString
	if err := row.Scan(&rec.ID, &rec.SummaryText, &rec.Source, &tags, &rel, &rec.CreatedAt, &occ, &ent, &rec.MemoryFormationStatus, &reject); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(tags, &rec.Tags); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(ent, &rec.Entities); err != nil {
		return nil, err
	}
	if rel.Valid {
		id, err := uuid.Parse(rel.String)
		if err == nil {
			rec.RelatedMemoryID = &id
		}
	}
	if occ.Valid {
		t := occ.Time
		rec.OccurredAt = &t
	}
	if reject.Valid {
		rec.RejectionReason = reject.String
	}
	if rec.MemoryFormationStatus == "" || rec.MemoryFormationStatus == "none" {
		rec.MemoryFormationStatus = FormationRejected
	}
	return &rec, nil
}

// SetRelatedMemoryID links an advisory experience to a probationary memory row and marks formation linked.
func (r *Repo) SetRelatedMemoryID(ctx context.Context, episodeID, memoryID uuid.UUID) error {
	if r == nil || r.DB == nil {
		return errors.New("similarity: repo not configured")
	}
	res, err := r.DB.ExecContext(ctx,
		`UPDATE advisory_experiences SET related_memory_id = $1, memory_formation_status = $2, rejection_reason = NULL WHERE id = $3`,
		memoryID, FormationLinked, episodeID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("similarity: advisory experience not found")
	}
	return nil
}

// SetFormationRejected marks a row as rejected at ingest (no probationary memory).
func (r *Repo) SetFormationRejected(ctx context.Context, episodeID uuid.UUID, reason string) error {
	if r == nil || r.DB == nil {
		return errors.New("similarity: repo not configured")
	}
	res, err := r.DB.ExecContext(ctx,
		`UPDATE advisory_experiences SET memory_formation_status = $1, rejection_reason = $2 WHERE id = $3`,
		FormationRejected, reason, episodeID,
	)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("similarity: advisory experience not found")
	}
	return nil
}

// DeleteRejectedOlderThan removes rejected rows older than the cutoff (admin / startup hygiene).
func (r *Repo) DeleteRejectedOlderThan(ctx context.Context, cutoff time.Time, limit int) (int64, error) {
	if r == nil || r.DB == nil {
		return 0, errors.New("similarity: repo not configured")
	}
	if limit <= 0 {
		limit = 10_000
	}
	res, err := r.DB.ExecContext(ctx, `
		DELETE FROM advisory_experiences
		WHERE id IN (
			SELECT id FROM advisory_experiences
			WHERE memory_formation_status = $1 AND created_at < $2
			LIMIT $3
		)`, FormationRejected, cutoff, limit)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func scanAdvisoryRows(rows *sql.Rows) ([]Record, error) {
	var out []Record
	for rows.Next() {
		rec, err := scanOneAdvisoryRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *rec)
	}
	return out, rows.Err()
}
