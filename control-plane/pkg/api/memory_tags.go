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

// IsSharedAutomationHygieneTag reports tags shared across many automation writes.
// They must not alone drive semantic-consolidate candidate matching (ANY-tag overlap).
func IsSharedAutomationHygieneTag(tag string) bool {
	t := strings.TrimSpace(tag)
	if t == "" {
		return false
	}
	if IsEphemeralTag(t) {
		return true
	}
	switch strings.ToLower(t) {
	case TagProofScenario, TagSmokeSharedMemory:
		return true
	default:
		return false
	}
}

// SituationTagsForSemanticConsolidate returns per-run / situational tags used to scope
// semantic near-duplicate merge. Shared proof/smoke/ephemeral hygiene tags are excluded.
func SituationTagsForSemanticConsolidate(tags []string) []string {
	if len(tags) == 0 {
		return nil
	}
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		if t == "" || IsSharedAutomationHygieneTag(t) {
			continue
		}
		out = append(out, t)
	}
	return out
}

// HasDisposableAutomationTags reports doctrine disposable rows: ephemeral plus proof or smoke tag.
func HasDisposableAutomationTags(tags []string) bool {
	hasEphemeral := false
	hasProofOrSmoke := false
	for _, t := range tags {
		if IsEphemeralTag(t) {
			hasEphemeral = true
		}
		switch strings.ToLower(strings.TrimSpace(t)) {
		case TagProofScenario, TagSmokeSharedMemory:
			hasProofOrSmoke = true
		}
	}
	return hasEphemeral && hasProofOrSmoke
}
