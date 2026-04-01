package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// EnforcementMaxProposalBytes must match internal/enforcement/validate.go (EvaluateRequest proposal_text cap).
const EnforcementMaxProposalBytes = 32768

// HandleToolsCall forwards an MCP tools/call to control-plane HTTP (same mapping as the stdio MCP adapter).
// policy gates record_experience / mcp_episode_ingest (nil uses DefaultMemoryFormationPolicy).
func HandleToolsCall(client *http.Client, base, apiKey string, params json.RawMessage, policy *MemoryFormationPolicy) (any, error) {
	pol := NormalizeMemoryFormation(policy)
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
	case "recall_context", "memory_context_resolve":
		return execMemoryContextResolve(client, base, apiKey, p.Arguments), nil
	case "memory_log_if_relevant", "auto_log_episode_if_relevant":
		return execMemoryLogIfRelevant(client, base, apiKey, p.Arguments, pol), nil
	case "record_experience", "mcp_episode_ingest":
		payload, vErr := buildAdvisoryEpisodeMCPBody(p.Arguments, pol)
		if vErr != nil {
			return ToolResultErr(vErr.Error()), nil
		}
		b, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		method = http.MethodPost
		fullURL = base + "/v1/advisory-episodes"
		body = bytes.NewReader(b)
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
	case "curation_pending":
		method = http.MethodGet
		fullURL = base + "/v1/curation/pending"
	case "curation_promotion_suggestions":
		method = http.MethodGet
		fullURL = base + "/v1/curation/promotion-suggestions"
	case "curation_strengthened":
		method = http.MethodGet
		min := parseOptionalMinSupport(p.Arguments)
		fullURL = base + "/v1/curation/strengthened?min_support=" + url.QueryEscape(strconv.Itoa(min))
	case "curation_materialize", "curation_promote_candidate":
		method = http.MethodPost
		candID, err := ParseCandidateID(p.Arguments)
		if err != nil {
			return nil, err
		}
		fullURL = base + "/v1/curation/candidates/" + url.PathEscape(candID) + "/materialize"
	case "curation_review_candidate":
		method = http.MethodGet
		candID, err := ParseCandidateID(p.Arguments)
		if err != nil {
			return nil, err
		}
		fullURL = base + "/v1/curation/candidates/" + url.PathEscape(candID) + "/review"
	case "curation_reject_candidate":
		method = http.MethodPost
		candID, err := ParseCandidateID(p.Arguments)
		if err != nil {
			return nil, err
		}
		fullURL = base + "/v1/curation/candidates/" + url.PathEscape(candID) + "/reject"
	case "curation_auto_promote":
		method = http.MethodPost
		fullURL = base + "/v1/curation/auto-promote"
		if len(bytes.TrimSpace(p.Arguments)) > 0 {
			body = bytes.NewReader(p.Arguments)
		} else {
			body = bytes.NewReader([]byte("{}"))
		}
	case "episode_search_similar":
		method = http.MethodPost
		fullURL = base + "/v1/advisory-episodes/similar"
		b, err := buildEpisodeSimilarBody(p.Arguments)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	case "episode_distill_explicit":
		method = http.MethodPost
		fullURL = base + "/v1/episodes/distill"
		b, err := buildEpisodeDistillBody(p.Arguments)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	case "memory_recall_advanced":
		method = http.MethodPost
		fullURL = base + "/v1/recall/compile"
		b, err := buildRecallAdvancedBody(p.Arguments)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	case "memory_preflight_check":
		method = http.MethodPost
		fullURL = base + "/v1/recall/preflight"
		b, err := buildPreflightBody(p.Arguments)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	case "memory_detect_contradictions":
		method = http.MethodPost
		fullURL = base + "/v1/contradictions/detect"
		b, err := buildContradictionDetectBody(p.Arguments)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(b)
	case "memory_list_contradictions":
		method = http.MethodGet
		fullURL, err = buildContradictionsListURL(base, p.Arguments)
		if err != nil {
			return nil, err
		}
	case "evidence_attach":
		return evidenceAttach(client, base, apiKey, p.Arguments), nil
	case "evidence_list":
		method = http.MethodGet
		fullURL, err = buildEvidenceListURL(base, p.Arguments)
		if err != nil {
			return nil, err
		}
	case "memory_relationships_get":
		method = http.MethodGet
		mid, err := parseRequiredUUIDArg(p.Arguments, "memory_id")
		if err != nil {
			return nil, err
		}
		fullURL = base + "/v1/memory/" + url.PathEscape(mid) + "/relationships"
	case "memory_relationships_create":
		method = http.MethodPost
		fullURL = base + "/v1/memory/relationships"
		if len(bytes.TrimSpace(p.Arguments)) == 0 {
			return nil, fmt.Errorf("memory_relationships_create requires arguments (from_memory_id, to_memory_id, relationship_type)")
		}
		body = bytes.NewReader(p.Arguments)
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
	if resp.StatusCode < 400 && method == http.MethodPost && strings.HasSuffix(fullURL, "/v1/advisory-episodes") {
		rawBody = augmentAdvisoryEpisodeSuccessJSON(rawBody)
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

func parseOptionalMinSupport(arguments json.RawMessage) int {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return 2
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return 2
	}
	switch v := m["min_support"].(type) {
	case float64:
		if v >= 1 {
			return int(v)
		}
	case int:
		if v >= 1 {
			return v
		}
	}
	return 2
}

// augmentAdvisoryEpisodeSuccessJSON adds a single affordance field on successful advisory creates (non-breaking for JSON clients).
func augmentAdvisoryEpisodeSuccessJSON(raw []byte) []byte {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return raw
	}
	m["mcp_affordance"] = "Advisory experience stored. Server may create probationary memory immediately when learning signals qualify; otherwise reject bucket only. Optional auto-distill may add pending candidates."
	b, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return b
}

func buildAdvisoryEpisodeMCPBody(arguments json.RawMessage, pol *MemoryFormationPolicy) (map[string]any, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, fmt.Errorf("record_experience/mcp_episode_ingest requires arguments with summary")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, fmt.Errorf("record_experience/mcp_episode_ingest: %w", err)
	}
	summary, _ := m["summary"].(string)
	if err := ValidateMcpEpisodeSummary(summary, pol); err != nil {
		return nil, err
	}
	tags := parseStringSliceField(m, "tags")
	if ek, ok := m["event_kind"].(string); ok && strings.TrimSpace(ek) != "" {
		tags = append(tags, "mcp:event:"+sanitizeMcpEventKind(ek))
	}
	out := map[string]any{
		"summary": strings.TrimSpace(summary),
		"source":  "mcp",
	}
	if len(tags) > 0 {
		out["tags"] = tags
	}
	if s, ok := m["correlation_id"].(string); ok && strings.TrimSpace(s) != "" {
		out["correlation_id"] = strings.TrimSpace(s)
	}
	ents := parseStringSliceField(m, "entities")
	if len(ents) > 0 {
		out["entities"] = ents
	}
	return out, nil
}

func parseStringSliceField(m map[string]any, key string) []string {
	v, ok := m[key]
	if !ok || v == nil {
		return nil
	}
	switch x := v.(type) {
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok && strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	case []string:
		var out []string
		for _, s := range x {
			if strings.TrimSpace(s) != "" {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out
	default:
		return nil
	}
}

func sanitizeMcpEventKind(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	var b strings.Builder
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			b.WriteRune(r)
		}
	}
	out := b.String()
	if len(out) > 64 {
		out = out[:64]
	}
	if out == "" {
		return "unspecified"
	}
	return out
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
