package agentcontract

import (
	"encoding/json"
	"fmt"

	"control-plane/internal/recall"
)

// MCPJSONContent extracts the structured JSON payload from an MCP tool result.
func MCPJSONContent(toolResp any) (map[string]any, error) {
	raw, err := json.Marshal(toolResp)
	if err != nil {
		return nil, err
	}
	var out struct {
		Content []struct {
			Type string         `json:"type"`
			JSON map[string]any `json:"json"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if len(out.Content) == 0 {
		return nil, fmt.Errorf("mcp response missing content")
	}
	if out.Content[0].Type != "json" {
		return nil, fmt.Errorf("mcp_text_only_without_json")
	}
	if out.Content[0].JSON == nil {
		return nil, fmt.Errorf("mcp json content missing json field")
	}
	return out.Content[0].JSON, nil
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
