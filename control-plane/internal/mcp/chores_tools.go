package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"control-plane/internal/chores"
)

// fetchHousekeepingLine asks the REST API for the oldest open curation chore
// and formats one optional maintenance ask for the visiting agent. Best-effort:
// any failure (endpoint missing, no chores, bad JSON) returns "".
func fetchHousekeepingLine(client *http.Client, base, apiKey string) string {
	req, err := http.NewRequest(http.MethodGet, base+"/v1/curation/chores?limit=1", nil)
	if err != nil {
		return ""
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return ""
	}
	rawBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return ""
	}
	var lr chores.ListResponse
	if err := json.Unmarshal(rawBody, &lr); err != nil || len(lr.Chores) == 0 {
		return ""
	}
	return chores.HousekeepingLine(&lr.Chores[0])
}

// toolResultWakeupWithHousekeeping wraps a successful wakeup REST body and
// appends the optional housekeeping line (one open chore, if any).
func toolResultWakeupWithHousekeeping(client *http.Client, base, apiKey string, rawBody []byte) map[string]any {
	var payload json.RawMessage
	if err := json.Unmarshal(rawBody, &payload); err != nil {
		return ToolResultErr("wakeup_context: invalid JSON response")
	}
	wrap := map[string]any{"wakeup_response": payload}
	if line := fetchHousekeepingLine(client, base, apiKey); line != "" {
		wrap["housekeeping"] = line
	}
	return toolResultStructuredJSON(wrap)
}

// parseOptionalChoreLimit reads the optional limit argument for list_chores (default 20).
func parseOptionalChoreLimit(arguments json.RawMessage) int {
	limit := 20
	if len(bytes.TrimSpace(arguments)) == 0 {
		return limit
	}
	var args struct {
		Limit int `json:"limit"`
	}
	if err := json.Unmarshal(arguments, &args); err == nil && args.Limit > 0 {
		limit = args.Limit
	}
	return limit
}

// choreResolveBody is the exact REST body for POST /v1/curation/chores/{id}/resolve
// (the endpoint rejects unknown fields, so chore_id/id aliases are stripped here).
type choreResolveBody struct {
	AgentID string `json:"agent_id"`
	Action  string `json:"action"`
	Reason  string `json:"reason,omitempty"`
}

// parseChoreResolveArgs extracts chore_id (or id), action, agent_id, and
// optional reason for the resolve_chore tool.
func parseChoreResolveArgs(arguments json.RawMessage) (choreID string, body []byte, err error) {
	var args struct {
		ChoreID string `json:"chore_id"`
		ID      string `json:"id"`
		Action  string `json:"action"`
		AgentID string `json:"agent_id"`
		Reason  string `json:"reason"`
	}
	if len(bytes.TrimSpace(arguments)) == 0 {
		return "", nil, fmt.Errorf("resolve_chore requires arguments (chore_id, action, agent_id)")
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return "", nil, fmt.Errorf("resolve_chore: invalid arguments: %w", err)
	}
	choreID = strings.TrimSpace(args.ChoreID)
	if choreID == "" {
		choreID = strings.TrimSpace(args.ID)
	}
	if choreID == "" {
		return "", nil, fmt.Errorf("resolve_chore requires chore_id")
	}
	if strings.TrimSpace(args.Action) == "" {
		return "", nil, fmt.Errorf("resolve_chore requires action")
	}
	if strings.TrimSpace(args.AgentID) == "" {
		return "", nil, fmt.Errorf("resolve_chore requires agent_id (corroboration counts distinct agents)")
	}
	b, err := json.Marshal(choreResolveBody{
		AgentID: strings.TrimSpace(args.AgentID),
		Action:  strings.TrimSpace(args.Action),
		Reason:  strings.TrimSpace(args.Reason),
	})
	if err != nil {
		return "", nil, err
	}
	return choreID, b, nil
}
