package agenttelemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
)

// ErrCandidateNotFound is returned when utility candidate id is unknown.
var ErrCandidateNotFound = errors.New("utility candidate not found")

// ErrEvaluationNotFound is returned when evaluation id is unknown.
var ErrEvaluationNotFound = errors.New("obedience evaluation not found")

// PolicyReader implements utilitypolicy.TelemetryReader.
type PolicyReader struct {
	Repo *Repo
}

// GetCandidate loads one utility candidate by id.
func (p *PolicyReader) GetCandidate(ctx context.Context, id uuid.UUID) (*UtilityCandidate, error) {
	if p == nil || p.Repo == nil {
		return nil, ErrCandidateNotFound
	}
	c, ok := p.Repo.getCandidate(ctx, id)
	if !ok {
		return nil, ErrCandidateNotFound
	}
	return &c, nil
}

// GetEvaluation loads one obedience evaluation by id.
func (p *PolicyReader) GetEvaluation(ctx context.Context, id uuid.UUID) (*ObedienceEvaluationRow, error) {
	if p == nil || p.Repo == nil {
		return nil, ErrEvaluationNotFound
	}
	e, ok := p.Repo.getEvaluation(ctx, id)
	if !ok {
		return nil, ErrEvaluationNotFound
	}
	return &e, nil
}

func (r *Repo) getCandidate(ctx context.Context, id uuid.UUID) (UtilityCandidate, bool) {
	if r.useDB() {
		var c UtilityCandidate
		err := r.DB.QueryRowContext(ctx, `
SELECT id, memory_id, evaluation_id, signal_type, signal_strength, safe_to_apply, reason, created_at
FROM agent_utility_candidates WHERE id = $1`, id).Scan(
			&c.ID, &c.MemoryID, &c.EvaluationID, &c.SignalType, &c.SignalStrength, &c.SafeToApply, &c.Reason, &c.CreatedAt,
		)
		if err == sql.ErrNoRows {
			return UtilityCandidate{}, false
		}
		if err != nil {
			return UtilityCandidate{}, false
		}
		return c, true
	}
	return UtilityCandidate{}, false
}

func (r *Repo) getEvaluation(ctx context.Context, id uuid.UUID) (ObedienceEvaluationRow, bool) {
	if !r.useDB() {
		return ObedienceEvaluationRow{}, false
	}
	var e ObedienceEvaluationRow
	var viol []byte
	var oid sql.NullString
	err := r.DB.QueryRowContext(ctx, `
SELECT id, session_id, task_id, recall_event_id, output_id, obedience_passed, obedience_score,
  violations, evaluator_version, created_at
FROM agent_obedience_evaluations WHERE id = $1`, id).Scan(
		&e.ID, &e.SessionID, &e.TaskID, &e.RecallEventID, &oid, &e.ObediencePassed, &e.ObedienceScore,
		&viol, &e.EvaluatorVersion, &e.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return ObedienceEvaluationRow{}, false
	}
	if err != nil {
		return ObedienceEvaluationRow{}, false
	}
	if oid.Valid {
		e.OutputID, _ = uuid.Parse(oid.String)
	}
	_ = json.Unmarshal(viol, &e.Violations)
	return e, true
}
