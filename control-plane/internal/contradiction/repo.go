package contradiction

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// Repo persists contradiction records.
type Repo struct {
	DB *sql.DB
}

// Create inserts a contradiction record. resolution_state defaults to unresolved.
func (r *Repo) Create(ctx context.Context, req CreateRequest) (*Record, error) {
	state := req.ResolutionState
	if state == "" {
		state = ResolutionUnresolved
	}
	if !validResolutionState(state) {
		state = ResolutionUnresolved
	}
	id := uuid.New()
	var rec Record
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO contradiction_records (id, memory_id, conflict_with_id, resolution_state)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, memory_id, conflict_with_id, resolution_state, created_at, updated_at`,
		id, req.MemoryID, req.ConflictWithID, state,
	).Scan(&rec.ID, &rec.MemoryID, &rec.ConflictWithID, &rec.ResolutionState, &rec.CreatedAt, &rec.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// GetByID returns a contradiction record by ID.
func (r *Repo) GetByID(ctx context.Context, id uuid.UUID) (*Record, error) {
	var rec Record
	err := r.DB.QueryRowContext(ctx,
		`SELECT id, memory_id, conflict_with_id, resolution_state, created_at, updated_at
		 FROM contradiction_records WHERE id = $1`,
		id,
	).Scan(&rec.ID, &rec.MemoryID, &rec.ConflictWithID, &rec.ResolutionState, &rec.CreatedAt, &rec.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// List returns contradiction records matching the request.
func (r *Repo) List(ctx context.Context, req ListRequest) ([]Record, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = 50
	}
	var rows *sql.Rows
	var err error
	if req.ResolutionState != "" && req.MemoryID != uuid.Nil {
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, memory_id, conflict_with_id, resolution_state, created_at, updated_at
			 FROM contradiction_records
			 WHERE resolution_state = $1 AND (memory_id = $2 OR conflict_with_id = $2)
			 ORDER BY created_at DESC LIMIT $3`,
			req.ResolutionState, req.MemoryID, limit)
	} else if req.ResolutionState != "" {
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, memory_id, conflict_with_id, resolution_state, created_at, updated_at
			 FROM contradiction_records WHERE resolution_state = $1 ORDER BY created_at DESC LIMIT $2`,
			req.ResolutionState, limit)
	} else if req.MemoryID != uuid.Nil {
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, memory_id, conflict_with_id, resolution_state, created_at, updated_at
			 FROM contradiction_records WHERE memory_id = $1 OR conflict_with_id = $1
			 ORDER BY created_at DESC LIMIT $2`,
			req.MemoryID, limit)
	} else {
		rows, err = r.DB.QueryContext(ctx,
			`SELECT id, memory_id, conflict_with_id, resolution_state, created_at, updated_at
			 FROM contradiction_records ORDER BY created_at DESC LIMIT $1`,
			limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Record
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.ID, &rec.MemoryID, &rec.ConflictWithID, &rec.ResolutionState, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, rec)
	}
	return list, rows.Err()
}

// UpdateResolution sets resolution_state and updated_at.
func (r *Repo) UpdateResolution(ctx context.Context, id uuid.UUID, resolutionState string) error {
	if !validResolutionState(resolutionState) {
		resolutionState = ResolutionUnresolved
	}
	_, err := r.DB.ExecContext(ctx,
		`UPDATE contradiction_records SET resolution_state = $1, updated_at = now() WHERE id = $2`,
		resolutionState, id)
	return err
}

// ListMemoryIDsInUnresolved returns all memory IDs that appear in any unresolved contradiction (either as memory_id or conflict_with_id).
// Used by recall to exclude these from the RIE bundle.
func (r *Repo) ListMemoryIDsInUnresolved(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := r.DB.QueryContext(ctx,
		`SELECT memory_id FROM contradiction_records WHERE resolution_state = 'unresolved'
		 UNION
		 SELECT conflict_with_id FROM contradiction_records WHERE resolution_state = 'unresolved'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids, rows.Err()
}

// ListUnresolvedPairs returns (memory_id, conflict_with_id) pairs for unresolved contradictions, ordered for determinism.
func (r *Repo) ListUnresolvedPairs(ctx context.Context, limit int) ([][2]uuid.UUID, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT memory_id, conflict_with_id FROM contradiction_records
		 WHERE resolution_state = 'unresolved'
		 ORDER BY memory_id::text, conflict_with_id::text
		 LIMIT $1`,
		limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out [][2]uuid.UUID
	for rows.Next() {
		var a, b uuid.UUID
		if err := rows.Scan(&a, &b); err != nil {
			return nil, err
		}
		out = append(out, [2]uuid.UUID{a, b})
	}
	return out, rows.Err()
}

func validResolutionState(s string) bool {
	switch s {
	case ResolutionUnresolved, ResolutionOverride, ResolutionDeprecated, ResolutionNarrowException:
		return true
	default:
		return false
	}
}
