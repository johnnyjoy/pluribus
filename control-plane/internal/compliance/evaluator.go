package compliance

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// EvaluateSession computes compliance from session events.
func EvaluateSession(events []Event, recallMaxAge time.Duration) Evaluation {
	if len(events) == 0 {
		return Evaluation{
			Status:       StatusUnknown,
			MissingSteps: []string{"insufficient_events"},
			Evidence:     map[string]any{"reason": "no telemetry events"},
			Warnings:     []string{"Pluribus only observes MCP calls; external edits are invisible."},
		}
	}

	windowStart := events[0].OccurredAt
	windowEnd := events[len(events)-1].OccurredAt
	sessionID := events[0].SessionID

	if IsDiagnosticOnlySession(events) {
		return Evaluation{
			SessionID:   sessionID,
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			Status:      StatusNotApplicable,
			MissingSteps: []string{},
			Evidence: map[string]any{
				"reason": "diagnostic_only_session",
			},
		}
	}

	var missing []string
	var warnings []string
	evidence := map[string]any{}

	hasRecall := false
	hasEnforcement := false
	hasRecord := false
	hasMaterial := false
	hasSubstantive := false

	zeroRecall, recallAt := lastRecallTime(events)
	for _, e := range events {
		if !isSuccessfulLoopEvent(e) {
			continue
		}
		tool := e.ToolName
		if tool == "" {
			continue
		}
		if IsSubstantiveActivity(tool) {
			hasSubstantive = true
		}
		if IsMaterialChange(tool) {
			hasMaterial = true
		}
		if e.EventType == EventRecallCalled || IsRecallTool(tool) {
			hasRecall = true
			evidence["recall_event_id"] = e.ID.String()
			evidence["recall_at"] = e.OccurredAt.Format(time.RFC3339Nano)
		}
		if e.EventType == EventEnforcementCalled || IsEnforcementTool(tool) {
			hasEnforcement = true
			evidence["enforcement_event_id"] = e.ID.String()
			evidence["enforcement_decision"] = e.EnforcementDecision
		}
		if e.EventType == EventRecordCalled || IsRecordTool(tool) {
			hasRecord = true
			evidence["record_event_id"] = e.ID.String()
		}
	}

	if !hasSubstantive && !hasMaterial {
		return Evaluation{
			SessionID:   sessionID,
			WindowStart: windowStart,
			WindowEnd:   windowEnd,
			Status:      StatusNotApplicable,
			MissingSteps: []string{},
			Evidence: map[string]any{
				"reason": "no_substantive_mcp_activity",
			},
			Warnings: warnings,
		}
	}

	if !hasRecall {
		missing = append(missing, "recall_before_work")
	} else if !zeroRecall && recallMaxAge > 0 {
		if mat, matAt := firstMaterialChangeAfter(events, recallAt); mat && matAt.Sub(recallAt) > recallMaxAge {
			missing = append(missing, "recall_stale_before_material_change")
			evidence["recall_staleness_ms"] = matAt.Sub(recallAt).Milliseconds()
		}
	}

	if hasMaterial && !hasEnforcement {
		missing = append(missing, "enforcement_before_material_change")
	}

	if hasSubstantive && !hasRecord {
		missing = append(missing, "record_after_outcome")
	}

	status := StatusCompliant
	if len(missing) > 0 {
		if hasRecall || hasEnforcement || hasRecord {
			status = StatusPartiallyCompliant
		} else {
			status = StatusNonCompliant
		}
		if len(missing) >= 2 && !hasRecall && !hasEnforcement && !hasRecord {
			status = StatusNonCompliant
		}
	}

	evidence["has_recall"] = hasRecall
	evidence["has_enforcement"] = hasEnforcement
	evidence["has_record"] = hasRecord
	evidence["has_material_change"] = hasMaterial
	evidence["event_count"] = len(events)

	return Evaluation{
		SessionID:    sessionID,
		WindowStart:  windowStart,
		WindowEnd:    windowEnd,
		Status:       status,
		MissingSteps: missing,
		Evidence:     evidence,
		Warnings:     append(warnings, "Compliance is based on MCP-visible telemetry only."),
	}
}

// HashRequest returns stable SHA256 hex of redacted request payload.
func HashRequest(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:8])
}

// SummarizeToolCall returns a redacted one-line summary for telemetry.
func SummarizeToolCall(toolName string, args json.RawMessage) string {
	var m map[string]any
	if len(args) > 0 {
		_ = json.Unmarshal(args, &m)
	}
	switch toolName {
	case "recall_context", "memory_context_resolve":
		return fmt.Sprintf("recall task=%q", truncate(firstArg(m, "task", "task_description", "query", "retrieval_query"), 80))
	case "record_experience", "mcp_episode_ingest":
		return fmt.Sprintf("record summary_len=%d", len([]rune(firstArg(m, "summary"))))
	case "enforcement_evaluate":
		return fmt.Sprintf("enforcement proposal_len=%d", len([]rune(firstArg(m, "proposal_text"))))
	default:
		return fmt.Sprintf("tool=%s keys=%d", toolName, len(m))
	}
}

func firstArg(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// EvaluatePersisted runs evaluation and stores result when repo is wired.
func (s *Service) EvaluatePersisted(ctx context.Context, sessionID uuid.UUID, recallMaxAge time.Duration) (Evaluation, error) {
	events, err := s.ListEvents(ctx, sessionID)
	if err != nil {
		return Evaluation{}, err
	}
	out := EvaluateSession(events, recallMaxAge)
	out.SessionID = sessionID
	out.EvaluatedAt = time.Now().UTC()
	if len(events) > 0 {
		out.WindowStart = events[0].OccurredAt
		out.WindowEnd = events[len(events)-1].OccurredAt
	}
	if s.Repo != nil && s.Repo.DB != nil {
		id, err := s.Repo.InsertEvaluation(ctx, out)
		if err != nil {
			return out, err
		}
		out.ID = id
	}
	return out, nil
}

func isSuccessfulLoopEvent(e Event) bool {
	if e.ResultStatus == "error" || e.EventType == EventMCPToolCallFailed {
		return false
	}
	switch e.EventType {
	case EventMCPToolCallCompleted, EventRecallCalled, EventEnforcementCalled,
		EventRecordCalled, EventMemoryMutation, EventCurationCalled, EventDiagnosticCalled:
		return true
	default:
		return false
	}
}
