package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPMemoryFeedbackHelpful(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, nil)
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_feedback","arguments":{"memory_id":"11111111-1111-4111-8111-111111111111","event_type":"helpful","reason":"guided correct tool"}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
	result, _ := out["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatal("empty content")
	}
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	if !strings.Contains(text, "new_utility_score") && !strings.Contains(text, "utility") {
		t.Fatalf("expected utility response: %s", text)
	}
}

func TestMCPMemoryFeedbackWrong(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, nil)
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_feedback","arguments":{"memory_id":"11111111-1111-4111-8111-111111111111","event_type":"wrong","reason":"factually incorrect guidance"}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] != nil {
		t.Fatalf("error: %v", out["error"])
	}
}

func TestMCPMemoryFeedbackInvalidTypeRejected(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, nil)
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_feedback","arguments":{"memory_id":"11111111-1111-4111-8111-111111111111","event_type":"bogus"}}}`
	out := postMCP(t, srv.URL, body)
	if out["error"] == nil {
		t.Fatal("expected JSON-RPC error for invalid event_type enum")
	}
}

func TestMCPMemoryFeedbackMissingMemoryRejected(t *testing.T) {
	h := NewHTTPHandler(mcpStubRouter(), DefaultMemoryFormationPolicy(), nil, nil)
	srv := httptestNewServer(h)
	defer srv.Close()

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"memory_feedback","arguments":{"memory_id":"00000000-0000-0000-0000-000000000000","event_type":"helpful"}}}`
	out := postMCP(t, srv.URL, body)
	result, _ := out["result"].(map[string]any)
	content, _ := result["content"].([]any)
	block, _ := content[0].(map[string]any)
	text, _ := block["text"].(string)
	if !strings.Contains(strings.ToLower(text), "not found") && !strings.Contains(strings.ToLower(text), "error") {
		t.Fatalf("expected not found: %s", text)
	}
}

func TestMCPMemoryFeedbackSchema(t *testing.T) {
	for _, tool := range toolRegistry() {
		if tool.Name != "memory_feedback" {
			continue
		}
		schema := tool.InputSchema
		if schema["type"] != "object" {
			t.Fatalf("schema type")
		}
		props, _ := schema["properties"].(map[string]any)
		for _, req := range []string{"memory_id", "event_type"} {
			if props[req] == nil {
				t.Fatalf("missing property %s", req)
			}
		}
		required, _ := schema["required"].([]string)
		if len(required) < 2 {
			// JSON decode may use []any in other paths; accept either.
			if reqAny, ok := schema["required"].([]any); !ok || len(reqAny) < 2 {
				t.Fatalf("required fields: %v", schema["required"])
			}
		}
		return
	}
	t.Fatal("memory_feedback not in registry")
}

func TestMCPMemoryFeedbackWrongRequiresReason(t *testing.T) {
	raw := map[string]any{"memory_id": "11111111-1111-4111-8111-111111111111", "event_type": "wrong"}
	b, _ := json.Marshal(raw)
	if err := ValidateToolArguments("memory_feedback", b); err == nil {
		t.Fatal("wrong without reason should fail validation")
	}
}
