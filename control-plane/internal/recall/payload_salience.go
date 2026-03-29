package recall

import (
	"encoding/json"
)

// PayloadDistinctContexts reads payload.salience.distinct_contexts (0 when missing).
func PayloadDistinctContexts(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	var wrap struct {
		Salience struct {
			DistinctContexts int `json:"distinct_contexts"`
		} `json:"salience"`
	}
	if json.Unmarshal(payload, &wrap) != nil {
		return 0
	}
	return wrap.Salience.DistinctContexts
}

// PayloadDistinctAgents reads payload.salience.distinct_agents (0 when missing).
func PayloadDistinctAgents(payload []byte) int {
	if len(payload) == 0 {
		return 0
	}
	var wrap struct {
		Salience struct {
			DistinctAgents int `json:"distinct_agents"`
		} `json:"salience"`
	}
	if json.Unmarshal(payload, &wrap) != nil {
		return 0
	}
	return wrap.Salience.DistinctAgents
}
