package mcp

import (
	"encoding/json"
	"fmt"
	"testing"

	"control-plane/internal/formation"
)

// Phase 4 MCP spec-compliance gate (hostile audit finding M1): the MCP
// specification only defines content block types text, image, audio,
// resource, and resource_link. The old server emitted "type":"json" blocks,
// which broke spec-strict clients (Cursor). This test calls every registered
// tool and asserts no result ever carries a non-spec content type again.
var mcpSpecContentTypes = map[string]bool{
	"text":          true,
	"image":         true,
	"audio":         true,
	"resource":      true,
	"resource_link": true,
}

func TestMCPSpecCompliance_noNonSpecContentBlocks(t *testing.T) {
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
			result, ok := out["result"].(map[string]any)
			if !ok || result == nil {
				t.Fatalf("tools/call %s missing result: %+v", spec.Name, out)
			}
			content, _ := result["content"].([]any)
			for i, block := range content {
				b, ok := block.(map[string]any)
				if !ok {
					t.Fatalf("content[%d] is not an object: %+v", i, block)
				}
				typ, _ := b["type"].(string)
				if !mcpSpecContentTypes[typ] {
					t.Fatalf("content[%d].type=%q is not an MCP spec content type (M1 regression)", i, typ)
				}
				if typ == "text" {
					if txt, _ := b["text"].(string); txt == "" {
						t.Fatalf("content[%d] text block has empty text", i)
					}
				}
				if _, hasJSON := b["json"]; hasJSON {
					t.Fatalf("content[%d] carries a legacy json field (M1 regression); use structuredContent", i)
				}
			}
		})
	}
}
