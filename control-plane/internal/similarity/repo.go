package similarity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// Repo persists advisory_episodes.
type Repo struct {
	DB *sql.DB
}

// Create inserts an advisory_episodes row.
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
	q := `INSERT INTO advisory_episodes (summary_text, source, tags, related_memory_id, occurred_at, entities)
		VALUES ($1, $2, $3::jsonb, $4, $5, $6::jsonb)
		RETURNING id, created_at`
	return r.DB.QueryRowContext(ctx, q, rec.SummaryText, rec.Source, tags, rel, occ, ent).Scan(&rec.ID, &rec.CreatedAt)
}

// ListCandidates returns rows for episodic similarity, ordered by effective time (newest first).
// occurredAfter / occurredBefore filter on COALESCE(occurred_at, created_at); nil means no bound.
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
		SELECT id, summary_text, source, tags, related_memory_id, created_at, occurred_at, entities
		FROM advisory_episodes
		WHERE ($1::timestamptz IS NULL OR COALESCE(occurred_at, created_at) >= $1)
		  AND ($2::timestamptz IS NULL OR COALESCE(occurred_at, created_at) <= $2)
		ORDER BY COALESCE(occurred_at, created_at) DESC
		LIMIT $3`, after, before, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdvisoryRows(rows)
}

// GetByID returns one advisory episode or nil if not found.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("similarity: repo not configured")
	}
	row := r.DB.QueryRowContext(ctx, `
		SELECT id, summary_text, source, tags, related_memory_id, created_at, occurred_at, entities
		FROM advisory_episodes WHERE id = $1`, id)
	rec, err := scanOneAdvisoryRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return rec, nil
}

// sqlRowScanner matches *sql.Row and *sql.Rows for scanOneAdvisoryRow (single row).
type sqlRowScanner interface {
	Scan(dest ...any) error
}

func scanOneAdvisoryRow(row sqlRowScanner) (*Record, error) {
	var rec Record
	var tags, ent []byte
	var rel sql.NullString
	var occ sql.NullTime
	if err := row.Scan(&rec.ID, &rec.SummaryText, &rec.Source, &tags, &rel, &rec.CreatedAt, &occ, &ent); err != nil {
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
	return &rec, nil
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
