package recall

import (
	"context"
	"encoding/json"
)

// CompileMultiTelemetryInput carries telemetry context for compile-multi.
type CompileMultiTelemetryInput struct {
	Opts      TelemetryOptions
	Interface string
	Request   CompileMultiRequest
	Response  *CompileMultiResponse
}

func (h *Handlers) attachCompileMultiTelemetry(ctx context.Context, in CompileMultiTelemetryInput) {
	if h == nil || h.Telemetry == nil || in.Response == nil || !in.Opts.TelemetryWanted() {
		return
	}
	if len(in.Response.Bundles) == 0 {
		return
	}
	primary := in.Response.Bundles[0].Bundle
	tel, err := h.Telemetry.RecordAutoRecall(ctx, AutoRecallInput{
		SessionID:     in.Opts.TelemetrySessionIDOrSession(),
		TaskID:        in.Opts.TaskID,
		Interface:     telemetryInterfaceFromHTTP(in.Interface),
		RecallRequest: requestMapFromStruct(in.Request),
		Bundle:        &primary,
		RecallMode:    "compile_multi",
	})
	if err == nil && tel.TelemetryEnabled {
		in.Response.Telemetry = &tel
	}
}

// RunMultiTelemetryInput carries telemetry for run-multi underlying recall.
type RunMultiTelemetryInput struct {
	Opts      TelemetryOptions
	Interface string
	Request   RunMultiRequest
	Response  *RunMultiResponse
	Bundle    *RecallBundle
}

func (h *Handlers) attachRunMultiTelemetry(ctx context.Context, in RunMultiTelemetryInput) {
	if h == nil || h.Telemetry == nil || in.Response == nil || !in.Opts.TelemetryWanted() {
		return
	}
	if in.Bundle == nil {
		return
	}
	tel, err := h.Telemetry.RecordAutoRecall(ctx, AutoRecallInput{
		SessionID:     in.Opts.TelemetrySessionIDOrSession(),
		TaskID:        in.Opts.TaskID,
		Interface:     telemetryInterfaceFromHTTP(in.Interface),
		RecallRequest: requestMapFromStruct(in.Request),
		Bundle:        in.Bundle,
		RecallMode:    "run_multi",
	})
	if err == nil && tel.TelemetryEnabled {
		in.Response.Telemetry = &tel
	}
}

// TelemetryFromJSON extracts recall telemetry fields from a REST JSON response.
func TelemetryFromJSON(raw json.RawMessage) *RecallTelemetry {
	var wrap struct {
		Telemetry *RecallTelemetry `json:"telemetry"`
	}
	if err := json.Unmarshal(raw, &wrap); err != nil {
		return nil
	}
	return wrap.Telemetry
}

func bundleFromRunMultiResponse(resp *RunMultiResponse) *RecallBundle {
	if resp == nil || len(resp.MemoriesUsed) == 0 {
		return nil
	}
	b := &RecallBundle{}
	for _, id := range resp.MemoriesUsed {
		b.Experience = append(b.Experience, MemoryItem{ID: id})
	}
	return b
}
