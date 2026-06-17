package utilitypolicy

import (
	"encoding/json"
	"errors"
	"net/http"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers exposes /v1/agent/utility/policy/* REST API.
type Handlers struct {
	Service *Service
}

func writePolicyErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrCandidateNotFound), errors.Is(err, ErrApplicationNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrAlreadyApplied), errors.Is(err, ErrAlreadyReverted):
		httpx.WriteError(w, http.StatusConflict, err.Error())
	case errors.Is(err, ErrSessionCapExceeded), errors.Is(err, ErrAgentCapExceeded),
		errors.Is(err, ErrStaleCandidate), errors.Is(err, ErrTamperedCandidate):
		httpx.WriteError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, ErrNoService):
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
	default:
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	}
}

func (h *Handlers) EvaluateCandidate(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	var req EvaluateCandidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	dec, err := h.Service.EvaluateCandidate(r.Context(), req.CandidateID, false)
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSON(w, dec)
}

func (h *Handlers) ApplyCandidate(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	var req ApplyCandidateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	rec, err := h.Service.ApplyCandidate(r.Context(), req.CandidateID, req.AppliedBy, false)
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, rec)
}

func (h *Handlers) ApplyBatch(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	var req ApplyBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	recs, err := h.Service.ApplyBatch(r.Context(), req.CandidateIDs, req.AppliedBy)
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, map[string]any{"applications": recs})
}

func (h *Handlers) RevertApplication(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	var req RevertApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid json")
		return
	}
	rec, err := h.Service.RevertApplication(r.Context(), req.RollbackToken, req.RevertReason, req.AppliedBy)
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSON(w, rec)
}

func (h *Handlers) GetCandidate(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "candidate_id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid candidate_id")
		return
	}
	rec, err := h.Service.GetCandidateApplication(r.Context(), id)
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSON(w, rec)
}

func (h *Handlers) GetMemoryHistory(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	memoryID := chi.URLParam(r, "memory_id")
	rows, err := h.Service.ListMemoryApplications(r.Context(), memoryID)
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSON(w, map[string]any{"applications": rows})
}

func (h *Handlers) ListApplications(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	rows, err := h.Service.ListApplications(r.Context())
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSON(w, map[string]any{"applications": rows})
}

func (h *Handlers) Summary(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "utility policy unavailable")
		return
	}
	sum, err := h.Service.Summary(r.Context())
	if err != nil {
		writePolicyErr(w, err)
		return
	}
	httpx.WriteJSON(w, sum)
}

// MountRoutes registers policy routes on a chi router (for tests).
func MountRoutes(r chi.Router, h *Handlers) {
	r.Route("/agent/utility/policy", func(r chi.Router) {
		r.Post("/evaluate-candidate", h.EvaluateCandidate)
		r.Post("/apply-candidate", h.ApplyCandidate)
		r.Post("/apply-batch", h.ApplyBatch)
		r.Post("/revert-application", h.RevertApplication)
		r.Get("/candidate/{candidate_id}", h.GetCandidate)
		r.Get("/memory/{memory_id}", h.GetMemoryHistory)
		r.Get("/applications", h.ListApplications)
		r.Get("/summary", h.Summary)
	})
}
