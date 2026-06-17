package recall

import (
	"context"
	"encoding/json"
	"strings"
)

func requestMapFromStruct(v any) map[string]any {
	raw, err := json.Marshal(v)
	if err != nil {
		return map[string]any{}
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return map[string]any{}
	}
	return m
}

func telemetryInterfaceFromHTTP(iface string) string {
	if strings.TrimSpace(iface) != "" {
		return iface
	}
	return "rest"
}

func (h *Handlers) attachBundleTelemetry(ctx context.Context, opts TelemetryOptions, iface string, bundle *RecallBundle, reqMap map[string]any, recallMode string) {
	if h == nil || h.Telemetry == nil || bundle == nil || !opts.TelemetryWanted() {
		return
	}
	sid := opts.TelemetrySessionIDOrSession()
	tel, err := h.Telemetry.RecordAutoRecall(ctx, AutoRecallInput{
		SessionID:     sid,
		TaskID:        strings.TrimSpace(opts.TaskID),
		Interface:     telemetryInterfaceFromHTTP(iface),
		RecallRequest: reqMap,
		Bundle:        bundle,
		RecallMode:    recallMode,
	})
	if err == nil {
		AttachTelemetry(bundle, tel)
	}
}

func (h *Handlers) attachWakeupTelemetry(ctx context.Context, opts TelemetryOptions, iface string, resp *WakeupResponse, reqMap map[string]any) {
	if h == nil || h.Telemetry == nil || resp == nil || !opts.TelemetryWanted() {
		return
	}
	sid := opts.TelemetrySessionIDOrSession()
	tel, err := h.Telemetry.RecordAutoRecall(ctx, AutoRecallInput{
		SessionID:     sid,
		TaskID:        strings.TrimSpace(opts.TaskID),
		Interface:     telemetryInterfaceFromHTTP(iface),
		RecallRequest: reqMap,
		Wakeup:        resp,
		RecallMode:    "wakeup",
	})
	if err == nil {
		AttachWakeupTelemetry(resp, tel)
	}
}

func telemetryOptionsFromQuery(q map[string][]string) TelemetryOptions {
	get := func(k string) string {
		if v, ok := q[k]; ok && len(v) > 0 {
			return strings.TrimSpace(v[0])
		}
		return ""
	}
	opts := TelemetryOptions{
		TelemetrySessionID: get("telemetry_session_id"),
		SessionID:          get("session_id"),
		TaskID:             get("task_id"),
	}
	if v := get("telemetry"); v != "" {
		b := strings.EqualFold(v, "true") || v == "1"
		opts.Telemetry = &b
	}
	return opts
}
