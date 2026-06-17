package mcp

import (
	"control-plane/internal/formation"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"control-plane/internal/compliance"

	"github.com/google/uuid"
)

func TestMCPTelemetry_recordsLoopEvents(t *testing.T) {
	inner := MCPFullStubRouter()
	svc := compliance.NewService(nil)
	h := NewHTTPHandler(inner, DefaultMemoryFormationPolicy(), svc, formation.NewGate(nil))

	initBody := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", strings.NewReader(initBody))
	h.ServeHTTP(rec, req)
	var initResp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &initResp); err != nil {
		t.Fatal(err)
	}
	res, _ := initResp["result"].(map[string]any)
	pluribus, _ := res["pluribus"].(map[string]any)
	sidStr, _ := pluribus["session_id"].(string)
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		t.Fatalf("session id: %v", err)
	}

	callBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "recall_context",
			"arguments": map[string]any{"task": "telemetry test"},
		},
	}
	b, _ := json.Marshal(callBody)
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewReader(b))
	req2.Header.Set(compliance.HeaderSessionID, sidStr)
	h.ServeHTTP(rec2, req2)

	events, err := svc.ListEvents(context.Background(), sid)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) < 3 {
		t.Fatalf("expected initialize + recall events, got %d", len(events))
	}
	hasRecall := false
	for _, e := range events {
		if e.EventType == compliance.EventRecallCalled || (e.ToolName == "recall_context" && e.ResultStatus == "ok") {
			hasRecall = true
		}
	}
	if !hasRecall {
		t.Fatalf("no recall telemetry: %+v", events)
	}

	ev := compliance.EvaluateSession(events, compliance.DefaultRecallMaxAge)
	if ev.Status == compliance.StatusCompliant {
		// recall only — still missing enforcement/record for full loop
		t.Log("partial session as expected")
	}
}

func TestMCPTelemetry_malformedCallLoggedAsFailure(t *testing.T) {
	inner := MCPFullStubRouter()
	svc := compliance.NewService(nil)
	h := NewHTTPHandler(inner, DefaultMemoryFormationPolicy(), svc, formation.NewGate(nil))

	sid := uuid.New()
	callBody := map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "recall_context",
			"arguments": map[string]any{},
		},
	}
	b, _ := json.Marshal(callBody)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewReader(b))
	req.Header.Set(compliance.HeaderSessionID, sid.String())
	h.ServeHTTP(rec, req)

	events, _ := svc.ListEvents(context.Background(), sid)
	hasFailed := false
	for _, e := range events {
		if e.EventType == compliance.EventMCPToolCallFailed {
			hasFailed = true
		}
	}
	if !hasFailed {
		t.Fatalf("expected failure event, got %+v", events)
	}
}
