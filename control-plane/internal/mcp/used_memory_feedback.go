package mcp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

const usedMemoryIDsCap = 8

// parseUsedMemoryIDs extracts unique valid UUIDs from record_experience arguments.
// Invalid entries are counted as skipped. Result is capped at usedMemoryIDsCap.
func parseUsedMemoryIDs(arguments json.RawMessage) (valid []string, skipped []string) {
	if len(bytes.TrimSpace(arguments)) == 0 {
		return nil, nil
	}
	var m map[string]any
	if json.Unmarshal(arguments, &m) != nil {
		return nil, nil
	}
	raw := parseStringSliceField(m, "used_memory_ids")
	seen := map[string]struct{}{}
	for _, s := range raw {
		id := strings.TrimSpace(s)
		if id == "" {
			continue
		}
		if _, err := uuid.Parse(id); err != nil {
			skipped = append(skipped, id)
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if len(valid) >= usedMemoryIDsCap {
			skipped = append(skipped, id)
			continue
		}
		valid = append(valid, id)
	}
	return valid, skipped
}

type usedMemoryFeedbackSummary struct {
	Applied []string `json:"applied,omitempty"`
	Skipped []string `json:"skipped,omitempty"`
	Errors  []string `json:"errors,omitempty"`
}

func applyUsedMemoryIDsFeedback(client *http.Client, base, apiKey string, arguments json.RawMessage, successJSON []byte) []byte {
	valid, skipped := parseUsedMemoryIDs(arguments)
	if len(valid) == 0 && len(skipped) == 0 {
		return successJSON
	}
	corr := ""
	if len(bytes.TrimSpace(arguments)) > 0 {
		var m map[string]any
		if json.Unmarshal(arguments, &m) == nil {
			corr = strings.TrimSpace(firstString(m, "correlation_id"))
		}
	}
	sum := usedMemoryFeedbackSummary{Skipped: skipped}
	for _, id := range valid {
		if err := postMemoryHelpful(client, base, apiKey, id, corr); err != nil {
			sum.Errors = append(sum.Errors, id+": "+err.Error())
			sum.Skipped = append(sum.Skipped, id)
			continue
		}
		sum.Applied = append(sum.Applied, id)
	}
	return mergeUtilityFeedback(successJSON, sum)
}

func postMemoryHelpful(client *http.Client, base, apiKey, memoryID, correlationID string) error {
	if client == nil {
		return fmt.Errorf("no http client")
	}
	body := map[string]any{
		"event_type":  "helpful",
		"source":      "agent",
		"source_tool": "record_experience",
	}
	if correlationID != "" {
		body["correlation_id"] = correlationID
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(base, "/")+"/v1/memory/"+memoryID+"/feedback", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	return nil
}

func mergeUtilityFeedback(successJSON []byte, sum usedMemoryFeedbackSummary) []byte {
	var m map[string]any
	if json.Unmarshal(successJSON, &m) != nil {
		return successJSON
	}
	m["utility_feedback"] = sum
	b, err := json.Marshal(m)
	if err != nil {
		return successJSON
	}
	return b
}
