package mcp

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

// buildEpisodeSimilarBody maps MCP args to POST /v1/advisory-episodes/similar JSON (SimilarRequest).
func buildEpisodeSimilarBody(arguments json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, fmt.Errorf("episode_search_similar requires arguments with query or summary_text")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, err
	}
	q := strings.TrimSpace(firstString(m, "query", "summary_text"))
	if q == "" {
		return nil, fmt.Errorf("episode_search_similar requires query or summary_text")
	}
	out := map[string]any{"query": q}
	if v, ok := m["tags"]; ok {
		out["tags"] = v
	}
	if v, ok := m["max_results"]; ok {
		out["max_results"] = v
	}
	if v, ok := m["occurred_after"]; ok {
		out["occurred_after"] = v
	}
	if v, ok := m["occurred_before"]; ok {
		out["occurred_before"] = v
	}
	if v, ok := m["entity"]; ok {
		out["entity"] = v
	}
	if v, ok := m["entities"]; ok {
		out["entities"] = v
	}
	return json.Marshal(out)
}

func firstString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if s, ok := m[k].(string); ok && strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

// buildEpisodeDistillBody maps MCP args to POST /v1/episodes/distill (DistillRequest).
func buildEpisodeDistillBody(arguments json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, fmt.Errorf("episode_distill_explicit requires arguments with episode_id or summary")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, err
	}
	ep, _ := m["episode_id"].(string)
	sum, _ := m["summary"].(string)
	ep, sum = strings.TrimSpace(ep), strings.TrimSpace(sum)
	if ep == "" && sum == "" {
		return nil, fmt.Errorf("episode_distill_explicit requires episode_id or summary")
	}
	out := map[string]any{}
	if ep != "" {
		out["episode_id"] = ep
	}
	if sum != "" {
		out["summary"] = sum
	}
	if v, ok := m["tags"]; ok {
		out["tags"] = v
	}
	if v, ok := m["entities"]; ok {
		out["entities"] = v
	}
	return json.Marshal(out)
}

// buildRecallAdvancedBody maps query + mode to POST /v1/recall/compile (CompileRequest).
// mode: continuity | constraint | pattern | episodic (maps to compile wire fields; see tool description).
func buildRecallAdvancedBody(arguments json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, fmt.Errorf("memory_recall_advanced requires arguments")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, err
	}
	q := strings.TrimSpace(firstString(m, "query", "retrieval_query"))
	if q == "" {
		return nil, fmt.Errorf("memory_recall_advanced requires query")
	}
	mode := strings.ToLower(strings.TrimSpace(firstString(m, "mode")))
	if mode == "" {
		mode = "continuity"
	}
	out := map[string]any{"retrieval_query": q}
	for _, k := range []string{"tags", "symbols", "agent_id", "repo_root", "proposal_text", "enable_triggered_recall"} {
		if v, ok := m[k]; ok {
			out[k] = v
		}
	}
	switch mode {
	case "continuity":
		out["mode"] = "continuity"
	case "constraint":
		out["mode"] = "continuity"
		c := 15
		out["variant_modifier"] = map[string]any{"limits": map[string]any{"constraints": c}}
	case "pattern":
		out["mode"] = "continuity"
		p := 15
		out["variant_modifier"] = map[string]any{"limits": map[string]any{"patterns": p}}
	case "episodic":
		out["mode"] = "thread"
	default:
		out["mode"] = mode
	}
	return json.Marshal(out)
}

// buildPreflightBody forwards POST /v1/recall/preflight (PreflightRequest).
func buildPreflightBody(arguments json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return []byte("{}"), nil
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, err
	}
	out := map[string]any{}
	if v, ok := m["changed_files_count"]; ok {
		out["changed_files_count"] = v
	}
	if v, ok := m["tags"]; ok {
		out["tags"] = v
	}
	if len(out) == 0 {
		return []byte("{}"), nil
	}
	return json.Marshal(out)
}

// buildContradictionsListURL appends query params for GET /v1/contradictions.
// buildEvidenceListURL is GET /v1/evidence?memory_id=… or ?kind=… (mutually exclusive priority: memory_id).
func buildEvidenceListURL(base string, arguments json.RawMessage) (string, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return "", fmt.Errorf("evidence_list requires memory_id or kind")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return "", err
	}
	if mid := strings.TrimSpace(firstString(m, "memory_id")); mid != "" {
		if _, err := uuid.Parse(mid); err != nil {
			return "", fmt.Errorf("memory_id: %w", err)
		}
		return base + "/v1/evidence?memory_id=" + url.QueryEscape(mid), nil
	}
	if k := strings.TrimSpace(firstString(m, "kind")); k != "" {
		return base + "/v1/evidence?kind=" + url.QueryEscape(k), nil
	}
	return "", fmt.Errorf("evidence_list requires memory_id or kind")
}

func buildContradictionsListURL(base string, arguments json.RawMessage) (string, error) {
	q := url.Values{}
	if len(bytes.TrimSpace(arguments)) == 0 {
		return base + "/v1/contradictions?" + q.Encode(), nil
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return "", err
	}
	if s, ok := m["resolution_state"].(string); ok && strings.TrimSpace(s) != "" {
		q.Set("resolution_state", strings.TrimSpace(s))
	}
	if s, ok := m["memory_id"].(string); ok && strings.TrimSpace(s) != "" {
		q.Set("memory_id", strings.TrimSpace(s))
	}
	switch v := m["limit"].(type) {
	case float64:
		if v >= 1 && v <= 500 {
			q.Set("limit", strconv.Itoa(int(v)))
		}
	case int:
		if v >= 1 && v <= 500 {
			q.Set("limit", strconv.Itoa(v))
		}
	}
	return base + "/v1/contradictions?" + q.Encode(), nil
}

// buildContradictionDetectBody maps to POST /v1/contradictions/detect (two memory UUIDs).
// parseRequiredUUIDArg extracts a UUID string from arguments (first matching key).
func parseRequiredUUIDArg(arguments json.RawMessage, keys ...string) (string, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return "", fmt.Errorf("missing arguments")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return "", err
	}
	s := strings.TrimSpace(firstString(m, keys...))
	if s == "" {
		return "", fmt.Errorf("requires one of: %s", strings.Join(keys, ", "))
	}
	if _, err := uuid.Parse(s); err != nil {
		return "", err
	}
	return s, nil
}

func buildContradictionDetectBody(arguments json.RawMessage) ([]byte, error) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, fmt.Errorf("memory_detect_contradictions requires memory_id and conflict_with_id")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return nil, err
	}
	a := strings.TrimSpace(firstString(m, "memory_id"))
	b := strings.TrimSpace(firstString(m, "conflict_with_id"))
	if a == "" || b == "" {
		return nil, fmt.Errorf("memory_detect_contradictions requires memory_id and conflict_with_id (UUIDs)")
	}
	if _, err := uuid.Parse(a); err != nil {
		return nil, fmt.Errorf("memory_id: %w", err)
	}
	if _, err := uuid.Parse(b); err != nil {
		return nil, fmt.Errorf("conflict_with_id: %w", err)
	}
	return json.Marshal(map[string]any{"memory_id": a, "conflict_with_id": b})
}

// evidenceAttach runs POST /v1/evidence then POST /v1/evidence/{id}/link (plaintext → base64).
func evidenceAttach(client *http.Client, base, apiKey string, arguments json.RawMessage) map[string]any {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return ToolResultErr("evidence_attach requires arguments (memory_id, evidence_text, optional kind)")
	}
	var m map[string]any
	if err := json.Unmarshal(arguments, &m); err != nil {
		return ToolResultErr(fmt.Sprintf("evidence_attach: %v", err))
	}
	mid := strings.TrimSpace(firstString(m, "memory_id"))
	txt := strings.TrimSpace(firstString(m, "evidence_text", "text", "content"))
	if mid == "" || txt == "" {
		return ToolResultErr("evidence_attach requires memory_id and evidence_text")
	}
	if _, err := uuid.Parse(mid); err != nil {
		return ToolResultErr("evidence_attach: memory_id must be a UUID")
	}
	kind := strings.TrimSpace(firstString(m, "kind"))
	if kind == "" {
		kind = "note"
	}
	b64 := base64.StdEncoding.EncodeToString([]byte(txt))
	createBody, err := json.Marshal(map[string]any{"kind": kind, "content": b64})
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req1, err := http.NewRequest(http.MethodPost, base+"/v1/evidence", bytes.NewReader(createBody))
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req1.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req1.Header.Set("X-API-Key", apiKey)
	}
	resp1, err := client.Do(req1)
	if err != nil {
		return ToolResultErr(fmt.Sprintf("http error: %v", err))
	}
	defer resp1.Body.Close()
	raw1, _ := io.ReadAll(resp1.Body)
	if resp1.StatusCode >= 400 {
		return map[string]any{
			"content": []map[string]any{
				{"type": "text", "text": fmt.Sprintf("HTTP %s\n%s", resp1.Status, string(raw1))},
			},
			"isError": true,
		}
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(raw1, &created); err != nil || strings.TrimSpace(created.ID) == "" {
		return ToolResultErr("evidence create: missing id in response")
	}
	eid := strings.TrimSpace(created.ID)
	linkBody, _ := json.Marshal(map[string]any{"memory_id": mid})
	req2, err := http.NewRequest(http.MethodPost, base+"/v1/evidence/"+url.PathEscape(eid)+"/link", bytes.NewReader(linkBody))
	if err != nil {
		return ToolResultErr(err.Error())
	}
	req2.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req2.Header.Set("X-API-Key", apiKey)
	}
	resp2, err := client.Do(req2)
	if err != nil {
		return ToolResultErr(fmt.Sprintf("link http error: %v", err))
	}
	defer resp2.Body.Close()
	raw2, _ := io.ReadAll(resp2.Body)
	text := fmt.Sprintf("evidence_id=%s\nPOST /v1/evidence (create):\n%s\n\nPOST /v1/evidence/%s/link:\nHTTP %s\n%s",
		eid, string(raw1), eid, resp2.Status, string(raw2))
	statusErr := resp2.StatusCode >= 400
	return map[string]any{
		"content": []map[string]any{
			{"type": "text", "text": text},
		},
		"isError": statusErr,
	}
}
