package utility

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Repo persists utility events and scores.
type Repo struct {
	DB *sql.DB
}

// MemoryExists returns true when memories row exists.
func (r *Repo) MemoryExists(ctx context.Context, memoryID uuid.UUID) (bool, error) {
	if r == nil || r.DB == nil {
		return false, ErrNoRepo
	}
	var n int
	err := r.DB.QueryRowContext(ctx, `SELECT 1 FROM memories WHERE id = $1`, memoryID).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// InsertEvent appends one utility event.
func (r *Repo) InsertEvent(ctx context.Context, e Event) (*Event, error) {
	if r == nil || r.DB == nil {
		return nil, ErrNoRepo
	}
	payload := e.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, ErrInvalidPayload
	}
	id := e.ID
	if id == uuid.Nil {
		id = uuid.New()
	}
	var out Event
	var payloadScan []byte
	err = r.DB.QueryRowContext(ctx,
		`INSERT INTO memory_utility_events (
			id, memory_id, event_type, event_weight, source, source_tool, source_session_id,
			correlation_id, recall_bundle_id, agent_loop_event_id, reason, evidence_id, payload
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13::jsonb)
		RETURNING id, memory_id, event_type, event_weight, source, COALESCE(source_tool,''),
			COALESCE(source_session_id,''), COALESCE(correlation_id,''), COALESCE(recall_bundle_id,''),
			agent_loop_event_id, reason, evidence_id, payload, created_at`,
		id, e.MemoryID, e.EventType, e.EventWeight, nullStr(e.Source), nullStr(e.SourceTool),
		nullStr(e.SourceSessionID), nullStr(e.CorrelationID), nullStr(e.RecallBundleID),
		e.AgentLoopEventID, e.Reason, e.EvidenceID, b,
	).Scan(
		&out.ID, &out.MemoryID, &out.EventType, &out.EventWeight, &out.Source, &out.SourceTool,
		&out.SourceSessionID, &out.CorrelationID, &out.RecallBundleID, &out.AgentLoopEventID,
		&out.Reason, &out.EvidenceID, &payloadScan, &out.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	_ = json.Unmarshal(payloadScan, &out.Payload)
	return &out, nil
}

// UpsertScore creates or updates aggregate score row.
func (r *Repo) UpsertScore(ctx context.Context, s Score) (*Score, error) {
	if r == nil || r.DB == nil {
		return nil, ErrNoRepo
	}
	s.UtilityScore = ClampUtilityScore(s.UtilityScore)
	var out Score
	err := r.DB.QueryRowContext(ctx,
		`INSERT INTO memory_utility_scores (
			memory_id, utility_score, helpful_count, harmful_count, wrong_count, outdated_count,
			irrelevant_count, confirmed_count, refuted_count, last_positive_at, last_negative_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,now())
		ON CONFLICT (memory_id) DO UPDATE SET
			utility_score = EXCLUDED.utility_score,
			helpful_count = EXCLUDED.helpful_count,
			harmful_count = EXCLUDED.harmful_count,
			wrong_count = EXCLUDED.wrong_count,
			outdated_count = EXCLUDED.outdated_count,
			irrelevant_count = EXCLUDED.irrelevant_count,
			confirmed_count = EXCLUDED.confirmed_count,
			refuted_count = EXCLUDED.refuted_count,
			last_positive_at = COALESCE(EXCLUDED.last_positive_at, memory_utility_scores.last_positive_at),
			last_negative_at = COALESCE(EXCLUDED.last_negative_at, memory_utility_scores.last_negative_at),
			updated_at = now()
		RETURNING memory_id, utility_score, helpful_count, harmful_count, wrong_count, outdated_count,
			irrelevant_count, confirmed_count, refuted_count, last_positive_at, last_negative_at, updated_at`,
		s.MemoryID, s.UtilityScore, s.HelpfulCount, s.HarmfulCount, s.WrongCount, s.OutdatedCount,
		s.IrrelevantCount, s.ConfirmedCount, s.RefutedCount, s.LastPositiveAt, s.LastNegativeAt,
	).Scan(
		&out.MemoryID, &out.UtilityScore, &out.HelpfulCount, &out.HarmfulCount, &out.WrongCount,
		&out.OutdatedCount, &out.IrrelevantCount, &out.ConfirmedCount, &out.RefutedCount,
		&out.LastPositiveAt, &out.LastNegativeAt, &out.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// GetScore returns score row or zero defaults if missing.
func (r *Repo) GetScore(ctx context.Context, memoryID uuid.UUID) (*Score, error) {
	if r == nil || r.DB == nil {
		return nil, ErrNoRepo
	}
	var out Score
	err := r.DB.QueryRowContext(ctx,
		`SELECT memory_id, utility_score, helpful_count, harmful_count, wrong_count, outdated_count,
			irrelevant_count, confirmed_count, refuted_count, last_positive_at, last_negative_at, updated_at
		 FROM memory_utility_scores WHERE memory_id = $1`,
		memoryID,
	).Scan(
		&out.MemoryID, &out.UtilityScore, &out.HelpfulCount, &out.HarmfulCount, &out.WrongCount,
		&out.OutdatedCount, &out.IrrelevantCount, &out.ConfirmedCount, &out.RefutedCount,
		&out.LastPositiveAt, &out.LastNegativeAt, &out.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return &Score{MemoryID: memoryID, UtilityScore: 0, UpdatedAt: time.Now().UTC()}, nil
	}
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// ListEvents returns events for a memory newest first.
func (r *Repo) ListEvents(ctx context.Context, memoryID uuid.UUID, limit int) ([]Event, error) {
	if r == nil || r.DB == nil {
		return nil, ErrNoRepo
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := r.DB.QueryContext(ctx,
		`SELECT id, memory_id, event_type, event_weight, source, COALESCE(source_tool,''),
			COALESCE(source_session_id,''), COALESCE(correlation_id,''), COALESCE(recall_bundle_id,''),
			agent_loop_event_id, reason, evidence_id, payload, created_at
		 FROM memory_utility_events WHERE memory_id = $1 ORDER BY created_at DESC LIMIT $2`,
		memoryID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Event
	for rows.Next() {
		var e Event
		var payloadScan []byte
		if err := rows.Scan(
			&e.ID, &e.MemoryID, &e.EventType, &e.EventWeight, &e.Source, &e.SourceTool,
			&e.SourceSessionID, &e.CorrelationID, &e.RecallBundleID, &e.AgentLoopEventID,
			&e.Reason, &e.EvidenceID, &payloadScan, &e.CreatedAt,
		); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(payloadScan, &e.Payload)
		list = append(list, e)
	}
	return list, rows.Err()
}

// GetScoresForMemories returns utility_score map for ids (missing → 0).
func (r *Repo) GetScoresForMemories(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]float64, error) {
	out := make(map[uuid.UUID]float64, len(ids))
	if r == nil || r.DB == nil || len(ids) == 0 {
		return out, nil
	}
	for _, id := range ids {
		out[id] = 0
	}
	// Simple per-id query for portability with sqlmock tests; batch IN for production ok too.
	for _, id := range ids {
		sc, err := r.GetScore(ctx, id)
		if err != nil {
			return nil, err
		}
		if sc != nil {
			out[id] = sc.UtilityScore
		}
	}
	return out, nil
}

// GetSummariesForMemories returns full utility score rows for lifecycle labeling.
func (r *Repo) GetSummariesForMemories(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Score, error) {
	out := make(map[uuid.UUID]Score, len(ids))
	if r == nil || r.DB == nil || len(ids) == 0 {
		return out, nil
	}
	for _, id := range ids {
		sc, err := r.GetScore(ctx, id)
		if err != nil {
			return nil, err
		}
		if sc != nil {
			out[id] = *sc
		}
	}
	return out, nil
}

// CountEventsByType returns count of events of given type for memory.
func (r *Repo) CountEventsByType(ctx context.Context, memoryID uuid.UUID, eventType string) (int, error) {
	if r == nil || r.DB == nil {
		return 0, ErrNoRepo
	}
	var n int
	err := r.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM memory_utility_events WHERE memory_id = $1 AND event_type = $2`,
		memoryID, eventType,
	).Scan(&n)
	return n, err
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
