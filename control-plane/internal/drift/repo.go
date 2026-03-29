package drift

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
)

// Repo performs drift check persistence.
type Repo struct {
	DB *sql.DB
}

// CreateCheck inserts a drift check record; violations and warnings stored as JSONB.
func (r *Repo) CreateCheck(ctx context.Context, passed bool, violations []DriftIssue, warnings []string) (id uuid.UUID, err error) {
	id = uuid.New()
	var violJSON, warnJSON interface{}
	if violations != nil {
		violJSON, _ = json.Marshal(violations)
	}
	if warnings != nil {
		warnJSON, _ = json.Marshal(warnings)
	}
	_, err = r.DB.ExecContext(ctx,
		`INSERT INTO drift_checks (id, passed, violations, warnings, created_at)
		 VALUES ($1, $2, $3, $4, now())`,
		id, passed, violJSON, warnJSON)
	return id, err
}
