package mcp

import (
	"encoding/json"
)

// mergeRecallTelemetryIntoMCPContext copies REST telemetry correlation fields into mcp_context.
func mergeRecallTelemetryIntoMCPContext(meta map[string]any, bundle json.RawMessage) {
	if meta == nil || len(bundle) == 0 {
		return
	}
	var wrap struct {
		Telemetry *struct {
			TelemetryEnabled   bool   `json:"telemetry_enabled"`
			TelemetrySessionID string `json:"telemetry_session_id"`
			RecallEventID      string `json:"recall_event_id"`
			RecallBundleID     string `json:"recall_bundle_id"`
			RecallRequestHash  string `json:"recall_request_hash"`
		} `json:"telemetry"`
	}
	if err := json.Unmarshal(bundle, &wrap); err != nil || wrap.Telemetry == nil || !wrap.Telemetry.TelemetryEnabled {
		return
	}
	meta["telemetry_enabled"] = true
	meta["telemetry_session_id"] = wrap.Telemetry.TelemetrySessionID
	meta["recall_event_id"] = wrap.Telemetry.RecallEventID
	meta["recall_bundle_id"] = wrap.Telemetry.RecallBundleID
	meta["recall_request_hash"] = wrap.Telemetry.RecallRequestHash
}
