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

// TestProofAgentLoopScenarios exercises MCP-visible loop compliance end-to-end (in-process).
func TestProofAgentLoopScenarios(t *testing.T) {
	scenarios := []struct {
		name       string
		calls      []string
		wantStatus string
	}{
		{
			name:       "compliant_sequence",
			calls:      []string{"recall_context", "enforcement_evaluate", "record_experience"},
			wantStatus: compliance.StatusCompliant,
		},
		{
			name:       "missing_recall",
			calls:      []string{"record_experience", "memory_create"},
			wantStatus: compliance.StatusPartiallyCompliant,
		},
		{
			name:       "missing_enforcement",
			calls:      []string{"recall_context", "memory_create", "record_experience"},
			wantStatus: compliance.StatusPartiallyCompliant,
		},
		{
			name:       "missing_record",
			calls:      []string{"recall_context", "enforcement_evaluate", "memory_create"},
			wantStatus: compliance.StatusPartiallyCompliant,
		},
		{
			name:       "diagnostic_only",
			calls:      []string{"health", "curation_pending"},
			wantStatus: compliance.StatusNotApplicable,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			svc := compliance.NewService(nil)
			h := NewHTTPHandler(MCPFullStubRouter(), DefaultMemoryFormationPolicy(), svc, formation.NewGate(nil))
			sid := proofInitialize(t, h)
			for _, tool := range sc.calls {
				proofToolCall(t, h, sid, tool)
			}
			events, _ := svc.ListEvents(context.Background(), sid)
			ev := compliance.EvaluateSession(events, compliance.DefaultRecallMaxAge)
			if ev.Status != sc.wantStatus {
				t.Fatalf("status=%q want %q missing=%v events=%d", ev.Status, sc.wantStatus, ev.MissingSteps, len(events))
			}
		})
	}

	t.Run("malformed_call_logged_not_compliant", func(t *testing.T) {
		svc := compliance.NewService(nil)
		h := NewHTTPHandler(MCPFullStubRouter(), DefaultMemoryFormationPolicy(), svc, formation.NewGate(nil))
		sid := uuid.New()
		body := map[string]any{
			"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": "recall_context", "arguments": map[string]any{}},
		}
		b, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewReader(b))
		req.Header.Set(compliance.HeaderSessionID, sid.String())
		h.ServeHTTP(rec, req)
		var resp map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &resp)
		if resp["error"] == nil {
			t.Fatal("expected JSON-RPC error for malformed recall")
		}
		events, _ := svc.ListEvents(context.Background(), sid)
		if len(events) == 0 {
			t.Fatal("expected failure telemetry")
		}
		ev := compliance.EvaluateSession(events, compliance.DefaultRecallMaxAge)
		if ev.Status == compliance.StatusCompliant {
			t.Fatal("malformed call must not be compliant")
		}
	})
}

func proofInitialize(t *testing.T, h http.Handler) uuid.UUID {
	t.Helper()
	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"proof","version":"1"}}}`
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", strings.NewReader(body))
	h.ServeHTTP(rec, req)
	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	res, _ := resp["result"].(map[string]any)
	pl, _ := res["pluribus"].(map[string]any)
	sidStr, _ := pl["session_id"].(string)
	sid, err := uuid.Parse(sidStr)
	if err != nil {
		t.Fatalf("session id: %v body=%s", err, rec.Body.String())
	}
	return sid
}

func proofToolCall(t *testing.T, h http.Handler, sid uuid.UUID, tool string) {
	t.Helper()
	args, err := MinValidToolCallArguments(tool)
	if err != nil {
		t.Fatalf("tool %s: %v", tool, err)
	}
	body := map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": tool, "arguments": args},
	}
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/mcp", bytes.NewReader(b))
	req.Header.Set(compliance.HeaderSessionID, sid.String())
	h.ServeHTTP(rec, req)
	var resp map[string]any
	_ = json.Unmarshal(rec.Body.Bytes(), &resp)
	if resp["error"] != nil {
		t.Fatalf("tool %s error: %v", tool, resp["error"])
	}
}
