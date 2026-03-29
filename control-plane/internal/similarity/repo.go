package similarity

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

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
	q := `INSERT INTO advisory_episodes (summary_text, source, tags, related_memory_id)
		VALUES ($1, $2, $3::jsonb, $4)
		RETURNING id, created_at`
	return r.DB.QueryRowContext(ctx, q, rec.SummaryText, rec.Source, tags, rel).Scan(&rec.ID, &rec.CreatedAt)
}

// ListRecent returns up to limit rows newest-first for similarity scan.
func (r *Repo) ListRecent(ctx context.Context, limit int) ([]Record, error) {
	if r == nil || r.DB == nil {
		return nil, errors.New("similarity: repo not configured")
	}
	if limit <= 0 {
		limit = 500
	}
	rows, err := r.DB.QueryContext(ctx, `
		SELECT id, summary_text, source, tags, related_memory_id, created_at
		FROM advisory_episodes
		ORDER BY created_at DESC
		LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Record
	for rows.Next() {
		var rec Record
		var tags []byte
		var rel sql.NullString
		if err := rows.Scan(&rec.ID, &rec.SummaryText, &rec.Source, &tags, &rel, &rec.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(tags, &rec.Tags); err != nil {
			return nil, err
		}
		if rel.Valid {
			id, err := uuid.Parse(rel.String)
			if err == nil {
				rec.RelatedMemoryID = &id
			}
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}
