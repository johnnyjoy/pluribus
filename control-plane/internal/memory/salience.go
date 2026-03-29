package memory

import (
	"encoding/json"
)

const maxSalienceContextHashes = 32
const maxSalienceAgentHashes = 32

// SaliencePayload is optional JSON under payload.salience for cross-context and cross-agent usage.
type SaliencePayload struct {
	DistinctContexts int      `json:"distinct_contexts"`
	ContextHashes    []string `json:"context_hashes,omitempty"`
	DistinctAgents   int      `json:"distinct_agents"`
	AgentHashes      []string `json:"agent_hashes,omitempty"`
}

// mergeSalienceForContext increments distinct_contexts when contextKey is new.
func mergeSalienceForContext(existing []byte, contextKey string) ([]byte, error) {
	return mergeSaliencePayload(existing, contextKey, "")
}

// mergeSaliencePayload updates salience for new context and/or agent hashes in one JSON round-trip.
func mergeSaliencePayload(existing []byte, contextKey, agentKey string) ([]byte, error) {
	if contextKey == "" && agentKey == "" {
		return existing, nil
	}
	root := map[string]json.RawMessage{}
	if len(existing) > 0 {
		_ = json.Unmarshal(existing, &root)
		if root == nil {
			root = map[string]json.RawMessage{}
		}
	}
	var sal SaliencePayload
	if raw, ok := root["salience"]; ok && len(raw) > 0 {
		_ = json.Unmarshal(raw, &sal)
	}
	changed := false
	if contextKey != "" {
		found := false
		for _, h := range sal.ContextHashes {
			if h == contextKey {
				found = true
				break
			}
		}
		if !found {
			sal.DistinctContexts++
			if len(sal.ContextHashes) < maxSalienceContextHashes {
				sal.ContextHashes = append(sal.ContextHashes, contextKey)
			}
			changed = true
		}
	}
	if agentKey != "" {
		found := false
		for _, h := range sal.AgentHashes {
			if h == agentKey {
				found = true
				break
			}
		}
		if !found {
			sal.DistinctAgents++
			if len(sal.AgentHashes) < maxSalienceAgentHashes {
				sal.AgentHashes = append(sal.AgentHashes, agentKey)
			}
			changed = true
		}
	}
	if !changed {
		b, err := json.Marshal(root)
		return b, err
	}
	rawSal, err := json.Marshal(sal)
	if err != nil {
		return existing, err
	}
	root["salience"] = rawSal
	return json.Marshal(root)
}
