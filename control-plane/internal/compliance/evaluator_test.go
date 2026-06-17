package compliance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func insertCompletedTool(t *testing.T, svc *Service, sid uuid.UUID, tool string, at time.Time) {
	t.Helper()
	e := Event{
		SessionID:    sid,
		OccurredAt:   at,
		EventType:    ClassifyTool(tool),
		ToolName:     tool,
		ResultStatus: "ok",
	}
	if err := svc.RecordEvent(context.Background(), e); err != nil {
		t.Fatal(err)
	}
}

func TestEvaluateSession_compliantSequence(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	base := time.Now().UTC().Add(-5 * time.Minute)
	_ = svc.EnsureSession(context.Background(), MCPContext{SessionID: sid, Transport: "test"})
	insertCompletedTool(t, svc, sid, "recall_context", base)
	insertCompletedTool(t, svc, sid, "enforcement_evaluate", base.Add(time.Minute))
	insertCompletedTool(t, svc, sid, "record_experience", base.Add(2*time.Minute))

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if ev.Status != StatusCompliant {
		t.Fatalf("status=%q missing=%v", ev.Status, ev.MissingSteps)
	}
}

func TestEvaluateSession_missingRecall(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	base := time.Now().UTC()
	insertCompletedTool(t, svc, sid, "memory_create", base)
	insertCompletedTool(t, svc, sid, "record_experience", base.Add(time.Minute))

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if ev.Status != StatusPartiallyCompliant && ev.Status != StatusNonCompliant {
		t.Fatalf("status=%q want partial or non compliant", ev.Status)
	}
	if !containsStep(ev.MissingSteps, "recall_before_work") {
		t.Fatalf("missing recall step: %v", ev.MissingSteps)
	}
}

func TestEvaluateSession_missingEnforcement(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	base := time.Now().UTC()
	insertCompletedTool(t, svc, sid, "recall_context", base)
	insertCompletedTool(t, svc, sid, "memory_create", base.Add(time.Minute))
	insertCompletedTool(t, svc, sid, "record_experience", base.Add(2*time.Minute))

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if !containsStep(ev.MissingSteps, "enforcement_before_material_change") {
		t.Fatalf("missing enforcement: %v", ev.MissingSteps)
	}
}

func TestEvaluateSession_missingRecord(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	base := time.Now().UTC()
	insertCompletedTool(t, svc, sid, "recall_context", base)
	insertCompletedTool(t, svc, sid, "enforcement_evaluate", base.Add(time.Minute))
	insertCompletedTool(t, svc, sid, "memory_create", base.Add(2*time.Minute))

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if !containsStep(ev.MissingSteps, "record_after_outcome") {
		t.Fatalf("missing record: %v", ev.MissingSteps)
	}
}

func TestEvaluateSession_diagnosticOnly(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	base := time.Now().UTC()
	_ = svc.RecordEvent(context.Background(), Event{
		SessionID: sid, OccurredAt: base, EventType: EventMCPInitialize, ResultStatus: "ok",
	})
	_ = svc.RecordEvent(context.Background(), Event{
		SessionID: sid, OccurredAt: base.Add(time.Second), EventType: EventMCPToolsList, ResultStatus: "ok",
	})
	insertCompletedTool(t, svc, sid, "health", base.Add(2*time.Second))
	insertCompletedTool(t, svc, sid, "curation_pending", base.Add(3*time.Second))

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if ev.Status != StatusNotApplicable {
		t.Fatalf("status=%q want not_applicable", ev.Status)
	}
}

func TestEvaluateSession_unknownNoEvents(t *testing.T) {
	ev := EvaluateSession(nil, DefaultRecallMaxAge)
	if ev.Status != StatusUnknown {
		t.Fatalf("status=%q", ev.Status)
	}
}

func TestEvaluateSession_failedCallNotCompliant(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	_ = svc.RecordEvent(context.Background(), Event{
		SessionID: sid, EventType: EventMCPToolCallFailed, ToolName: "recall_context",
		ResultStatus: "error", ErrorCode: "-32602",
	})
	insertCompletedTool(t, svc, sid, "memory_create", time.Now().UTC())

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if ev.Status == StatusCompliant {
		t.Fatal("failed recall must not yield compliant")
	}
	if !containsStep(ev.MissingSteps, "recall_before_work") {
		t.Fatalf("expected missing recall: %v", ev.MissingSteps)
	}
}

func TestEvaluateSession_recordWithoutRecall_hostile(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	insertCompletedTool(t, svc, sid, "record_experience", time.Now().UTC())
	insertCompletedTool(t, svc, sid, "memory_create", time.Now().UTC())

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if containsStep(ev.MissingSteps, "recall_before_work") == false {
		t.Fatalf("expected missing recall: %v", ev.MissingSteps)
	}
}

func TestEvaluateSession_enforcementWithoutRecall_hostile(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	insertCompletedTool(t, svc, sid, "enforcement_evaluate", time.Now().UTC())
	insertCompletedTool(t, svc, sid, "memory_create", time.Now().UTC())

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if !containsStep(ev.MissingSteps, "recall_before_work") {
		t.Fatalf("expected missing recall: %v", ev.MissingSteps)
	}
}

func TestEvaluateSession_curationWithoutEnforcement_hostile(t *testing.T) {
	svc := NewService(nil)
	sid := uuid.New()
	insertCompletedTool(t, svc, sid, "recall_context", time.Now().UTC())
	insertCompletedTool(t, svc, sid, "curation_materialize", time.Now().UTC())

	events, _ := svc.ListEvents(context.Background(), sid)
	ev := EvaluateSession(events, DefaultRecallMaxAge)
	if !containsStep(ev.MissingSteps, "enforcement_before_material_change") {
		t.Fatalf("expected missing enforcement: %v", ev.MissingSteps)
	}
}

func containsStep(steps []string, want string) bool {
	for _, s := range steps {
		if s == want {
			return true
		}
	}
	return false
}
