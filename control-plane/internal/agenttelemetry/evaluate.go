package agenttelemetry

import (
	"fmt"
	"strings"

	"control-plane/internal/agentobedience"
	"control-plane/internal/agentcontract"
	"control-plane/internal/recall"
)

// buildTelemetryFromPersisted constructs evaluator input from stored rows.
func buildTelemetryFromPersisted(
	sess TelemetrySession,
	recall RecallEvent,
	decisions []MemoryDecisionRow,
	output *OutputEvent,
) agentobedience.MemoryUseTelemetry {
	tel := agentobedience.MemoryUseTelemetry{
		RunID:             recall.ID.String(),
		SessionID:         sess.ID.String(),
		TaskID:            recall.TaskID,
		Interface:         recall.Interface,
		RecallRequestID:   recall.ID.String(),
		RecallBundleID:    recall.RecallBundleID,
		RecalledMemoryIDs: append([]string(nil), recall.RecalledMemoryIDs...),
	}
	if output != nil {
		tel.OutputID = output.ID.String()
		tel.FinalOutput = agentobedience.FinalOutput{
			Facts:           append([]string(nil), output.OutputFacts...),
			Actions:         append([]string(nil), output.OutputActions...),
			MemoryCitations: append([]string(nil), output.MemoryCitations...),
		}
	}
	for _, d := range decisions {
		tel.MemoryDecisions = append(tel.MemoryDecisions, agentobedience.MemoryDecision{
			MemoryID:             d.MemoryID,
			Decision:             d.Decision,
			Reason:               d.Reason,
			ContractFieldsCited:  append([]string(nil), d.ContractFieldsCited...),
			OutputFactsSupported: append([]string(nil), d.OutputFactsSupported...),
			ViolationCodes:       append([]string(nil), d.ViolationCodes...),
		})
		switch strings.ToLower(d.Decision) {
		case "used":
			tel.UsedMemoryIDs = append(tel.UsedMemoryIDs, d.MemoryID)
		case "misused":
			tel.UsedMemoryIDs = append(tel.UsedMemoryIDs, d.MemoryID)
			tel.MisusedMemoryIDs = append(tel.MisusedMemoryIDs, d.MemoryID)
		case "unsafe":
			tel.UsedMemoryIDs = append(tel.UsedMemoryIDs, d.MemoryID)
			tel.UnsafeMemoryIDs = append(tel.UnsafeMemoryIDs, d.MemoryID)
		case "ignored":
			tel.IgnoredMemoryIDs = append(tel.IgnoredMemoryIDs, d.MemoryID)
		case "historical_only":
			tel.HistoricalOnlyMemoryIDs = append(tel.HistoricalOnlyMemoryIDs, d.MemoryID)
		}
	}
	tel.TelemetryComplete = telemetryComplete(tel)
	return tel
}

func telemetryComplete(t agentobedience.MemoryUseTelemetry) bool {
	if len(t.RecalledMemoryIDs) == 0 {
		return false
	}
	if len(t.MemoryDecisions) == 0 {
		return false
	}
	if len(t.FinalOutput.Facts) == 0 && len(t.FinalOutput.Actions) == 0 {
		return false
	}
	return true
}

// bundleFromRecallJSON extracts CaseMemory slice from stored recall bundle.
func bundleFromRecallJSON(bundle map[string]any) []agentobedience.CaseMemory {
	if bundle == nil {
		return nil
	}
	items, _ := bundle["items"].([]any)
	var out []agentobedience.CaseMemory
	for _, it := range items {
		m, ok := it.(map[string]any)
		if !ok {
			continue
		}
		cm := agentobedience.CaseMemory{
			MemoryID:       strField(m, "memory_id"),
			Statement:      strField(m, "statement"),
			SchemaType:     strField(m, "schema_type"),
			LifecycleRole:  strField(m, "lifecycle_role"),
			Status:         strField(m, "status"),
			Scope:          strField(m, "scope"),
			UseInstruction: strField(m, "use_instruction"),
			MisuseWarning:  strField(m, "misuse_warning"),
			SourceType:     strField(m, "source_type"),
			AuthorityBasis: strField(m, "authority_basis"),
			QualityState:   strField(m, "quality_state"),
		}
		if ns, ok := m["negative_scope"].([]any); ok {
			for _, v := range ns {
				if s, ok := v.(string); ok {
					cm.NegativeScope = append(cm.NegativeScope, s)
				}
			}
		}
		out = append(out, cm)
	}
	return out
}

func strField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}

// validateMemoryID allows test/external IDs with prefix test: or external:
func validateMemoryID(id string) error {
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("memory_id required")
	}
	if strings.HasPrefix(id, "test:") || strings.HasPrefix(id, "external:") || strings.HasPrefix(id, "mem_") {
		return nil
	}
	if len(id) >= 8 {
		return nil
	}
	return fmt.Errorf("malformed memory_id: %s", id)
}

// recallBundleJSONFromCase builds stored recall bundle JSON from obedience memories.
func recallBundleJSONFromCase(c agentobedience.ObedienceCase) map[string]any {
	bundle := agentobedience.BundleFromCase(c)
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
	return map[string]any{"items": rawItems, "bundle_id": "bundle_" + c.TaskID}
}

func recalledIDsFromBundle(b *recall.RecallBundle) []string {
	if b == nil {
		return nil
	}
	items := agentcontract.CollectMemoryItems(b)
	ids := make([]string, 0, len(items))
	for _, it := range items {
		ids = append(ids, it.ID)
	}
	return ids
}

func validateUsedSubset(recalled, used []string) bool {
	recallSet := map[string]struct{}{}
	for _, id := range recalled {
		recallSet[id] = struct{}{}
	}
	for _, id := range used {
		if _, ok := recallSet[id]; !ok {
			return false
		}
	}
	return true
}
