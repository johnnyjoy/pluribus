package agenttelemetry

import (
	"context"
	"testing"

	"control-plane/internal/agentobedience"
	"control-plane/internal/recall"

	"github.com/google/uuid"
)

func TestAutoRecallFixtureCount(t *testing.T) {
	cases, err := LoadAutoRecallCases("")
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) < 28 {
		t.Fatalf("need >=28 auto recall cases, got %d", len(cases))
	}
}

func TestRecordAutoRecallPersistsAndReturnsCorrelation(t *testing.T) {
	svc := NewTestService()
	sid := uuid.New().String()
	_, err := svc.StartSession(context.Background(), StartSessionRequest{Interface: "rest", SessionID: sid})
	if err != nil {
		t.Fatal(err)
	}
	bundle := &recall.RecallBundle{
		GoverningConstraints: []recall.MemoryItem{{ID: "test:mem-1", Statement: "rule", QualityState: "verified"}},
	}
	tel, err := svc.RecordAutoRecall(context.Background(), recall.AutoRecallInput{
		SessionID:     sid,
		Interface:     "rest",
		RecallRequest: map[string]any{"retrieval_query": "audit"},
		Bundle:        bundle,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !tel.TelemetryEnabled || tel.RecallEventID == "" || tel.RecallBundleID == "" || tel.RecallRequestHash == "" {
		t.Fatalf("missing correlation: %+v", tel)
	}
	rid, err := uuid.Parse(tel.RecallEventID)
	if err != nil {
		t.Fatal(err)
	}
	ev, ok := svc.getRecall(context.Background(), rid)
	if !ok || len(ev.RecalledMemoryIDs) == 0 {
		t.Fatal("recall event not persisted")
	}
}

func TestAutoRecallIdempotentRequestHash(t *testing.T) {
	svc := NewTestService()
	sid := uuid.New().String()
	_, _ = svc.StartSession(context.Background(), StartSessionRequest{Interface: "rest", SessionID: sid})
	req := map[string]any{"retrieval_query": "same"}
	bundle := &recall.RecallBundle{Continuity: []recall.MemoryItem{{ID: "test:a", QualityState: "verified"}}}
	t1, err := svc.RecordAutoRecall(context.Background(), recall.AutoRecallInput{SessionID: sid, Interface: "rest", RecallRequest: req, Bundle: bundle})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := svc.RecordAutoRecall(context.Background(), recall.AutoRecallInput{SessionID: sid, Interface: "rest", RecallRequest: req, Bundle: bundle})
	if err != nil {
		t.Fatal(err)
	}
	if t1.RecallEventID != t2.RecallEventID {
		t.Fatalf("expected idempotent recall event: %s vs %s", t1.RecallEventID, t2.RecallEventID)
	}
}

func TestEvaluateRejectsUnknownAndWrongSessionRecall(t *testing.T) {
	svc := NewTestService()
	s1 := uuid.New().String()
	s2 := uuid.New().String()
	_, _ = svc.StartSession(context.Background(), StartSessionRequest{Interface: "rest", SessionID: s1})
	_, _ = svc.StartSession(context.Background(), StartSessionRequest{Interface: "rest", SessionID: s2})
	ev, err := svc.RecordAutoRecall(context.Background(), recall.AutoRecallInput{
		SessionID: s1, Interface: "rest",
		RecallRequest: map[string]any{"q": "x"},
		Bundle:        &recall.RecallBundle{Continuity: []recall.MemoryItem{{ID: "test:m", QualityState: "verified"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Evaluate(context.Background(), EvaluateRequest{SessionID: s1, RecallEventID: uuid.New().String()})
	if err != ErrUnknownRecall {
		t.Fatalf("unknown recall: got %v", err)
	}
	oc := agentobedience.ObedienceCase{
		ID: "x", TaskID: "t", Interface: "rest", AgentMode: agentobedience.AgentObedient,
		InputMemories: []agentobedience.CaseMemory{{MemoryID: "test:m", Statement: "s", QualityState: "verified"}},
	}
	bundle := agentobedience.BundleFromCase(oc)
	tel := agentobedience.RunScriptedAgent(oc, bundle, "rest")
	_, _ = svc.RecordDecision(context.Background(), RecordDecisionRequest{
		SessionID: s2, RecallEventID: ev.RecallEventID,
		Decisions: []struct {
			MemoryID             string   `json:"memory_id"`
			Decision             string   `json:"decision"`
			Reason               string   `json:"reason"`
			ContractFieldsCited  []string `json:"contract_fields_cited"`
			OutputFactsSupported []string `json:"output_facts_supported"`
		}{{MemoryID: tel.MemoryDecisions[0].MemoryID, Decision: tel.MemoryDecisions[0].Decision, Reason: "r", ContractFieldsCited: tel.MemoryDecisions[0].ContractFieldsCited}},
	})
	_, _ = svc.RecordOutput(context.Background(), RecordOutputRequest{
		SessionID: s2, RecallEventID: ev.RecallEventID, OutputFacts: tel.FinalOutput.Facts,
	})
	_, err = svc.Evaluate(context.Background(), EvaluateRequest{SessionID: s2, RecallEventID: ev.RecallEventID})
	if err != ErrWrongSessionRecall {
		t.Fatalf("wrong session: got %v", err)
	}
}

func TestTelemetryDisabledNoAutoRecord(t *testing.T) {
	opts := recall.TelemetryOptions{}
	if opts.TelemetryWanted() {
		t.Fatal("expected telemetry off without session")
	}
}

func TestHostileAutoRecallCasesMetadata(t *testing.T) {
	cases, err := LoadAutoRecallCases("")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, c := range cases {
		seen[c.CaseID] = true
	}
	required := []string{
		"rest_compile_auto_persists_recall_event", "evaluate_rejects_unknown_recall_event_id",
		"telemetry_disabled_preserves_backward_compatibility", "postgres_recall_event_roundtrip",
		"mcp_recall_context_auto_persists_recall_event", "duplicate_request_id_idempotent_or_documented",
	}
	for _, id := range required {
		if !seen[id] {
			t.Fatalf("missing case %s", id)
		}
	}
}
