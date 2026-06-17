package agenttelemetry

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Repo persists agent telemetry in Postgres.
type Repo struct {
	DB *sql.DB
}

func (r *Repo) useDB() bool {
	return r != nil && r.DB != nil
}

func (r *Repo) insertSession(ctx context.Context, s TelemetrySession) error {
	if !r.useDB() {
		return nil
	}
	tags, _ := json.Marshal(s.Tags)
	meta, _ := json.Marshal(s.Metadata)
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_telemetry_sessions (id, started_at, ended_at, interface, agent_id, client_name, tags, metadata_json)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (id) DO NOTHING
`, s.ID, s.StartedAt, s.EndedAt, s.Interface, s.AgentID, s.ClientName, tags, meta)
	return err
}

func (r *Repo) getSession(ctx context.Context, id uuid.UUID) (*TelemetrySession, bool) {
	if !r.useDB() {
		return nil, false
	}
	var s TelemetrySession
	var tags, meta []byte
	var ended sql.NullTime
	err := r.DB.QueryRowContext(ctx, `
SELECT id, started_at, ended_at, interface, agent_id, client_name, tags, metadata_json
FROM agent_telemetry_sessions WHERE id = $1`, id).Scan(
		&s.ID, &s.StartedAt, &ended, &s.Interface, &s.AgentID, &s.ClientName, &tags, &meta)
	if err != nil {
		return nil, false
	}
	if ended.Valid {
		s.EndedAt = &ended.Time
	}
	_ = json.Unmarshal(tags, &s.Tags)
	_ = json.Unmarshal(meta, &s.Metadata)
	return &s, true
}

func (r *Repo) insertRecall(ctx context.Context, ev RecallEvent) error {
	if !r.useDB() {
		return nil
	}
	reqJSON, _ := json.Marshal(ev.RecallRequestJSON)
	bundleJSON, _ := json.Marshal(ev.RecallBundleJSON)
	idsJSON, _ := json.Marshal(ev.RecalledMemoryIDs)
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_recall_events (
  id, session_id, task_id, interface, recall_request_json, recall_bundle_id,
  recalled_memory_ids, recall_bundle_json, recall_mode, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`, ev.ID, ev.SessionID, ev.TaskID, ev.Interface, reqJSON, ev.RecallBundleID,
		idsJSON, bundleJSON, ev.RecallMode, ev.CreatedAt)
	return err
}

func (r *Repo) getRecall(ctx context.Context, id uuid.UUID) (*RecallEvent, bool) {
	if !r.useDB() {
		return nil, false
	}
	var ev RecallEvent
	var reqJSON, bundleJSON, idsJSON []byte
	err := r.DB.QueryRowContext(ctx, `
SELECT id, session_id, task_id, interface, recall_request_json, recall_bundle_id,
  recalled_memory_ids, recall_bundle_json, recall_mode, created_at
FROM agent_recall_events WHERE id = $1`, id).Scan(
		&ev.ID, &ev.SessionID, &ev.TaskID, &ev.Interface, &reqJSON, &ev.RecallBundleID,
		&idsJSON, &bundleJSON, &ev.RecallMode, &ev.CreatedAt)
	if err != nil {
		return nil, false
	}
	_ = json.Unmarshal(reqJSON, &ev.RecallRequestJSON)
	_ = json.Unmarshal(bundleJSON, &ev.RecallBundleJSON)
	_ = json.Unmarshal(idsJSON, &ev.RecalledMemoryIDs)
	return &ev, true
}

func (r *Repo) listRecallsBySession(ctx context.Context, sid uuid.UUID) []RecallEvent {
	if !r.useDB() {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, session_id, task_id, interface, recall_request_json, recall_bundle_id,
  recalled_memory_ids, recall_bundle_json, recall_mode, created_at
FROM agent_recall_events WHERE session_id = $1 ORDER BY created_at ASC`, sid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []RecallEvent
	for rows.Next() {
		var ev RecallEvent
		var reqJSON, bundleJSON, idsJSON []byte
		if err := rows.Scan(&ev.ID, &ev.SessionID, &ev.TaskID, &ev.Interface, &reqJSON, &ev.RecallBundleID,
			&idsJSON, &bundleJSON, &ev.RecallMode, &ev.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(reqJSON, &ev.RecallRequestJSON)
		_ = json.Unmarshal(bundleJSON, &ev.RecallBundleJSON)
		_ = json.Unmarshal(idsJSON, &ev.RecalledMemoryIDs)
		out = append(out, ev)
	}
	return out
}

func (r *Repo) insertDecisions(ctx context.Context, recallID uuid.UUID, rows []MemoryDecisionRow) error {
	if !r.useDB() {
		return nil
	}
	for _, d := range rows {
		cited, _ := json.Marshal(d.ContractFieldsCited)
		facts, _ := json.Marshal(d.OutputFactsSupported)
		viol, _ := json.Marshal(d.ViolationCodes)
		if _, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_memory_decisions (
  id, recall_event_id, memory_id, decision, reason, contract_fields_cited,
  output_facts_supported, violation_codes, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
`, d.ID, recallID, d.MemoryID, d.Decision, d.Reason, cited, facts, viol, d.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) listDecisionsByRecall(ctx context.Context, recallID uuid.UUID) []MemoryDecisionRow {
	if !r.useDB() {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, recall_event_id, memory_id, decision, reason, contract_fields_cited,
  output_facts_supported, violation_codes, created_at
FROM agent_memory_decisions WHERE recall_event_id = $1 ORDER BY created_at ASC`, recallID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	return scanDecisionRows(rows)
}

func scanDecisionRows(rows *sql.Rows) []MemoryDecisionRow {
	var out []MemoryDecisionRow
	for rows.Next() {
		var d MemoryDecisionRow
		var cited, facts, viol []byte
		if err := rows.Scan(&d.ID, &d.RecallEventID, &d.MemoryID, &d.Decision, &d.Reason, &cited, &facts, &viol, &d.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(cited, &d.ContractFieldsCited)
		_ = json.Unmarshal(facts, &d.OutputFactsSupported)
		_ = json.Unmarshal(viol, &d.ViolationCodes)
		out = append(out, d)
	}
	return out
}

func (r *Repo) insertOutput(ctx context.Context, o OutputEvent) error {
	if !r.useDB() {
		return nil
	}
	facts, _ := json.Marshal(o.OutputFacts)
	actions, _ := json.Marshal(o.OutputActions)
	cites, _ := json.Marshal(o.MemoryCitations)
	var rid any
	if o.RecallEventID != uuid.Nil {
		rid = o.RecallEventID
	}
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_output_events (
  id, session_id, task_id, recall_event_id, output_facts, output_actions, memory_citations, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`, o.ID, o.SessionID, o.TaskID, rid, facts, actions, cites, o.CreatedAt)
	return err
}

func (r *Repo) getOutput(ctx context.Context, id uuid.UUID) (*OutputEvent, bool) {
	if !r.useDB() {
		return nil, false
	}
	var o OutputEvent
	var facts, actions, cites []byte
	var rid sql.NullString
	err := r.DB.QueryRowContext(ctx, `
SELECT id, session_id, task_id, recall_event_id, output_facts, output_actions, memory_citations, created_at
FROM agent_output_events WHERE id = $1`, id).Scan(
		&o.ID, &o.SessionID, &o.TaskID, &rid, &facts, &actions, &cites, &o.CreatedAt)
	if err != nil {
		return nil, false
	}
	if rid.Valid {
		if u, err := uuid.Parse(rid.String); err == nil {
			o.RecallEventID = u
		}
	}
	_ = json.Unmarshal(facts, &o.OutputFacts)
	_ = json.Unmarshal(actions, &o.OutputActions)
	_ = json.Unmarshal(cites, &o.MemoryCitations)
	return &o, true
}

func (r *Repo) listOutputsBySession(ctx context.Context, sid uuid.UUID) []OutputEvent {
	if !r.useDB() {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, session_id, task_id, recall_event_id, output_facts, output_actions, memory_citations, created_at
FROM agent_output_events WHERE session_id = $1 ORDER BY created_at ASC`, sid)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []OutputEvent
	for rows.Next() {
		var o OutputEvent
		var facts, actions, cites []byte
		var rid sql.NullString
		if err := rows.Scan(&o.ID, &o.SessionID, &o.TaskID, &rid, &facts, &actions, &cites, &o.CreatedAt); err != nil {
			continue
		}
		if rid.Valid {
			if u, err := uuid.Parse(rid.String); err == nil {
				o.RecallEventID = u
			}
		}
		_ = json.Unmarshal(facts, &o.OutputFacts)
		_ = json.Unmarshal(actions, &o.OutputActions)
		_ = json.Unmarshal(cites, &o.MemoryCitations)
		out = append(out, o)
	}
	return out
}

func (r *Repo) insertEval(ctx context.Context, e ObedienceEvaluationRow) error {
	if !r.useDB() {
		return nil
	}
	viol, _ := json.Marshal(e.Violations)
	var oid any
	if e.OutputID != uuid.Nil {
		oid = e.OutputID
	}
	_, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_obedience_evaluations (
  id, session_id, task_id, recall_event_id, output_id, obedience_passed, obedience_score,
  violations, evaluator_version, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`, e.ID, e.SessionID, e.TaskID, e.RecallEventID, oid, e.ObediencePassed, e.ObedienceScore,
		viol, e.EvaluatorVersion, e.CreatedAt)
	return err
}

func (r *Repo) insertViolations(ctx context.Context, evalID uuid.UUID, rows []ViolationRow) error {
	if !r.useDB() {
		return nil
	}
	for _, v := range rows {
		details, _ := json.Marshal(v.Details)
		if _, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_memory_use_violations (
  id, evaluation_id, memory_id, violation_code, severity, details_json, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7)
`, v.ID, evalID, v.MemoryID, v.ViolationCode, v.Severity, details, v.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) insertCandidates(ctx context.Context, evalID uuid.UUID, rows []UtilityCandidate) error {
	if !r.useDB() {
		return nil
	}
	for _, c := range rows {
		if _, err := r.DB.ExecContext(ctx, `
INSERT INTO agent_utility_candidates (
  id, memory_id, evaluation_id, signal_type, signal_strength, safe_to_apply, reason, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`, c.ID, c.MemoryID, evalID, c.SignalType, c.SignalStrength, c.SafeToApply, c.Reason, c.CreatedAt); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repo) listAllViolations(ctx context.Context) []ViolationRow {
	if !r.useDB() {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, evaluation_id, memory_id, violation_code, severity, details_json, created_at
FROM agent_memory_use_violations ORDER BY created_at ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []ViolationRow
	for rows.Next() {
		var v ViolationRow
		var details []byte
		if err := rows.Scan(&v.ID, &v.EvaluationID, &v.MemoryID, &v.ViolationCode, &v.Severity, &details, &v.CreatedAt); err != nil {
			continue
		}
		_ = json.Unmarshal(details, &v.Details)
		out = append(out, v)
	}
	return out
}

func (r *Repo) listAllCandidates(ctx context.Context) []UtilityCandidate {
	if !r.useDB() {
		return nil
	}
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, memory_id, evaluation_id, signal_type, signal_strength, safe_to_apply, reason, created_at
FROM agent_utility_candidates ORDER BY created_at ASC`)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []UtilityCandidate
	for rows.Next() {
		var c UtilityCandidate
		if err := rows.Scan(&c.ID, &c.MemoryID, &c.EvaluationID, &c.SignalType, &c.SignalStrength, &c.SafeToApply, &c.Reason, &c.CreatedAt); err != nil {
			continue
		}
		out = append(out, c)
	}
	return out
}

func (r *Repo) sessionSummary(ctx context.Context, sid uuid.UUID) *SessionSummary {
	if !r.useDB() {
		return nil
	}
	sess, ok := r.getSession(ctx, sid)
	if !ok {
		return nil
	}
	sum := &SessionSummary{Session: *sess}
	sum.RecallEvents = r.listRecallsBySession(ctx, sid)
	for _, rec := range sum.RecallEvents {
		sum.Decisions = append(sum.Decisions, r.listDecisionsByRecall(ctx, rec.ID)...)
	}
	sum.Outputs = r.listOutputsBySession(ctx, sid)
	rows, err := r.DB.QueryContext(ctx, `
SELECT id, session_id, task_id, recall_event_id, output_id, obedience_passed, obedience_score,
  violations, evaluator_version, created_at
FROM agent_obedience_evaluations WHERE session_id = $1 ORDER BY created_at ASC`, sid)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var e ObedienceEvaluationRow
			var viol []byte
			var oid sql.NullString
			if err := rows.Scan(&e.ID, &e.SessionID, &e.TaskID, &e.RecallEventID, &oid, &e.ObediencePassed, &e.ObedienceScore,
				&viol, &e.EvaluatorVersion, &e.CreatedAt); err != nil {
				continue
			}
			if oid.Valid {
				if u, err := uuid.Parse(oid.String); err == nil {
					e.OutputID = u
				}
			}
			_ = json.Unmarshal(viol, &e.Violations)
			sum.Evaluations = append(sum.Evaluations, e)
			vrows, _ := r.DB.QueryContext(ctx, `
SELECT id, evaluation_id, memory_id, violation_code, severity, details_json, created_at
FROM agent_memory_use_violations WHERE evaluation_id = $1`, e.ID)
			if vrows != nil {
				for vrows.Next() {
					var v ViolationRow
					var details []byte
					if err := vrows.Scan(&v.ID, &v.EvaluationID, &v.MemoryID, &v.ViolationCode, &v.Severity, &details, &v.CreatedAt); err != nil {
						continue
					}
					_ = json.Unmarshal(details, &v.Details)
					sum.Violations = append(sum.Violations, v)
				}
				vrows.Close()
			}
			crows, _ := r.DB.QueryContext(ctx, `
SELECT id, memory_id, evaluation_id, signal_type, signal_strength, safe_to_apply, reason, created_at
FROM agent_utility_candidates WHERE evaluation_id = $1`, e.ID)
			if crows != nil {
				for crows.Next() {
					var c UtilityCandidate
					if err := crows.Scan(&c.ID, &c.MemoryID, &c.EvaluationID, &c.SignalType, &c.SignalStrength, &c.SafeToApply, &c.Reason, &c.CreatedAt); err != nil {
						continue
					}
					sum.Candidates = append(sum.Candidates, c)
				}
				crows.Close()
			}
		}
	}
	return sum
}

func (r *Repo) memorySummary(ctx context.Context, memoryID string) MemorySummary {
	if !r.useDB() {
		return MemorySummary{MemoryID: memoryID}
	}
	sum := MemorySummary{MemoryID: memoryID}
	rows, _ := r.DB.QueryContext(ctx, `SELECT recalled_memory_ids FROM agent_recall_events`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var idsJSON []byte
			if rows.Scan(&idsJSON) != nil {
				continue
			}
			var ids []string
			_ = json.Unmarshal(idsJSON, &ids)
			for _, id := range ids {
				if id == memoryID {
					sum.RecallCount++
				}
			}
		}
	}
	drows, _ := r.DB.QueryContext(ctx, `SELECT decision FROM agent_memory_decisions WHERE memory_id = $1`, memoryID)
	if drows != nil {
		defer drows.Close()
		for drows.Next() {
			var dec string
			if drows.Scan(&dec) != nil {
				continue
			}
			switch dec {
			case "used":
				sum.UsedCount++
			case "ignored", "historical_only":
				sum.IgnoredCount++
			}
		}
	}
	var vcount int
	_ = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FROM agent_memory_use_violations WHERE memory_id = $1`, memoryID).Scan(&vcount)
	sum.ViolationCount = vcount
	var pass, total int
	_ = r.DB.QueryRowContext(ctx, `SELECT COUNT(*) FILTER (WHERE obedience_passed), COUNT(*) FROM agent_obedience_evaluations`).Scan(&pass, &total)
	if total > 0 {
		sum.ObediencePassRate = float64(pass) / float64(total)
	}
	return sum
}

// EvaluateTransactional persists evaluation + violations + candidates atomically.
func (r *Repo) EvaluateTransactional(ctx context.Context, eval ObedienceEvaluationRow, vrows []ViolationRow, cands []UtilityCandidate) error {
	if !r.useDB() {
		return fmt.Errorf("postgres not configured")
	}
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	viol, _ := json.Marshal(eval.Violations)
	var oid any
	if eval.OutputID != uuid.Nil {
		oid = eval.OutputID
	}
	if _, err := tx.ExecContext(ctx, `
INSERT INTO agent_obedience_evaluations (
  id, session_id, task_id, recall_event_id, output_id, obedience_passed, obedience_score,
  violations, evaluator_version, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
`, eval.ID, eval.SessionID, eval.TaskID, eval.RecallEventID, oid, eval.ObediencePassed, eval.ObedienceScore,
		viol, eval.EvaluatorVersion, eval.CreatedAt); err != nil {
		return err
	}
	for _, v := range vrows {
		details, _ := json.Marshal(v.Details)
		if _, err := tx.ExecContext(ctx, `
INSERT INTO agent_memory_use_violations (
  id, evaluation_id, memory_id, violation_code, severity, details_json, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7)
`, v.ID, eval.ID, v.MemoryID, v.ViolationCode, v.Severity, details, v.CreatedAt); err != nil {
			return err
		}
	}
	for _, c := range cands {
		if _, err := tx.ExecContext(ctx, `
INSERT INTO agent_utility_candidates (
  id, memory_id, evaluation_id, signal_type, signal_strength, safe_to_apply, reason, created_at
) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
`, c.ID, c.MemoryID, eval.ID, c.SignalType, c.SignalStrength, c.SafeToApply, c.Reason, c.CreatedAt); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// SchemaMigrationRate returns 1.0 when all telemetry tables exist.
func (r *Repo) SchemaMigrationRate(ctx context.Context) float64 {
	if !r.useDB() {
		return 0
	}
	tables := []string{
		"agent_telemetry_sessions", "agent_recall_events", "agent_memory_decisions",
		"agent_output_events", "agent_obedience_evaluations", "agent_memory_use_violations",
		"agent_utility_candidates",
	}
	for _, t := range tables {
		var ok bool
		err := r.DB.QueryRowContext(ctx, `
SELECT EXISTS (
  SELECT 1 FROM information_schema.tables
  WHERE table_schema = 'public' AND table_name = $1
)`, t).Scan(&ok)
		if err != nil || !ok {
			return 0
		}
	}
	return 1
}

// FixedNow sets created_at for deterministic tests.
func FixedNow() time.Time {
	return time.Date(2026, 5, 25, 12, 0, 0, 0, time.UTC)
}
