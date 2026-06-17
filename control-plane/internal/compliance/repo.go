package compliance

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repo persists compliance telemetry in Postgres.
type Repo struct {
	DB *sql.DB
}

// UpsertSession creates or updates last_seen for a session.
func (r *Repo) UpsertSession(ctx context.Context, s Session) error {
	if r == nil || r.DB == nil {
		return nil
	}
	meta, _ := json.Marshal(s.Metadata)
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_sessions (id, started_at, last_seen_at, client_name, client_version, transport, repo_root, workspace_hint, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT (id) DO UPDATE SET
  last_seen_at = EXCLUDED.last_seen_at,
  client_name = CASE WHEN EXCLUDED.client_name <> '' THEN EXCLUDED.client_name ELSE agent_sessions.client_name END,
  client_version = CASE WHEN EXCLUDED.client_version <> '' THEN EXCLUDED.client_version ELSE agent_sessions.client_version END,
  repo_root = CASE WHEN EXCLUDED.repo_root <> '' THEN EXCLUDED.repo_root ELSE agent_sessions.repo_root END,
  workspace_hint = CASE WHEN EXCLUDED.workspace_hint <> '' THEN EXCLUDED.workspace_hint ELSE agent_sessions.workspace_hint END,
  metadata = agent_sessions.metadata || EXCLUDED.metadata
`, s.ID, s.StartedAt, s.LastSeenAt, s.ClientName, s.ClientVersion, s.Transport, s.RepoRoot, s.WorkspaceHint, meta)
	return err
}

// InsertEvent records one telemetry event.
func (r *Repo) InsertEvent(ctx context.Context, e Event) (uuid.UUID, error) {
	if r == nil || r.DB == nil {
		return uuid.Nil, nil
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	meta, _ := json.Marshal(e.Metadata)
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_loop_events (
  id, session_id, occurred_at, event_type, tool_name, loop_role, risk_level, correlation_id,
  request_hash, request_summary, result_status, error_code, error_message, duration_ms,
  enforcement_decision, memory_id, metadata
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
`, e.ID, e.SessionID, e.OccurredAt, e.EventType, e.ToolName, e.LoopRole, e.RiskLevel, e.CorrelationID,
		e.RequestHash, e.RequestSummary, e.ResultStatus, e.ErrorCode, e.ErrorMessage, e.DurationMS,
		e.EnforcementDecision, e.MemoryID, meta)
	return e.ID, err
}

// ListEvents returns events for a session in time order.
func (r *Repo) ListEvents(ctx context.Context, sessionID uuid.UUID, from, to time.Time) ([]Event, error) {
	if r == nil || r.DB == nil {
		return nil, nil
	}
	q := `SELECT id, session_id, occurred_at, event_type, tool_name, loop_role, risk_level, correlation_id,
  request_hash, request_summary, result_status, error_code, error_message, duration_ms,
  enforcement_decision, memory_id, metadata
FROM agent_loop_events WHERE session_id = $1`
	args := []any{sessionID}
	n := 2
	if !from.IsZero() {
		q += fmt.Sprintf(` AND occurred_at >= $%d`, n)
		args = append(args, from)
		n++
	}
	if !to.IsZero() {
		q += fmt.Sprintf(` AND occurred_at <= $%d`, n)
		args = append(args, to)
	}
	q += ` ORDER BY occurred_at ASC, id ASC`
	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Event
	for rows.Next() {
		var e Event
		var meta []byte
		if err := rows.Scan(&e.ID, &e.SessionID, &e.OccurredAt, &e.EventType, &e.ToolName, &e.LoopRole, &e.RiskLevel,
			&e.CorrelationID, &e.RequestHash, &e.RequestSummary, &e.ResultStatus, &e.ErrorCode, &e.ErrorMessage,
			&e.DurationMS, &e.EnforcementDecision, &e.MemoryID, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &e.Metadata)
		out = append(out, e)
	}
	return out, rows.Err()
}

// GetSession returns one session by ID.
func (r *Repo) GetSession(ctx context.Context, id uuid.UUID) (*Session, error) {
	if r == nil || r.DB == nil {
		return nil, sql.ErrNoRows
	}
	var s Session
	var meta []byte
	err := r.DB.QueryRowContext(ctx, `
SELECT id, started_at, last_seen_at, client_name, client_version, transport, repo_root, workspace_hint, metadata
FROM agent_sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.StartedAt, &s.LastSeenAt, &s.ClientName, &s.ClientVersion, &s.Transport, &s.RepoRoot, &s.WorkspaceHint, &meta)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(meta, &s.Metadata)
	return &s, nil
}

// ListSessions returns recent sessions.
func (r *Repo) ListSessions(ctx context.Context, limit int) ([]Session, error) {
	if r == nil || r.DB == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, started_at, last_seen_at, client_name, client_version, transport, repo_root, workspace_hint, metadata
FROM agent_sessions ORDER BY last_seen_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Session
	for rows.Next() {
		var s Session
		var meta []byte
		if err := rows.Scan(&s.ID, &s.StartedAt, &s.LastSeenAt, &s.ClientName, &s.ClientVersion, &s.Transport, &s.RepoRoot, &s.WorkspaceHint, &meta); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(meta, &s.Metadata)
		out = append(out, s)
	}
	return out, rows.Err()
}

// InsertEvaluation stores evaluation result.
func (r *Repo) InsertEvaluation(ctx context.Context, e Evaluation) (uuid.UUID, error) {
	if r == nil || r.DB == nil {
		return uuid.Nil, nil
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	missing, _ := json.Marshal(e.MissingSteps)
	evidence, _ := json.Marshal(e.Evidence)
	meta, _ := json.Marshal(e.Metadata)
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_loop_evaluations (id, session_id, evaluated_at, window_start, window_end, status, missing_steps, evidence, metadata)
VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
`, e.ID, e.SessionID, e.EvaluatedAt, e.WindowStart, e.WindowEnd, e.Status, missing, evidence, meta)
	return e.ID, err
}
