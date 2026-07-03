package mcp

import (
	"control-plane/internal/formation"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func mcpStubRouter() http.Handler {
	return MCPFullStubRouter()
}

func postMCP(t *testing.T, srvURL, body string) map[string]any {
	t.Helper()
	res, err := http.Post(srvURL, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("HTTP %d", res.StatusCode)
	}
	var out map[string]any
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestMCPBehavior_initialize(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"pluribus-phase1-proof","version":"0.1.0"}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	result, _ := out["result"].(map[string]any)
	if result["serverInfo"] == nil {
		t.Fatalf("missing serverInfo: %+v", result)
	}
	inst, _ := result["instructions"].(string)
	if !strings.Contains(inst, "recall_context") || !strings.Contains(inst, "record_experience") || !strings.Contains(inst, "resolve_chore") {
		t.Fatalf("instructions missing loop tools: %q", inst)
	}
}

func TestMCPBehavior_toolsListSchemasNonEmpty(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	out := postMCP(t, srv.URL, `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	result, _ := out["result"].(map[string]any)
	tools, _ := result["tools"].([]any)
	if len(tools) != len(registryForList()) {
		t.Fatalf("tools count %d want %d", len(tools), len(registryForList()))
	}
	for _, raw := range tools {
		tool, _ := raw.(map[string]any)
		name, _ := tool["name"].(string)
		schema, _ := tool["inputSchema"].(map[string]any)
		if schema == nil {
			t.Fatalf("tool %s nil schema", name)
		}
		if schema["type"] != "object" {
			t.Fatalf("tool %s schema type %v", name, schema["type"])
		}
		if _, ok := schema["additionalProperties"]; !ok {
			t.Fatalf("tool %s missing additionalProperties", name)
		}
	}
}

func TestMCPBehavior_toolsCallRecall(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"recall_context","arguments":{"task":"audit MCP behavior"}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	result, _ := out["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("empty content")
	}
	c0, _ := content[0].(map[string]any)
	text, _ := c0["text"].(string)
	if !strings.Contains(text, "recall_bundle") && !strings.Contains(text, "mcp_context") {
		preview := text
		if len(preview) > 200 {
			preview = preview[:200]
		}
		t.Fatalf("expected recall wrapper, got %q", preview)
	}
}

func TestMCPBehavior_toolsCallRecord(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"record_experience","arguments":{"summary":"Fixed MCP schema validation for recall and record tools in phase 1."}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
}

func TestMCPBehavior_toolsCallEnforcement(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"enforcement_evaluate","arguments":{"proposal_text":"We will migrate the database to SQLite for simplicity."}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
}

func TestMCPBehavior_toolsCallCurationPending(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"curation_pending","arguments":{}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
}

func TestMCPBehavior_unknownTool(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"does_not_exist","arguments":{}}}`
	out := postMCP(t, srv.URL, body)
	errObj, _ := out["error"].(map[string]any)
	if errObj == nil {
		t.Fatalf("expected JSON-RPC error, got %+v", out)
	}
	if code, _ := errObj["code"].(float64); code != -32602 {
		t.Fatalf("code %v want -32602", errObj["code"])
	}
}

func TestMCPBehavior_missingRequiredArgument(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"recall_context","arguments":{}}}`
	out := postMCP(t, srv.URL, body)
	errObj, _ := out["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error")
	}
	msg, _ := errObj["message"].(string)
	if !strings.Contains(msg, "task") {
		t.Fatalf("message %q", msg)
	}
}

func TestMCPBehavior_extraArgumentTolerated(t *testing.T) {
	// Unknown arguments are tolerated (dropped at forwarding), never rejected:
	// MCP clients attach metadata like agent_id/repo_root to every call (H2).
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"enforcement_evaluate","arguments":{"proposal_text":"x","extra":1,"agent_id":"cursor:test"}}}`
	out := postMCP(t, srv.URL, body)
	if errObj, _ := out["error"].(map[string]any); errObj != nil {
		t.Fatalf("extra properties must be tolerated, got error %v", errObj)
	}
}

func TestMCPBehavior_unknownMethod(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	out := postMCP(t, srv.URL, `{"jsonrpc":"2.0","id":9,"method":"nope/method","params":{}}`)
	errObj, _ := out["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error")
	}
	if code, _ := errObj["code"].(float64); code != -32601 {
		t.Fatalf("code %v", errObj["code"])
	}
}

func TestMCPBehavior_malformedJSON(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	res, err := http.Post(srv.URL, "application/json", strings.NewReader(`{not json`))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(res.Body).Decode(&out)
	errObj, _ := out["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected parse error")
	}
}

func TestMCPBehavior_wrongArgumentType(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"recall_context","arguments":{"task":12345}}}`
	out := postMCP(t, srv.URL, body)
	errObj, _ := out["error"].(map[string]any)
	if errObj == nil {
		t.Fatal("expected error for wrong type")
	}
	if code, _ := errObj["code"].(float64); code != -32602 {
		t.Fatalf("code %v want -32602", errObj["code"])
	}
}

func httptestNewServer(h http.Handler) *httptest.Server {
	return httptest.NewServer(h)
}
