package recall

import (
	"context"
	"encoding/json"
	"strings"
)

// RecallTelemetry carries correlation IDs for automatic recall telemetry (Phase 11J).
type RecallTelemetry struct {
	TelemetryEnabled   bool   `json:"telemetry_enabled"`
	TelemetrySessionID string `json:"telemetry_session_id,omitempty"`
	RecallEventID      string `json:"recall_event_id,omitempty"`
	RecallBundleID     string `json:"recall_bundle_id,omitempty"`
	RecallRequestHash  string `json:"recall_request_hash,omitempty"`
}

// TelemetryOptions are optional fields on recall requests that enable automatic telemetry.
type TelemetryOptions struct {
	TelemetrySessionID string `json:"telemetry_session_id,omitempty"`
	SessionID          string `json:"session_id,omitempty"`
	TaskID             string `json:"telemetry_work_ref,omitempty"`
	// Telemetry when true forces telemetry on; when false forces off. Empty = auto when session supplied.
	Telemetry *bool `json:"telemetry,omitempty"`
}

// TelemetrySessionID resolves telemetry session from request fields.
func (o TelemetryOptions) TelemetrySessionIDOrSession() string {
	if s := strings.TrimSpace(o.TelemetrySessionID); s != "" {
		return s
	}
	return strings.TrimSpace(o.SessionID)
}

// TelemetryWanted reports whether automatic recall telemetry should run.
func (o TelemetryOptions) TelemetryWanted() bool {
	if o.Telemetry != nil {
		return *o.Telemetry
	}
	return o.TelemetrySessionIDOrSession() != ""
}

// AutoRecallInput is passed to TelemetryRecorder after a successful recall.
type AutoRecallInput struct {
	SessionID     string
	TaskID        string
	Interface     string
	RecallRequest map[string]any
	Bundle        *RecallBundle
	RecallMode    string
	Wakeup        *WakeupResponse
}

// TelemetryRecorder persists automatic recall exposure events.
type TelemetryRecorder interface {
	RecordAutoRecall(ctx context.Context, in AutoRecallInput) (RecallTelemetry, error)
}

// AttachTelemetry sets telemetry on a recall bundle response.
func AttachTelemetry(bundle *RecallBundle, tel RecallTelemetry) {
	if bundle == nil {
		return
	}
	if tel.TelemetryEnabled {
		bundle.Telemetry = &tel
	}
}

// AttachWakeupTelemetry sets telemetry on wakeup response.
func AttachWakeupTelemetry(resp *WakeupResponse, tel RecallTelemetry) {
	if resp == nil {
		return
	}
	if tel.TelemetryEnabled {
		resp.Telemetry = &tel
	}
}

// RequestMapFromJSON decodes a recall request body into a map for hashing/persistence.
func RequestMapFromJSON(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	return m
}
