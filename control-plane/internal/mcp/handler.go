package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"control-plane/internal/app"
	"control-plane/internal/compliance"
	"control-plane/internal/formation"

	"github.com/google/uuid"
)

// Version is the MCP serverInfo.version (stdio and HTTP share this).
const Version = "0.1.8"

const (
	loopbackHost      = "mcp.loopback.invalid"
	maxMCPBodyBytes   = 16 << 20
	mcpServerNameHTTP = "pluribus"
)

// WrapHandler wraps the API handler. When MCP is enabled (default), POST /v1/mcp serves
// MCP JSON-RPC; all other requests go to inner. inner receives loopback tool traffic (same
// middleware and routes as real HTTP).
func WrapHandler(inner http.Handler, cfg *app.Config, telemetry *compliance.Service) http.Handler {
	tier := ""
	if cfg != nil && cfg.MCP != nil {
		tier = cfg.MCP.ToolsTier
	}
	InitToolsTier(tier)
	if cfg == nil || !cfg.MCPEnabled() {
		return inner
	}
	h := NewHTTPHandler(inner, PolicyFromAppConfig(cfg), telemetry, formation.NewGate(cfg.Memory.Formation))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/v1/mcp" {
			h.ServeHTTP(w, r)
			return
		}
		inner.ServeHTTP(w, r)
	})
}

// NewHTTPHandler serves MCP JSON-RPC over HTTP. Tool calls use HTTP client with a loopback
// RoundTripper into inner (preserves API key middleware on nested requests).
func NewHTTPHandler(inner http.Handler, policy *MemoryFormationPolicy, telemetry *compliance.Service, gate *formation.Gate) http.Handler {
	client := &http.Client{
		Transport: &loopbackTransport{h: inner},
		Timeout:   10 * time.Minute,
	}
	base := "http://" + loopbackHost
	return &httpHandler{client: client, base: base, policy: policy, telemetry: telemetry, formation: gate}
}

type loopbackTransport struct {
	h http.Handler
}

func (t *loopbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.h.ServeHTTP(rec, req)
	return rec.Result(), nil
}

type httpHandler struct {
	client    *http.Client
	base      string
	policy    *MemoryFormationPolicy
	telemetry *compliance.Service
	formation *formation.Gate
	req       *http.Request
	sessionID uuid.UUID
}

type jsonRPCWire struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id"`
	Method  string           `json:"method"`
	Params  json.RawMessage  `json:"params"`
}

type jsonRPCWireResponse struct {
	JSONRPC string           `json:"jsonrpc"`
	ID      *json.RawMessage `json:"id,omitempty"`
	Result  any              `json:"result,omitempty"`
	Error   *jsonRPCErrorObj `json:"error,omitempty"`
}

type jsonRPCErrorObj struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *httpHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.req = r
	h.sessionID = uuid.Nil
	if sid := strings.TrimSpace(r.Header.Get(compliance.HeaderSessionID)); sid != "" {
		if id, err := uuid.Parse(sid); err == nil {
			h.sessionID = id
		}
	}
	if r.Method != http.MethodPost {
		http.Error(w, "MCP endpoint accepts POST only", http.StatusMethodNotAllowed)
		return
	}
	ct := strings.ToLower(r.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "application/json") {
		http.Error(w, "Content-Type must be application/json for MCP JSON-RPC", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, maxMCPBodyBytes))
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to read request body: %v", err), http.StatusBadRequest)
		return
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 {
		http.Error(w, "request body is empty (expected JSON-RPC payload)", http.StatusBadRequest)
		return
	}
	apiKey := strings.TrimSpace(r.Header.Get("X-API-Key"))
	if apiKey == "" {
		apiKey = strings.TrimSpace(r.URL.Query().Get("token"))
	}

	if body[0] == '[' {
		var batch []json.RawMessage
		if err := json.Unmarshal(body, &batch); err != nil {
			writeJSONRPCParseError(w)
			return
		}
		out := make([]jsonRPCWireResponse, 0, len(batch))
		for _, raw := range batch {
			resp, ok := h.handleOne(raw, apiKey)
			if !ok {
				continue
			}
			out = append(out, resp)
		}
		writeJSON(w, http.StatusOK, out)
		return
	}

	resp, ok := h.handleOne(body, apiKey)
	if !ok {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

func writeJSONRPCParseError(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"jsonrpc":"2.0","error":{"code":-32700,"message":"parse error: invalid JSON in request body"},"id":null}`))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

// handleOne returns (response, false) for notifications; (response, true) when a JSON body should be sent.
func (h *httpHandler) handleOne(raw json.RawMessage, apiKey string) (jsonRPCWireResponse, bool) {
	var req jsonRPCWire
	if err := json.Unmarshal(raw, &req); err != nil {
		return jsonRPCWireResponse{
			JSONRPC: "2.0",
			Error:   &jsonRPCErrorObj{Code: -32700, Message: "parse error"},
		}, true
	}
	if req.JSONRPC != "2.0" {
		return jsonRPCWireResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCErrorObj{Code: -32600, Message: `invalid request: "jsonrpc" must be "2.0"`},
		}, req.ID != nil
	}
	if req.ID == nil && strings.HasPrefix(req.Method, "notifications/") {
		return jsonRPCWireResponse{}, false
	}
	if req.Method == "" {
		return jsonRPCWireResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   &jsonRPCErrorObj{Code: -32600, Message: "invalid request: method field is required"},
		}, req.ID != nil
	}

	started := time.Now()
	result, wireErr := h.dispatch(req.Method, req.Params, apiKey)
	h.recordMethod(req.Method, started, wireErr)
	if wireErr != nil {
		return jsonRPCWireResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error:   wireErr,
		}, req.ID != nil
	}
	return jsonRPCWireResponse{JSONRPC: "2.0", ID: req.ID, Result: result}, req.ID != nil
}

func (h *httpHandler) dispatch(method string, params json.RawMessage, apiKey string) (any, *jsonRPCErrorObj) {
	switch method {
	case "initialize":
		mc := h.mcpContext("", "")
		var initParams struct {
			ClientInfo struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			} `json:"clientInfo"`
		}
		_ = json.Unmarshal(params, &initParams)
		mc = compliance.ContextFromRequest(h.req, initParams.ClientInfo.Name, initParams.ClientInfo.Version)
		if h.telemetry != nil {
			_ = h.telemetry.EnsureSession(context.Background(), mc)
		}
		h.sessionID = mc.SessionID
		res := InitializeResult(mcpServerNameHTTP, Version)
		return enrichInitializeResult(res, mc.SessionID.String()), nil
	case "ping":
		return map[string]any{}, nil
	case "tools/list":
		return map[string]any{"tools": ToolDefinitions()}, nil
	case "tools/call":
		var p struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &jsonRPCErrorObj{Code: -32602, Message: "invalid tools/call params: " + err.Error()}
		}
		if strings.TrimSpace(p.Name) == "" {
			return nil, &jsonRPCErrorObj{Code: -32602, Message: "missing required argument: name"}
		}
		callStarted := time.Now()
		if err := ValidateToolArguments(p.Name, p.Arguments); err != nil {
			h.recordToolCall(p.Name, p.Arguments, callStarted, &jsonRPCErrorObj{Code: -32602, Message: err.Error()}, nil)
			return nil, &jsonRPCErrorObj{Code: -32602, Message: err.Error()}
		}
		callParams, _ := json.Marshal(p)
		res, err := HandleToolsCall(h.client, h.base, apiKey, callParams, h.policy, h.formation)
		if err != nil {
			h.recordToolCall(p.Name, p.Arguments, callStarted, &jsonRPCErrorObj{Code: -32000, Message: err.Error()}, nil)
			return nil, &jsonRPCErrorObj{Code: -32000, Message: err.Error()}
		}
		h.recordToolCall(p.Name, p.Arguments, callStarted, nil, res)
		return res, nil
	case "prompts/list":
		return map[string]any{"prompts": PromptDefinitions()}, nil
	case "prompts/get":
		var p struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &jsonRPCErrorObj{Code: -32602, Message: "invalid prompts/get params"}
		}
		msgs, ok := PromptMessages(strings.TrimSpace(p.Name))
		if !ok {
			return nil, &jsonRPCErrorObj{Code: -32602, Message: fmt.Sprintf("unknown prompt: %s", p.Name)}
		}
		return map[string]any{"messages": msgs}, nil
	case "resources/list":
		return map[string]any{"resources": ResourceDefinitions()}, nil
	case "resources/read":
		var p struct {
			URI string `json:"uri"`
		}
		if err := json.Unmarshal(params, &p); err != nil {
			return nil, &jsonRPCErrorObj{Code: -32602, Message: "invalid resources/read params"}
		}
		text, ok := ResourceText(strings.TrimSpace(p.URI))
		if !ok {
			return nil, &jsonRPCErrorObj{Code: -32602, Message: fmt.Sprintf("unknown resource: %s", p.URI)}
		}
		return map[string]any{
			"contents": []map[string]any{
				{
					"uri":      p.URI,
					"mimeType": "text/markdown",
					"text":     text,
				},
			},
		}, nil
	default:
		if strings.HasPrefix(method, "notifications/") {
			return nil, nil
		}
		return nil, &jsonRPCErrorObj{Code: -32601, Message: "method not found: " + method}
	}
}
