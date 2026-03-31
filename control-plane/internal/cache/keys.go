package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// EvidenceBundleKey is a fingerprint for recall bundle cache keys when evidence-in-bundle is configured.
// Zero value means evidence-in-bundle is off (backward compatible with older cache entries only if server still uses zero defaults).
type EvidenceBundleKey struct {
	Enabled         bool
	MaxPerMemory    int
	MaxPerBundle    int
	SummaryMaxChars int
}

// RecallBundleKey returns a cache key for a recall compile result.
// Key format: recall:bundle:{hash} so the same request hits the same entry.
// symbols and LSP focus fields are included so different symbol sets / files do not collide.
// evidence includes bounded supporting-evidence mode so bundles with vs without receipts do not collide.
func RecallBundleKey(tags []string, maxPerKind, maxTotal, maxTokens int, retrievalQuery, proposalText string, symbols []string, repoRoot, lspFocusPath string, lspFocusLine, lspFocusColumn int, correlationID string, evidence EvidenceBundleKey) string {
	symCopy := append([]string(nil), symbols...)
	sort.Strings(symCopy)
	h := sha256.New()
	enc := json.NewEncoder(h)
	_ = enc.Encode(map[string]interface{}{
		"tags":             tags,
		"retrieval_query":  retrievalQuery,
		"proposal_text":    proposalText,
		"max_per_kind":     maxPerKind,
		"max_total":        maxTotal,
		"max_tokens":       maxTokens,
		"symbols":          symCopy,
		"repo_root":        repoRoot,
		"lsp_focus_path":   lspFocusPath,
		"lsp_focus_line":   lspFocusLine,
		"lsp_focus_column": lspFocusColumn,
		"correlation_id":   correlationID,
		"evidence_in_bundle": map[string]interface{}{
			"enabled":           evidence.Enabled,
			"max_per_memory":    evidence.MaxPerMemory,
			"max_per_bundle":    evidence.MaxPerBundle,
			"summary_max_chars": evidence.SummaryMaxChars,
		},
	})
	return "recall:bundle:" + hex.EncodeToString(h.Sum(nil))
}

// PreflightKey returns a cache key for a preflight result.
func PreflightKey(changedFiles int, tags []string) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	_ = enc.Encode(map[string]interface{}{
		"changed_files": changedFiles,
		"tags":          tags,
	})
	return "preflight:" + hex.EncodeToString(h.Sum(nil))
}

// MemorySearchKey returns a cache key for memory search results (tag-index cache).
// Search is not partitioned by project; key uses tags + status + max + optional kind filter.
func MemorySearchKey(tags []string, status string, max int, kinds []string) string {
	h := sha256.New()
	enc := json.NewEncoder(h)
	_ = enc.Encode(map[string]interface{}{
		"tags":   tags,
		"status": status,
		"max":    max,
		"kinds":  kinds,
	})
	return "memory:tags:" + hex.EncodeToString(h.Sum(nil))
}
