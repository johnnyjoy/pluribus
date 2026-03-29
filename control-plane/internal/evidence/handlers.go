package evidence

import (
	"net/http"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers provides HTTP handlers for evidence.
type Handlers struct {
	Service *Service
}

// Create handles POST /v1/evidence.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec, err := h.Service.Create(r.Context(), req)
	if err != nil {
		switch err {
		case ErrContentRequired:
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		case ErrNoStorage, ErrNoRepo:
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		default:
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	httpx.WriteJSON(w, rec)
}

// ListResponseByMemory is the response for GET /v1/evidence?memory_id= (Task 79: traceability + score).
type ListResponseByMemory struct {
	Evidence      []Record `json:"evidence"`
	EvidenceScore float64  `json:"evidence_score"`
}

// List handles GET /v1/evidence?kind=... OR ?memory_id=... (traceability with evidence_score).
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	memoryIDStr := r.URL.Query().Get("memory_id")
	if memoryIDStr != "" {
		memoryID, err := uuid.Parse(memoryIDStr)
		if err != nil {
			httpx.WriteError(w, http.StatusBadRequest, "invalid memory_id")
			return
		}
		list, err := h.Service.ListEvidenceForMemory(r.Context(), memoryID)
		if err != nil {
			if err == ErrNoRepo {
				httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
				return
			}
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
			return
		}
		score, _ := h.Service.ComputeEvidenceScore(r.Context(), memoryID)
		httpx.WriteJSON(w, ListResponseByMemory{Evidence: list, EvidenceScore: score})
		return
	}
	kind := r.URL.Query().Get("kind")
	list, err := h.Service.List(r.Context(), kind)
	if err != nil {
		if err == ErrNoRepo {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, list)
}

// GetByID handles GET /v1/evidence/{id}.
func (h *Handlers) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		httpx.WriteError(w, http.StatusBadRequest, "id required")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid evidence id: expected a UUID")
		return
	}
	rec, err := h.Service.Get(r.Context(), id)
	if err != nil {
		if err == ErrNotFound {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, rec)
}

// LinkRequest is the body for POST /v1/evidence/{id}/link.
type LinkRequest struct {
	MemoryID string `json:"memory_id"`
}

// Link handles POST /v1/evidence/{id}/link with body {"memory_id": "uuid"}.
func (h *Handlers) Link(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		httpx.WriteError(w, http.StatusBadRequest, "id required")
		return
	}
	evidenceID, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid evidence id: expected a UUID")
		return
	}
	var req LinkRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.MemoryID == "" {
		httpx.WriteError(w, http.StatusBadRequest, "memory_id required")
		return
	}
	memoryID, err := uuid.Parse(req.MemoryID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory_id")
		return
	}
	if err := h.Service.LinkEvidenceToMemory(r.Context(), memoryID, evidenceID); err != nil {
		if err == ErrLinkIDsRequired || err == ErrNoRepo {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
