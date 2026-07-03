package mcp

import (
	"encoding/json"
)

// structuredRecallTools return agent-facing recall payloads as JSON (not text-only).
var structuredRecallTools = map[string]string{
	"recall_compile":         "recall_bundle",
	"recall_get":             "recall_bundle",
	"memory_recall_advanced": "recall_bundle",
	"wakeup_context":         "wakeup_response",
	"recall_run_multi":       "run_multi_response",
}

// structuredTelemetryTools return agent telemetry payloads as JSON.
var structuredTelemetryTools = map[string]string{
	"agent_telemetry_start_session":        "telemetry_session",
	"agent_telemetry_record_recall":        "telemetry_recall",
	"agent_telemetry_record_decision":      "telemetry_decision",
	"agent_telemetry_record_output":        "telemetry_output",
	"agent_telemetry_evaluate":             "telemetry_evaluation",
	"agent_telemetry_get_session":          "telemetry_session_summary",
	"agent_telemetry_get_memory":           "telemetry_memory_summary",
	"agent_telemetry_get_violations":       "telemetry_violations",
	"agent_telemetry_get_utility_candidates": "telemetry_utility_candidates",
}

func isStructuredTelemetryTool(name string) bool {
	_, ok := structuredTelemetryTools[name]
	return ok
}

func toolResultStructuredTelemetry(toolName string, rawBody []byte, statusErr bool, statusText string) map[string]any {
	key, ok := structuredTelemetryTools[toolName]
	if !ok {
		return nil
	}
	if statusErr {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": statusText},
			},
			"isError": true,
		}
	}
	var payload json.RawMessage
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return ToolResultErr(toolName + ": invalid JSON response")
	}
	wrap := map[string]any{key: payload}
	return toolResultStructuredJSON(wrap)
}

func isStructuredRecallTool(name string) bool {
	_, ok := structuredRecallTools[name]
	return ok
}

func toolResultStructuredRecall(toolName string, rawBody []byte, statusErr bool, statusText string) map[string]any {
	key, ok := structuredRecallTools[toolName]
	if !ok {
		return nil
	}
	if statusErr {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": statusText},
			},
			"isError": true,
		}
	}
	var payload json.RawMessage
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return ToolResultErr(toolName + ": invalid JSON response")
	}
	wrap := map[string]any{key: payload}
	return toolResultStructuredJSON(wrap)
}

// toolResultStructuredJSON returns an MCP spec-compliant result: content blocks
// only use the "text" type (the MCP spec allows text, image, audio, resource_link,
// resource — never "json"); the machine-readable payload rides in structuredContent.
func toolResultStructuredJSON(payload map[string]any) map[string]any {
	out, err := json.Marshal(payload)
	if err != nil {
		return ToolResultErr(err.Error())
	}
	return map[string]any{
		"content": []map[string]any{
			{
				"type": "text",
				"text": string(out),
			},
		},
		"structuredContent": payload,
		"isError":           false,
	}
}
