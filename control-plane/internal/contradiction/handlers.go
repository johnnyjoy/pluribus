package contradiction

import (
	"net/http"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers provides HTTP handlers for contradiction records (Task 78).
type Handlers struct {
	Service *Service
}

// Create handles POST /v1/contradictions.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec, err := h.Service.Create(r.Context(), req)
	if err != nil {
		if err == ErrSelfContradiction {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, rec)
}

// List handles GET /v1/contradictions (query: resolution_state, memory_id, limit).
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	req := ListRequest{
		ResolutionState: r.URL.Query().Get("resolution_state"),
		Limit:           50,
	}
	if mid := r.URL.Query().Get("memory_id"); mid != "" {
		if id, err := uuid.Parse(mid); err == nil {
			req.MemoryID = id
		}
	}
	list, err := h.Service.List(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Record{}
	}
	httpx.WriteJSON(w, list)
}

// GetByID handles GET /v1/contradictions/{id}.
func (h *Handlers) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid contradiction id: expected a UUID")
		return
	}
	rec, err := h.Service.GetByID(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rec == nil {
		httpx.WriteError(w, http.StatusNotFound, "contradiction not found")
		return
	}
	httpx.WriteJSON(w, rec)
}

// UpdateResolution handles PATCH /v1/contradictions/{id}/resolution.
func (h *Handlers) UpdateResolution(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid contradiction id: expected a UUID")
		return
	}
	var body UpdateResolutionRequest
	if err := httpx.DecodeJSON(r, &body); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := h.Service.UpdateResolution(r.Context(), id, body.ResolutionState); err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	rec, _ := h.Service.GetByID(r.Context(), id)
	httpx.WriteJSON(w, rec)
}

// DetectAndRecordRequest is the body for POST /v1/contradictions/detect.
type DetectAndRecordRequest struct {
	MemoryID      uuid.UUID `json:"memory_id"`
	ConflictWithID uuid.UUID `json:"conflict_with_id"`
}

// DetectAndRecord handles POST /v1/contradictions/detect. If conflict is detected (attribute overlap), creates an unresolved record and returns it; otherwise 204.
func (h *Handlers) DetectAndRecord(w http.ResponseWriter, r *http.Request) {
	var req DetectAndRecordRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec, err := h.Service.DetectAndRecord(r.Context(), req.MemoryID, req.ConflictWithID)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if rec == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	httpx.WriteJSON(w, rec)
}
