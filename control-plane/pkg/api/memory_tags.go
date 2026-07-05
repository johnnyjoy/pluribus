package api

import "strings"

// Ephemeral and proof tags mark disposable rows (CI receipts, install smoke).
// They are not durable-historical tags; TTL expiration may archive them when eligible.
const (
	TagEphemeral         = "ephemeral"
	TagProofScenario     = "proof-scenario"
	TagSmokeSharedMemory = "smoke-shared-memory"
)

// IsEphemeralTag reports whether tag marks a row as disposable (ephemeral or ephemeral:*).
func IsEphemeralTag(tag string) bool {
	t := strings.ToLower(strings.TrimSpace(tag))
	if t == TagEphemeral {
		return true
	}
	return strings.HasPrefix(t, "ephemeral:")
}
