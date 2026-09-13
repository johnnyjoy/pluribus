package mcp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"control-plane/internal/formation"
)

func TestParseUsedMemoryIDs_capAndInvalid(t *testing.T) {
	args := `{
		"summary":"x",
		"used_memory_ids":[
			"11111111-1111-4111-8111-111111111111",
			"not-a-uuid",
			"11111111-1111-4111-8111-111111111111",
			"22222222-2222-4222-8222-222222222222",
			"33333333-3333-4333-8333-333333333333",
			"44444444-4444-4444-8444-444444444444",
			"55555555-5555-4555-8555-555555555555",
			"66666666-6666-4666-8666-666666666666",
			"77777777-7777-4777-8777-777777777777",
			"88888888-8888-4888-8888-888888888888",
			"99999999-9999-4999-8999-999999999999"
		]
	}`
	valid, skipped := parseUsedMemoryIDs(json.RawMessage(args))
	if len(valid) != usedMemoryIDsCap {
		t.Fatalf("valid=%d want %d: %v", len(valid), usedMemoryIDsCap, valid)
	}
	if len(skipped) < 2 {
		t.Fatalf("expected invalid + overflow in skipped, got %v", skipped)
	}
	foundBad := false
	for _, s := range skipped {
		if s == "not-a-uuid" {
			foundBad = true
		}
	}
	if !foundBad {
		t.Fatalf("skipped missing invalid uuid: %v", skipped)
	}
}

func TestParseUsedMemoryIDs_empty(t *testing.T) {
	valid, skipped := parseUsedMemoryIDs(json.RawMessage(`{"summary":"x"}`))
	if len(valid) != 0 || len(skipped) != 0 {
		t.Fatalf("valid=%v skipped=%v", valid, skipped)
	}
}

func TestRecordExperience_usedMemoryIDsAppliesHelpful(t *testing.T) {
	var helpfulHits atomic.Int32
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/advisory-episodes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"00000000-0000-4000-8000-000000000001","summary_text":"ok","source":"mcp"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/feedback") && r.Method == http.MethodPost {
			helpfulHits.Add(1)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"event_type":"helpful"}`))
			return
		}
		http.NotFound(w, r)
	})
	h := NewHTTPHandler(inner, DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"record_experience","arguments":{"summary":"Fixed MCP schema validation for recall and record tools in phase 1.","used_memory_ids":["11111111-1111-4111-8111-111111111111","00000000-0000-0000-0000-000000000000"]}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	if helpfulHits.Load() != 2 {
		t.Fatalf("helpful posts=%d want 2", helpfulHits.Load())
	}
	result, _ := out["result"].(map[string]any)
	content, _ := result["content"].([]any)
	c0, _ := content[0].(map[string]any)
	text, _ := c0["text"].(string)
	if !strings.Contains(text, "utility_feedback") || !strings.Contains(text, "11111111-1111-4111-8111-111111111111") {
		t.Fatalf("expected utility_feedback in record text: %s", text)
	}
}

func TestRecordExperience_omittedUsedIDsSkipsFeedback(t *testing.T) {
	var helpfulHits atomic.Int32
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/advisory-episodes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"00000000-0000-4000-8000-000000000001","summary_text":"ok","source":"mcp"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/feedback") {
			helpfulHits.Add(1)
		}
		http.NotFound(w, r)
	})
	h := NewHTTPHandler(inner, DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"record_experience","arguments":{"summary":"Fixed MCP schema validation for recall and record tools in phase 1."}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	if helpfulHits.Load() != 0 {
		t.Fatalf("helpful posts=%d want 0", helpfulHits.Load())
	}
}

func TestRecordExperience_feedback404DoesNotFailRecord(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/advisory-episodes" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"00000000-0000-4000-8000-000000000001","summary_text":"ok","source":"mcp"}`))
			return
		}
		if strings.HasSuffix(r.URL.Path, "/feedback") {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error":"memory not found"}`))
			return
		}
		http.NotFound(w, r)
	})
	h := NewHTTPHandler(inner, DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptest.NewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"record_experience","arguments":{"summary":"Fixed MCP schema validation for recall and record tools in phase 1.","used_memory_ids":["00000000-0000-0000-0000-000000000000"]}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	result, _ := out["result"].(map[string]any)
	if result["isError"] == true {
		t.Fatalf("record must succeed when feedback 404s: %+v", result)
	}
	content, _ := result["content"].([]any)
	c0, _ := content[0].(map[string]any)
	text, _ := c0["text"].(string)
	if !strings.Contains(text, "utility_feedback") || !strings.Contains(text, "errors") {
		t.Fatalf("expected utility_feedback errors: %s", text)
	}
}
