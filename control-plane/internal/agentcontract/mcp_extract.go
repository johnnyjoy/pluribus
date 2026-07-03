package agentcontract

import (
	"encoding/json"
	"fmt"

	"control-plane/internal/recall"
)

// MCPJSONContent extracts the structured JSON payload from an MCP tool result.
// Spec-compliant results carry the machine-readable payload in structuredContent
// with a serialized-JSON text block (the MCP spec forbids a "json" content type).
// The legacy {"type":"json","json":...} block and JSON-parseable text blocks are
// also accepted; only flattened prose without a structured payload is a defect.
func MCPJSONContent(toolResp any) (map[string]any, error) {
	raw, err := json.Marshal(toolResp)
	if err != nil {
		return nil, err
	}
	var out struct {
		StructuredContent map[string]any `json:"structuredContent"`
		Content           []struct {
			Type string         `json:"type"`
			JSON map[string]any `json:"json"`
			Text string         `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out.StructuredContent != nil {
		return out.StructuredContent, nil
	}
	if len(out.Content) == 0 {
		return nil, fmt.Errorf("mcp response missing content")
	}
	if out.Content[0].Type == "json" && out.Content[0].JSON != nil {
		return out.Content[0].JSON, nil
	}
	if out.Content[0].Type == "text" && out.Content[0].Text != "" {
		var m map[string]any
		if json.Unmarshal([]byte(out.Content[0].Text), &m) == nil && m != nil {
			return m, nil
		}
	}
	return nil, fmt.Errorf("mcp_text_only_without_json")
}

// MCPRecallBundleFromTool extracts recall_bundle from MCP structured JSON.
func MCPRecallBundleFromTool(toolResp any) (*recall.RecallBundle, error) {
	wrap, err := MCPJSONContent(toolResp)
	if err != nil {
		return nil, err
	}
	raw, ok := wrap["recall_bundle"]
	if !ok {
		return nil, fmt.Errorf("missing recall_bundle in mcp json")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var bundle recall.RecallBundle
	if err := json.Unmarshal(b, &bundle); err != nil {
		return nil, err
	}
	return &bundle, nil
}

// MCPWakeupFromTool extracts wakeup_response from MCP structured JSON.
func MCPWakeupFromTool(toolResp any) (*recall.WakeupResponse, error) {
	wrap, err := MCPJSONContent(toolResp)
	if err != nil {
		return nil, err
	}
	raw, ok := wrap["wakeup_response"]
	if !ok {
		return nil, fmt.Errorf("missing wakeup_response in mcp json")
	}
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var w recall.WakeupResponse
	if err := json.Unmarshal(b, &w); err != nil {
		return nil, err
	}
	return &w, nil
}

// MCPIsStructuredJSON reports whether MCP tool response uses structured JSON (not text-only safety source).
func MCPIsStructuredJSON(toolResp any) bool {
	_, err := MCPJSONContent(toolResp)
	return err == nil
}
