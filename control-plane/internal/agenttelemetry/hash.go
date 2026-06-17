package agenttelemetry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"

	"control-plane/internal/agentcontract"
	"control-plane/internal/recall"
)

// HashRecallRequest returns a stable hash for idempotency and correlation.
func HashRecallRequest(sessionID string, req map[string]any) string {
	payload := map[string]any{
		"session_id": strings.TrimSpace(sessionID),
		"request":    canonicalizeMap(req),
	}
	raw, _ := json.Marshal(payload)
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:16])
}

// StableBundleID returns a deterministic bundle id from bundle JSON content.
func StableBundleID(bundle map[string]any) string {
	raw, _ := json.Marshal(canonicalizeMap(bundle))
	sum := sha256.Sum256(raw)
	return "bundle_" + hex.EncodeToString(sum[:16])
}

func canonicalizeMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	keys := make([]string, 0, len(m))
	for k := range m {
		if strings.HasPrefix(k, "_") {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		out[k] = canonicalizeValue(m[k])
	}
	return out
}

func canonicalizeValue(v any) any {
	switch x := v.(type) {
	case map[string]any:
		return canonicalizeMap(x)
	case []any:
		cp := make([]any, len(x))
		for i, e := range x {
			cp[i] = canonicalizeValue(e)
		}
		return cp
	case []string:
		cp := append([]string(nil), x...)
		sort.Strings(cp)
		return cp
	default:
		return v
	}
}

// BundleJSONFromRecall encodes a recall bundle for telemetry persistence.
func BundleJSONFromRecall(bundle *recall.RecallBundle) map[string]any {
	if bundle == nil {
		return map[string]any{"items": []any{}}
	}
	items := agentcontract.CollectMemoryItems(bundle)
	var rawItems []map[string]any
	for _, it := range items {
		rawItems = append(rawItems, map[string]any{
			"memory_id":       it.ID,
			"statement":       it.Statement,
			"schema_type":     it.SchemaType,
			"lifecycle_role":  it.LifecycleRole,
			"status":          it.Status,
			"scope":           it.Scope,
			"negative_scope":  it.NegativeScope,
			"use_instruction": it.UseInstruction,
			"misuse_warning":  it.MisuseWarning,
			"source_type":     it.SourceType,
			"authority_basis": it.AuthorityBasis,
			"superseded_by":   it.SupersededBy,
			"quality_state":   it.QualityState,
			"quality_score":   it.QualityScore,
		})
	}
	out := map[string]any{"items": rawItems}
	out["bundle_id"] = StableBundleID(out)
	return out
}

// BundleJSONFromWakeup encodes wakeup response memories for telemetry persistence.
func BundleJSONFromWakeup(w *recall.WakeupResponse) map[string]any {
	if w == nil {
		return map[string]any{"items": []any{}}
	}
	var rawItems []map[string]any
	appendItems := func(items []recall.MemoryItem) {
		for _, it := range items {
			rawItems = append(rawItems, map[string]any{
				"memory_id":      it.ID,
				"statement":      it.Statement,
				"schema_type":    it.SchemaType,
				"lifecycle_role": it.LifecycleRole,
				"status":         it.Status,
				"scope":          it.Scope,
				"quality_state":  it.QualityState,
			})
		}
	}
	appendItems(w.Identity)
	appendItems(w.GoverningMemory)
	out := map[string]any{"items": rawItems, "wakeup": true}
	out["bundle_id"] = StableBundleID(out)
	return out
}

// RecalledIDsFromBundleJSON extracts memory ids from persisted bundle JSON.
func RecalledIDsFromBundleJSON(bundle map[string]any) []string {
	if bundle == nil {
		return nil
	}
	items, _ := bundle["items"].([]any)
	var ids []string
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		if id, ok := m["memory_id"].(string); ok && strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	return ids
}
