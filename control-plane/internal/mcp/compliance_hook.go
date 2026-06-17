package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"control-plane/internal/compliance"

	"github.com/google/uuid"
)

// complianceRecorder is optional loop telemetry (nil disables recording).
type complianceRecorder interface {
	EnsureSession(ctx context.Context, mc compliance.MCPContext) error
	RecordMCPMethod(ctx context.Context, mc compliance.MCPContext, method string, errCode int, errMsg string, duration time.Duration)
	RecordToolCall(ctx context.Context, mc compliance.MCPContext, toolName string, args json.RawMessage, started time.Time, success bool, errCode int, errMsg, enforcementDecision string)
}

// SetComplianceTelemetry attaches Phase 2 loop telemetry to an HTTP MCP handler.
func SetComplianceTelemetry(h http.Handler, svc *compliance.Service) http.Handler {
	if hh, ok := h.(*httpHandler); ok {
		hh.telemetry = svc
		return hh
	}
	return h
}

func (h *httpHandler) mcpContext(initName, initVersion string) compliance.MCPContext {
	mc := compliance.ContextFromRequest(h.req, initName, initVersion)
	if h.sessionID != uuid.Nil {
		mc.SessionID = h.sessionID
	}
	return mc
}

func (h *httpHandler) recordMethod(method string, started time.Time, err *jsonRPCErrorObj) {
	if h.telemetry == nil {
		return
	}
	code, msg := 0, ""
	if err != nil {
		code = err.Code
		msg = err.Message
	}
	h.telemetry.RecordMCPMethod(context.Background(), h.mcpContext("", ""), method, code, msg, time.Since(started))
}

func (h *httpHandler) recordToolCall(toolName string, args json.RawMessage, started time.Time, wireErr *jsonRPCErrorObj, result any) {
	if h.telemetry == nil {
		return
	}
	success := wireErr == nil
	code, msg := 0, ""
	if wireErr != nil {
		code = wireErr.Code
		msg = wireErr.Message
	}
	enforceDecision := extractEnforcementDecision(result)
	h.telemetry.RecordToolCall(context.Background(), h.mcpContext("", ""), toolName, args, started, success, code, msg, enforceDecision)
}

func extractEnforcementDecision(result any) string {
	m, ok := result.(map[string]any)
	if !ok {
		return ""
	}
	content, _ := m["content"].([]any)
	if len(content) == 0 {
		return ""
	}
	c0, _ := content[0].(map[string]any)
	text, _ := c0["text"].(string)
	if !strings.Contains(text, "decision") {
		return ""
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(text), &parsed); err != nil {
		return ""
	}
	if d, ok := parsed["decision"].(string); ok {
		return d
	}
	return ""
}

func enrichInitializeResult(result map[string]any, sessionID string) map[string]any {
	if sessionID == "" {
		return result
	}
	result["pluribus"] = map[string]any{
		"session_id": sessionID,
		"headers": map[string]string{
			compliance.HeaderSessionID:     "Send on subsequent MCP requests to correlate telemetry",
			compliance.HeaderCorrelationID: "Optional per-call correlation",
		},
	}
	return result
}
