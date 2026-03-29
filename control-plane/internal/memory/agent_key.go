package memory

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// AgentUsageKey returns a short opaque hash for optional client agent_id (salience / reinforcement only).
// Empty input returns empty (caller skips agent merge). Prefix "agent:" avoids collision with context hashes.
func AgentUsageKey(agentID string) string {
	s := strings.TrimSpace(agentID)
	if s == "" {
		return ""
	}
	sum := sha256.Sum256([]byte("agent:" + s))
	return hex.EncodeToString(sum[:8])
}
