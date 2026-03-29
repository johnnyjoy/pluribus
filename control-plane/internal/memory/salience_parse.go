package memory

import (
	"encoding/json"
)

// SalienceDistinctCounts reads payload.salience distinct_contexts and distinct_agents (0 when missing).
func SalienceDistinctCounts(payload []byte) (contexts, agents int) {
	if len(payload) == 0 {
		return 0, 0
	}
	var w struct {
		Salience struct {
			DistinctContexts int `json:"distinct_contexts"`
			DistinctAgents   int `json:"distinct_agents"`
		} `json:"salience"`
	}
	if json.Unmarshal(payload, &w) != nil {
		return 0, 0
	}
	return w.Salience.DistinctContexts, w.Salience.DistinctAgents
}
