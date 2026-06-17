package utilitypolicy

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Repo persists utility policy applications to Postgres.
type Repo struct {
	DB *sql.DB
}

func (r *Repo) useDB() bool {
	return r != nil && r.DB != nil
}

// SchemaMigrationRate reports whether migration 0013 tables exist.
func (r *Repo) SchemaMigrationRate(ctx context.Context) float64 {
	if !r.useDB() {
		return 0
	}
	var n int
	err := r.DB.QueryRowContext(ctx, `
SELECT 1 FROM information_schema.tables
WHERE table_schema = 'public' AND table_name = 'agent_utility_applications' LIMIT 1`).Scan(&n)
	if err == sql.ErrNoRows {
		return 0
	}
	if err != nil {
		return 0
	}
	return 1.0
}

// HasApplication returns true if candidate_id was already applied.
func (r *Repo) HasApplication(ctx context.Context, candidateID uuid.UUID) (bool, error) {
	if !r.useDB() {
		return false, nil
	}
	var n int
	err := r.DB.QueryRowContext(ctx,
		`SELECT 1 FROM agent_utility_applications WHERE candidate_id = $1 LIMIT 1`, candidateID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CountPositiveBySession counts apply_positive for session+memory.
func (r *Repo) CountPositiveBySession(ctx context.Context, sessionID uuid.UUID, memoryID string) (int, error) {
	if !r.useDB() {
		return 0, nil
	}
	var n int
	err := r.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM agent_utility_applications
WHERE session_id = $1 AND memory_id = $2 AND decision = 'apply_positive' AND reverted_at IS NULL`,
		sessionID, memoryID).Scan(&n)
	return n, err
}

// CountPositiveByAgent counts apply_positive for agent+memory.
func (r *Repo) CountPositiveByAgent(ctx context.Context, agentID, memoryID string) (int, error) {
	if !r.useDB() {
		return 0, nil
	}
	var n int
	err := r.DB.QueryRowContext(ctx, `
SELECT COUNT(*) FROM agent_utility_applications
WHERE agent_id = $1 AND memory_id = $2 AND decision = 'apply_positive' AND reverted_at IS NULL`,
		agentID, memoryID).Scan(&n)
	return n, err
}

// InsertApplication persists one application row.
func (r *Repo) InsertApplication(ctx context.Context, rec ApplicationRecord) error {
	if !r.useDB() {
		return nil
	}
	evidence, _ := json.Marshal(rec.Evidence)
	if evidence == nil {
		evidence = []byte("[]")
	}
	id := rec.ApplicationID
	if id == uuid.Nil {
		id = uuid.New()
	}
	var sessionID any
	if rec.SessionID != uuid.Nil {
		sessionID = rec.SessionID
	}
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_utility_applications (
  application_id, candidate_id, memory_id, evaluation_id, decision, delta,
  previous_utility_score, new_utility_score, policy_version, reason, evidence_json,
  rollback_token, applied_by, session_id, agent_id, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11::jsonb,$12,$13,$14,$15,$16)`,
		id, rec.CandidateID, rec.MemoryID, rec.EvaluationID, rec.Decision, rec.Delta,
		rec.PreviousUtilityScore, rec.NewUtilityScore, rec.PolicyVersion, rec.Reason, evidence,
		rec.RollbackToken, rec.AppliedBy, sessionID, rec.AgentID, rec.CreatedAt)
	if err != nil {
		return err
	}
	rec.ApplicationID = id
	return nil
}

// GetByRollbackToken loads application by rollback token.
func (r *Repo) GetByRollbackToken(ctx context.Context, token string) (*ApplicationRecord, error) {
	if !r.useDB() {
		return nil, ErrApplicationNotFound
	}
	var rec ApplicationRecord
	var evidence []byte
	var sessionID sql.NullString
	var revertedAt sql.NullTime
	err := r.DB.QueryRowContext(ctx, `
SELECT application_id, candidate_id, memory_id, evaluation_id, decision, delta,
  previous_utility_score, new_utility_score, policy_version, reason, evidence_json,
  rollback_token, applied_by, session_id, agent_id, created_at, reverted_at, revert_reason
FROM agent_utility_applications WHERE rollback_token = $1`, token).Scan(
		&rec.ApplicationID, &rec.CandidateID, &rec.MemoryID, &rec.EvaluationID, &rec.Decision, &rec.Delta,
		&rec.PreviousUtilityScore, &rec.NewUtilityScore, &rec.PolicyVersion, &rec.Reason, &evidence,
		&rec.RollbackToken, &rec.AppliedBy, &sessionID, &rec.AgentID, &rec.CreatedAt, &revertedAt, &rec.RevertReason,
	)
	if err == sql.ErrNoRows {
		return nil, ErrApplicationNotFound
	}
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(evidence, &rec.Evidence)
	if sessionID.Valid {
		rec.SessionID, _ = uuid.Parse(sessionID.String)
	}
	if revertedAt.Valid {
		t := revertedAt.Time
		rec.RevertedAt = &t
	}
	return &rec, nil
}

// MarkReverted sets reverted_at on an application.
func (r *Repo) MarkReverted(ctx context.Context, applicationID uuid.UUID, reason string) error {
	if !r.useDB() {
		return nil
	}
	_, err := r.DB.ExecContext(ctx, `
UPDATE agent_utility_applications SET reverted_at = $2, revert_reason = $3
WHERE application_id = $1 AND reverted_at IS NULL`,
		applicationID, time.Now().UTC(), reason)
	return err
}

// ListByMemory returns applications for a memory.
func (r *Repo) ListByMemory(ctx context.Context, memoryID string, limit int) ([]ApplicationRecord, error) {
	if !r.useDB() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT application_id, candidate_id, memory_id, evaluation_id, decision, delta,
  previous_utility_score, new_utility_score, policy_version, reason, evidence_json,
  rollback_token, applied_by, session_id, agent_id, created_at, reverted_at, revert_reason
FROM agent_utility_applications WHERE memory_id = $1 ORDER BY created_at DESC LIMIT $2`, memoryID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows)
}

// ListAll returns recent applications.
func (r *Repo) ListAll(ctx context.Context, limit int) ([]ApplicationRecord, error) {
	if !r.useDB() {
		return nil, nil
	}
	if limit <= 0 {
		limit = 200
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT application_id, candidate_id, memory_id, evaluation_id, decision, delta,
  previous_utility_score, new_utility_score, policy_version, reason, evidence_json,
  rollback_token, applied_by, session_id, agent_id, created_at, reverted_at, revert_reason
FROM agent_utility_applications ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanApplications(rows)
}

func scanApplications(rows *sql.Rows) ([]ApplicationRecord, error) {
	var out []ApplicationRecord
	for rows.Next() {
		var rec ApplicationRecord
		var evidence []byte
		var sessionID sql.NullString
		var revertedAt sql.NullTime
		if err := rows.Scan(
			&rec.ApplicationID, &rec.CandidateID, &rec.MemoryID, &rec.EvaluationID, &rec.Decision, &rec.Delta,
			&rec.PreviousUtilityScore, &rec.NewUtilityScore, &rec.PolicyVersion, &rec.Reason, &evidence,
			&rec.RollbackToken, &rec.AppliedBy, &sessionID, &rec.AgentID, &rec.CreatedAt, &revertedAt, &rec.RevertReason,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(evidence, &rec.Evidence)
		if sessionID.Valid {
			rec.SessionID, _ = uuid.Parse(sessionID.String)
		}
		if revertedAt.Valid {
			t := revertedAt.Time
			rec.RevertedAt = &t
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// Summary aggregates application counts.
func (r *Repo) Summary(ctx context.Context) (PolicySummary, error) {
	if !r.useDB() {
		return PolicySummary{PolicyVersion: PolicyVersion}, nil
	}
	var s PolicySummary
	s.PolicyVersion = PolicyVersion
	rows, err := r.DB.QueryContext(ctx, `SELECT decision, reverted_at IS NOT NULL FROM agent_utility_applications`)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var decision string
		var reverted bool
		if err := rows.Scan(&decision, &reverted); err != nil {
			return s, err
		}
		s.TotalApplications++
		switch decision {
		case DecisionApplyPositive:
			s.PositiveApplications++
		case DecisionApplyNegative:
			s.NegativeApplications++
		case DecisionRecordOnly:
			s.RecordOnlyCount++
		case DecisionReviewRequired:
			s.ReviewRequiredCount++
		case DecisionReject:
			s.RejectedCount++
		}
		if reverted {
			s.RevertedCount++
		}
	}
	return s, rows.Err()
}
