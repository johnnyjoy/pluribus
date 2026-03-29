package evidence

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// Repo performs evidence record persistence.
type Repo struct {
	DB *sql.DB
}

// Create inserts an evidence record; returns it with ID and CreatedAt set.
func (r *Repo) Create(ctx context.Context, digest, path, kind string) (*Record, error) {
	id := uuid.New()
	var rec Record
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO evidence_records (id, digest, path, kind, created_at)
		 VALUES ($1, $2, $3, $4, now())
		 RETURNING id, digest, path, COALESCE(kind,''), created_at`,
		id, digest, path, nullString(kind),
	).Scan(&rec.ID, &rec.Digest, &rec.Path, &rec.Kind, &rec.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// List returns evidence records, optionally filtered by kind (empty = all).
func (r *Repo) List(ctx context.Context, kind string) ([]Record, error) {
	var rows *sql.Rows
	var err error
	if kind != "" {
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, digest, path, COALESCE(kind,''), created_at
			 FROM evidence_records WHERE kind = $1 ORDER BY created_at DESC`,
			kind)
	} else {
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, digest, path, COALESCE(kind,''), created_at
			 FROM evidence_records ORDER BY created_at DESC`)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Digest, &rec.Path, &rec.Kind, &rec.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

// GetByID returns one evidence record by id, or nil if not found.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	var rec Record
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, digest, path, COALESCE(kind,''), created_at
		 FROM evidence_records WHERE id = $1`,
		id,
	).Scan(&rec.ID, &rec.Digest, &rec.Path, &rec.Kind, &rec.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// CreateLink links an evidence record to a memory object (memory_evidence_links).
func (r *Repo) CreateLink(ctx context.Context, memoryID, evidenceID uuid.UUID) error {
	_, err := r.DB.ExecContext(ctx,
		`INSERT INTO memory_evidence_links (memory_id, evidence_id) VALUES ($1, $2) ON CONFLICT (memory_id, evidence_id) DO NOTHING`,
		memoryID, evidenceID)
	return err
}

// ListEvidenceByMemory returns all evidence records linked to the given memory (Task 79: traceability).
func (r *Repo) ListEvidenceByMemory(ctx context.Context, memoryID uuid.UUID) ([]Record, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT e.id, e.digest, e.path, COALESCE(e.kind,''), e.created_at
		 FROM evidence_records e
		 INNER JOIN memory_evidence_links l ON l.evidence_id = e.id
		 WHERE l.memory_id = $1 ORDER BY e.created_at DESC`,
		memoryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.Digest, &rec.Path, &rec.Kind, &rec.CreatedAt); err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
