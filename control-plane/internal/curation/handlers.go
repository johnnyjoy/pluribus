package curation

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"control-plane/internal/httpx"
)

// Handlers provides HTTP handlers for curation.
type Handlers struct {
	Service *Service
}

// Digest handles POST /v1/curation/digest.
func (h *Handlers) Digest(w http.ResponseWriter, r *http.Request) {
	var req DigestRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.Service.Digest(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, result)
}

// Materialize handles POST /v1/curation/candidates/{id}/materialize — creates memory from proposal_json.
func (h *Handlers) Materialize(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid candidate id: expected a UUID")
		return
	}
	obj, err := h.Service.Materialize(r.Context(), id)
	if err != nil {
		if err.Error() == "candidate not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, obj)
}

// Evaluate handles POST /v1/curation/evaluate.
func (h *Handlers) Evaluate(w http.ResponseWriter, r *http.Request) {
	var req EvaluateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.Service.Evaluate(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, result)
}

// ListPending handles GET /v1/curation/pending (global pending queue).
func (h *Handlers) ListPending(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusInternalServerError, "curation service not configured")
		return
	}
	list, err := h.Service.ListPending(r.Context())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []CandidateEvent{}
	}
	httpx.WriteJSON(w, list)
}

// MarkPromoted handles POST /v1/curation/candidates/{id}/promote.
// If the request body contains a valid PromoteToPatternRequest (payload), creates a pattern memory and then marks promoted.
// If the body is empty or invalid, only marks the candidate as promoted (existing behavior).
func (h *Handlers) MarkPromoted(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid candidate id: expected a UUID")
		return
	}
	body, _ := io.ReadAll(r.Body)
	if len(body) > 0 {
		var promoteReq PromoteToPatternRequest
		if json.Unmarshal(body, &promoteReq) == nil && promoteReq.Payload.Polarity != "" {
			if h.Service.Memory == nil {
				httpx.WriteError(w, http.StatusServiceUnavailable, "promote to pattern not configured")
				return
			}
			obj, err := h.Service.PromoteToPattern(r.Context(), id, &promoteReq.Payload)
			if err != nil {
				if err.Error() == "candidate not found" {
					httpx.WriteError(w, http.StatusNotFound, err.Error())
					return
				}
				httpx.WriteError(w, http.StatusBadRequest, err.Error())
				return
			}
			w.WriteHeader(http.StatusCreated)
			httpx.WriteJSON(w, obj)
			return
		}
	}
	if err := h.Service.MarkPromoted(r.Context(), id); err != nil {
		if err.Error() == "candidate not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, map[string]string{"status": "promoted"})
}

// MarkRejected handles POST /v1/curation/candidates/{id}/reject.
func (h *Handlers) MarkRejected(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid candidate id: expected a UUID")
		return
	}
	if err := h.Service.MarkRejected(r.Context(), id); err != nil {
		if err.Error() == "candidate not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, map[string]string{"status": "rejected"})
}
