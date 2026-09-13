package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
)

func ensureDefaultAgentID(toolName string, arguments json.RawMessage, clientName string) json.RawMessage {
	clientName = strings.TrimSpace(clientName)
	if clientName == "" {
		return arguments
	}
	switch toolName {
	case "record_experience", "mcp_episode_ingest", "resolve_chore", "memory_create", "memory_feedback":
	default:
		return arguments
	}
	var m map[string]any
	if len(bytes.TrimSpace(arguments)) == 0 {
		m = map[string]any{}
	} else if err := json.Unmarshal(arguments, &m); err != nil {
		return arguments
	}
	if aid, _ := m["agent_id"].(string); strings.TrimSpace(aid) != "" {
		return arguments
	}
	m["agent_id"] = clientName
	b, err := json.Marshal(m)
	if err != nil {
		return arguments
	}
	return b
}
