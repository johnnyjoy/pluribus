package mcp

import (
	"control-plane/internal/formation"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestCallCoverage_everyToolClassified(t *testing.T) {
	if issues := CallCoverageIssues(); len(issues) > 0 {
		t.Fatalf("call coverage classification issues:\n- %s", strings.Join(issues, "\n- "))
	}
	if len(toolRegistry()) != 59 {
		t.Fatalf("expected 59 tools, got %d", len(toolRegistry()))
	}
}

func TestAllRegisteredTools_toolsCall(t *testing.T) {
	h := NewHTTPHandler(MCPFullStubRouter(), DefaultMemoryFormationPolicy(), nil, formation.NewGate(nil))
	srv := httptestNewServer(h)
	defer srv.Close()

	for _, spec := range toolRegistry() {
		t.Run(spec.Name, func(t *testing.T) {
			args, err := MinValidToolCallArguments(spec.Name)
			if err != nil {
				t.Fatal(err)
			}
			argsJSON, err := json.Marshal(args)
			if err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":%q,"arguments":%s}}`,
				spec.Name, string(argsJSON))
			out := postMCP(t, srv.URL, body)
			if out["error"] != nil {
				t.Fatalf("tools/call %s JSON-RPC error: %v", spec.Name, out["error"])
			}
			result, ok := out["result"].(map[string]any)
			if !ok || result == nil {
				t.Fatalf("tools/call %s missing result: %+v", spec.Name, out)
			}
			if isErr, _ := result["isError"].(bool); isErr {
				content, _ := result["content"].([]any)
				msg := ""
				if len(content) > 0 {
					if c0, ok := content[0].(map[string]any); ok {
						msg, _ = c0["text"].(string)
					}
				}
				t.Fatalf("tools/call %s returned isError: %s", spec.Name, msg)
			}
		})
	}
}
