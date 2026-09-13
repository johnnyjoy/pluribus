package mcp

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"
)

const activityLaneCap = 8

func wantsActivityLane(strategy string, occurredAfter, occurredBefore string) bool {
	if strings.TrimSpace(occurredAfter) != "" || strings.TrimSpace(occurredBefore) != "" {
		return true
	}
	return strategy == ctxStrategyEpisodic
}

func fetchActivityLaneText(client *http.Client, base, apiKey, query, occurredAfter, occurredBefore string) string {
	if client == nil {
		return ""
	}
	body := map[string]any{
		"query":       query,
		"max_results": activityLaneCap,
	}
	if strings.TrimSpace(occurredAfter) != "" {
		body["occurred_after"] = strings.TrimSpace(occurredAfter)
	}
	if strings.TrimSpace(occurredBefore) != "" {
		body["occurred_before"] = strings.TrimSpace(occurredBefore)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return ""
	}
	req, err := http.NewRequest(http.MethodPost, strings.TrimRight(base, "/")+"/v1/advisory-episodes/similar", bytes.NewReader(raw))
	if err != nil {
		return ""
	}
	req.Header.Set("Content-Type", "application/json")
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
	payload, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return formatActivityLane(payload)
}

func formatActivityLane(payload []byte) string {
	var wrap struct {
		Cases []struct {
			ID        string     `json:"id"`
			Summary   string     `json:"summary"`
			Occurred  *time.Time `json:"occurred_at"`
			CreatedAt time.Time  `json:"created_at"`
		} `json:"advisory_similar_cases"`
	}
	if json.Unmarshal(payload, &wrap) != nil || len(wrap.Cases) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("Activity:\n")
	for i, c := range wrap.Cases {
		if i >= activityLaneCap {
			break
		}
		sum := strings.TrimSpace(c.Summary)
		if sum == "" {
			continue
		}
		when := ""
		if c.Occurred != nil {
			when = c.Occurred.UTC().Format(time.RFC3339)
		} else if !c.CreatedAt.IsZero() {
			when = c.CreatedAt.UTC().Format(time.RFC3339)
		}
		sb.WriteString("- ")
		if id := strings.TrimSpace(c.ID); id != "" {
			sb.WriteByte('[')
			sb.WriteString(id)
			sb.WriteString("] ")
		}
		if when != "" {
			sb.WriteString(when)
			sb.WriteString(" ")
		}
		sb.WriteString(sum)
		sb.WriteByte('\n')
	}
	return strings.TrimSpace(sb.String())
}

func appendActivityLane(text, activity string) string {
	activity = strings.TrimSpace(activity)
	if activity == "" {
		return text
	}
	if strings.TrimSpace(text) == "" {
		return activity
	}
	return strings.TrimSpace(text) + "\n\n" + activity
}

func activityLaneBounds(arguments json.RawMessage) (after, before string) {
	var m map[string]any
	if json.Unmarshal(arguments, &m) != nil {
		return "", ""
	}
	return strings.TrimSpace(firstString(m, "occurred_after")), strings.TrimSpace(firstString(m, "occurred_before"))
}

func activityQueryFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if q, ok := meta["retrieval_query"].(string); ok {
		return strings.TrimSpace(q)
	}
	return ""
}

func activityStrategyFromMeta(meta map[string]any) string {
	if meta == nil {
		return ""
	}
	if s, ok := meta["strategy"].(string); ok {
		return strings.TrimSpace(s)
	}
	return ""
}
