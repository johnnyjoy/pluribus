package recall

import (
	"net/http"
	"strings"

	"control-plane/internal/httpx"
)

// Handlers provides HTTP handlers for recall compile and preflight.
type Handlers struct {
	Service   *Service
	Telemetry TelemetryRecorder
}

// Compile handles POST /v1/recall/compile.
func (h *Handlers) Compile(w http.ResponseWriter, r *http.Request) {
	var req CompileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	var bundle *RecallBundle
	var err error
	if req.EnableTriggeredRecall {
		bundle, _, err = h.Service.CompileTriggered(r.Context(), req)
	} else {
		bundle, err = h.Service.Compile(r.Context(), req)
	}
	if err != nil {
		if err == ErrNoCompiler {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	mode := strings.TrimSpace(req.RecallMode)
	if mode == "" {
		mode = "current"
	}
	h.attachBundleTelemetry(r.Context(), req.TelemetryOptions, "rest", bundle, requestMapFromStruct(req), mode)
	httpx.WriteJSON(w, bundle)
}

// Wakeup handles POST /v1/recall/wakeup (L0/L1 session-start view over the same compiler and memory semantics).
func (h *Handlers) Wakeup(w http.ResponseWriter, r *http.Request) {
	var req WakeupRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	out, err := h.Service.Wakeup(r.Context(), req)
	if err != nil {
		if err == ErrNoCompiler {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.attachWakeupTelemetry(r.Context(), req.TelemetryOptions, "rest", out, requestMapFromStruct(req))
	httpx.WriteJSON(w, out)
}

// Preflight handles POST /v1/recall/preflight.
func (h *Handlers) Preflight(w http.ResponseWriter, r *http.Request) {
	var req PreflightRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result := h.Service.Preflight(r.Context(), req)
	httpx.WriteJSON(w, result)
}

// CompileMulti handles POST /v1/recall/compile-multi.
func (h *Handlers) CompileMulti(w http.ResponseWriter, r *http.Request) {
	var req CompileMultiRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.CompileMulti(r.Context(), req)
	if err != nil {
		if err == ErrNoCompiler {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.attachCompileMultiTelemetry(r.Context(), CompileMultiTelemetryInput{
		Opts: req.TelemetryOptions, Interface: "rest", Request: req, Response: resp,
	})
	httpx.WriteJSON(w, resp)
}

// RunMulti handles POST /v1/recall/run-multi (Pluribus Phase A contract).
func (h *Handlers) RunMulti(w http.ResponseWriter, r *http.Request) {
	var req RunMultiRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.RunMulti(r.Context(), req)
	if err != nil {
		if err == ErrRunMultiNotConfigured {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	h.attachRunMultiTelemetry(r.Context(), RunMultiTelemetryInput{
		Opts: req.TelemetryOptions, Interface: "rest", Request: req, Response: resp,
		Bundle: bundleFromRunMultiResponse(resp),
	})
	httpx.WriteJSON(w, resp)
}
