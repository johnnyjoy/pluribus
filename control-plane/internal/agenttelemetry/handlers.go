package agenttelemetry

import (
	"encoding/json"
	"errors"
	"net/http"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
)

// Handlers exposes /v1/agent/telemetry/* REST API.
type Handlers struct {
	Service *Service
}

func (h *Handlers) StartSession(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	var req StartSessionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	sess, err := h.Service.StartSession(r.Context(), req)
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, sess)
}

func (h *Handlers) RecordRecall(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	var req RecordRecallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	ev, err := h.Service.RecordRecall(r.Context(), req)
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, ev)
}

func (h *Handlers) RecordDecision(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	var req RecordDecisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	rows, err := h.Service.RecordDecision(r.Context(), req)
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, map[string]any{"decisions": rows})
}

func (h *Handlers) RecordOutput(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	var req RecordOutputRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	out, err := h.Service.RecordOutput(r.Context(), req)
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, out)
}

func (h *Handlers) Evaluate(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	var req EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	resp, err := h.Service.Evaluate(r.Context(), req)
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, resp)
}

func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	sum, err := h.Service.GetSessionSummary(r.Context(), chi.URLParam(r, "session_id"))
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, sum)
}

func (h *Handlers) GetMemory(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	sum, err := h.Service.GetMemorySummary(r.Context(), chi.URLParam(r, "memory_id"))
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, sum)
}

func (h *Handlers) ListViolations(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	rows, err := h.Service.ListViolations(r.Context(), r.URL.Query().Get("memory_id"), r.URL.Query().Get("violation_code"))
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	if rows == nil {
		rows = []ViolationRow{}
	}
	httpx.WriteJSONStatus(w, http.StatusOK, map[string]any{"violations": rows})
}

func (h *Handlers) ListUtilityCandidates(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "telemetry unavailable")
		return
	}
	rows, err := h.Service.ListUtilityCandidates(r.Context(), r.URL.Query().Get("memory_id"))
	if err != nil {
		writeTelemetryErr(w, err)
		return
	}
	if rows == nil {
		rows = []UtilityCandidate{}
	}
	httpx.WriteJSONStatus(w, http.StatusOK, map[string]any{"utility_candidates": rows})
}

func writeTelemetryErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrUnknownSession), errors.Is(err, ErrUnknownRecall), errors.Is(err, ErrUnknownOutput):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrMalformedPayload):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		if errors.Is(err, ErrSelfReportBypass) {
			httpx.WriteError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	}
}
