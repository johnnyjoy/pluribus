package recall

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// BundleRow is the persisted recall bundle (id, payload, created_at).
type BundleRow struct {
	ID        uuid.UUID
	Payload   RecallBundle
	CreatedAt interface{} // time.Time from DB
}

// Repo performs recall bundle persistence.
type Repo struct {
	DB *sql.DB
}

// CreateBundle inserts a recall bundle; payload is stored as JSONB.
func (r *Repo) CreateBundle(ctx context.Context, payload RecallBundle) (id uuid.UUID, err error) {
	id = uuid.New()
	raw, err := json.Marshal(payload)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO recall_bundles (id, payload, created_at)
		 VALUES ($1, $2, now())`,
		id, raw)
	return id, err
}
