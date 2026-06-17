package agenttelemetry

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"control-plane/internal/agentobedience"
)

func TestHTTPDecisionRoundTrip(t *testing.T) {
	cases, _ := LoadPersistenceCases("")
	c := cases[0]
	oc := c.ToObedienceCase()
	b := agentobedience.BundleFromCase(oc)
	tel := agentobedience.RunScriptedAgent(oc, b, "rest")

	svc := NewTestService()
	srv := httptest.NewServer(newTelemetryRouter(svc))
	defer srv.Close()

	sessBody, _ := json.Marshal(map[string]any{"interface": "rest"})
	sessResp, _ := postJSON(srv.URL+"/v1/agent/telemetry/session/start", "", sessBody)
	sid := sessResp["session_id"].(string)
	bj := recallBundleJSONFromCase(oc)
	recBody, _ := json.Marshal(map[string]any{
		"session_id": sid, "task_id": c.TaskID, "recalled_memory_ids": recalledIDsFromBundle(b), "recall_bundle": bj,
	})
	recResp, _ := postJSON(srv.URL+"/v1/agent/telemetry/recall", "", recBody)
	rid := recResp["recall_event_id"].(string)

	var decs []map[string]any
	for _, d := range tel.MemoryDecisions {
		decs = append(decs, map[string]any{
			"memory_id": d.MemoryID, "decision": d.Decision, "reason": d.Reason,
			"contract_fields_cited": d.ContractFieldsCited, "output_facts_supported": d.OutputFactsSupported,
		})
	}
	decRaw, _ := json.Marshal(map[string]any{"session_id": sid, "recall_event_id": rid, "decisions": decs})
	_, err := postJSON(srv.URL+"/v1/agent/telemetry/decision", "", decRaw)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	suid, _ := parseUUID(sid)
	ruuid, _ := parseUUID(rid)
	stored := svc.mem.listDecisionsByRecall(ctx, ruuid)
	t.Logf("stored decisions=%+v", stored)

	rec, _ := svc.mem.getRecall(ctx, ruuid)
	out, _ := svc.RecordOutput(ctx, RecordOutputRequest{
		SessionID: suid.String(), TaskID: c.TaskID, RecallEventID: rid,
		OutputFacts: tel.FinalOutput.Facts, MemoryCitations: tel.FinalOutput.MemoryCitations,
	})
	sess, _ := svc.mem.getSession(ctx, suid)
	tel2 := buildTelemetryFromPersisted(*sess, *rec, stored, out)
	tel2.TelemetryComplete = true
	tel2.RubricResult = agentobedience.EvaluateRubric(tel2.FinalOutput.Facts, c.ExpectedOutputFacts, nil)
	_ = tel2
	resp, err := svc.Evaluate(ctx, EvaluateRequest{
		SessionID: sid, TaskID: c.TaskID, RecallEventID: rid, OutputID: out.ID.String(),
		ExpectedFacts: c.ExpectedOutputFacts, TaskTags: c.TaskTags,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("svc.Evaluate passed=%v violations=%v", resp.Evaluation.ObediencePassed, resp.Evaluation.Violations)
}
