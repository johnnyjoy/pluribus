package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
)

// EnforcementMaxProposalBytes must match internal/enforcement/validate.go (EvaluateRequest proposal_text cap).
const EnforcementMaxProposalBytes = 32768

// HandleToolsCall forwards an MCP tools/call to control-plane HTTP (same mapping as the stdio MCP adapter).
func HandleToolsCall(client *http.Client, base, apiKey string, params json.RawMessage) (any, error) {
	var p toolsCallParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, err
	}
	if p.Name == "" {
		return nil, fmt.Errorf("missing tool name")
	}
	var (
		body      io.Reader
		method    string
		fullURL   string
		err       error
		statusErr bool
		rawBody   []byte
	)

	switch p.Name {
	case "health":
		method = http.MethodGet
		fullURL = base + "/healthz"
	case "recall_compile":
		method = http.MethodPost
		fullURL = base + "/v1/recall/compile"
		if len(p.Arguments) == 0 {
			return nil, fmt.Errorf("recall_compile requires arguments (JSON body)")
		}
		body = bytes.NewReader(p.Arguments)
	case "recall_get":
		method = http.MethodGet
		fullURL, err = BuildRecallGetURL(base, p.Arguments)
		if err != nil {
			return nil, err
		}
	case "recall_run_multi":
		method = http.MethodPost
		fullURL = base + "/v1/recall/run-multi"
		if len(p.Arguments) == 0 {
			return nil, fmt.Errorf("recall_run_multi requires arguments")
		}
		body = bytes.NewReader(p.Arguments)
	case "memory_create":
		method = http.MethodPost
		fullURL = base + "/v1/memory"
		if len(p.Arguments) == 0 {
			return nil, fmt.Errorf("memory_create requires arguments")
		}
		body = bytes.NewReader(p.Arguments)
	case "memory_promote":
		method = http.MethodPost
		fullURL = base + "/v1/memory/promote"
		if len(p.Arguments) == 0 {
			return nil, fmt.Errorf("memory_promote requires arguments")
		}
		body = bytes.NewReader(p.Arguments)
	case "curation_digest":
		method = http.MethodPost
		fullURL = base + "/v1/curation/digest"
		if err := ValidateDigestArguments(p.Arguments); err != nil {
			return nil, err
		}
		body = bytes.NewReader(p.Arguments)
	case "curation_materialize":
		method = http.MethodPost
		candID, err := ParseCandidateID(p.Arguments)
		if err != nil {
			return nil, err
		}
		fullURL = base + "/v1/curation/candidates/" + url.PathEscape(candID) + "/materialize"
	case "enforcement_evaluate":
		method = http.MethodPost
		fullURL = base + "/v1/enforcement/evaluate"
		if err := ValidateEnforcementArguments(p.Arguments); err != nil {
			return nil, err
		}
		body = bytes.NewReader(p.Arguments)
	default:
		return nil, fmt.Errorf("unknown tool: %s", p.Name)
	}

	req, err := http.NewRequest(method, fullURL, body)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	resp, err := client.Do(req)
	if err != nil {
		return ToolResultErr(fmt.Sprintf("http error: %v", err)), nil
	}
	defer resp.Body.Close()
	rawBody, _ = io.ReadAll(resp.Body)
	if len(rawBody) > 4*1024*1024 {
		rawBody = append(rawBody[:4*1024*1024], []byte("\n...truncated")...)
	}
	text := string(rawBody)
	if resp.StatusCode >= 400 {
		statusErr = true
		text = fmt.Sprintf("HTTP %s\n%s", resp.Status, text)
	}
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"isError": statusErr,
	}, nil
}

type toolsCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// ValidateDigestArguments ensures MCP calls match DigestRequest requirements before HTTP (clear errors; bounded intent).
func ValidateDigestArguments(arguments json.RawMessage) error {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return fmt.Errorf("curation_digest requires arguments (work_summary, …)")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return fmt.Errorf("curation_digest arguments: %w", err)
	}
	ws, _ := m["work_summary"].(string)
	if strings.TrimSpace(ws) == "" {
		return fmt.Errorf("curation_digest requires work_summary (bounded narrative; see server digest limits)")
	}
	return nil
}

// ParseCandidateID extracts a UUID from MCP arguments for curation_materialize (candidate_id or id).
func ParseCandidateID(arguments json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return "", fmt.Errorf("curation_materialize requires arguments with candidate_id (UUID)")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return "", fmt.Errorf("curation_materialize arguments: %w", err)
	}
	var raw string
	if s, ok := m["candidate_id"].(string); ok {
		raw = strings.TrimSpace(s)
	}
	if raw == "" {
		if s, ok := m["id"].(string); ok {
			raw = strings.TrimSpace(s)
		}
	}
	if raw == "" {
		return "", fmt.Errorf("curation_materialize requires candidate_id (UUID)")
	}
	if _, err := uuid.Parse(raw); err != nil {
		return "", fmt.Errorf("curation_materialize: candidate_id must be a valid UUID")
	}
	return raw, nil
}

// ValidateEnforcementArguments ensures MCP calls match EvaluateRequest before HTTP.
func ValidateEnforcementArguments(arguments json.RawMessage) error {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return fmt.Errorf("enforcement_evaluate requires arguments (proposal_text, …)")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return fmt.Errorf("enforcement_evaluate arguments: %w", err)
	}
	pt, _ := m["proposal_text"].(string)
	if strings.TrimSpace(pt) == "" {
		return fmt.Errorf("enforcement_evaluate requires proposal_text (bounded; see server limits)")
	}
	if len(pt) > EnforcementMaxProposalBytes {
		return fmt.Errorf("enforcement_evaluate: proposal_text exceeds max length (%d bytes)", EnforcementMaxProposalBytes)
	}
	return nil
}

// ToolResultErr builds an MCP tool error payload.
func ToolResultErr(msg string) map[string]any {
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": msg},
		},
		"isError": true,
	}
}

// BuildRecallGetURL builds GET /v1/recall/ URL with optional query params from JSON arguments.
func BuildRecallGetURL(base string, arguments json.RawMessage) (string, error) {
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return "", err
	}
	q := url.Values{}
	if s, ok := m["retrieval_query"].(string); ok && strings.TrimSpace(s) != "" {
		q.Set("retrieval_query", strings.TrimSpace(s))
	} else if s, ok := m["query"].(string); ok && strings.TrimSpace(s) != "" {
		q.Set("retrieval_query", strings.TrimSpace(s))
	}
	appendRepeated := func(key string) {
		switch v := m[key].(type) {
		case nil:
		case string:
			if v != "" {
				q.Add(key, v)
			}
		case []any:
			for _, x := range v {
				if s, ok := x.(string); ok && s != "" {
					q.Add(key, s)
				}
			}
		}
	}
	appendRepeated("tags")
	appendRepeated("symbols")
	for _, key := range []string{"max_per_kind", "max_total", "max_tokens"} {
		if v, ok := m[key]; ok && v != nil {
			switch n := v.(type) {
			case float64:
				q.Set(key, fmt.Sprintf("%.0f", n))
			case int:
				q.Set(key, fmt.Sprintf("%d", n))
			case string:
				if n != "" {
					q.Set(key, n)
				}
			}
		}
	}
	return base + "/v1/recall/?" + q.Encode(), nil
}
