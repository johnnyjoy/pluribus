package ingest

import (
	"errors"
	"net/http"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers serves POST /v1/ingest/cognition.
type Handlers struct {
	Service *Service
}

// Cognition handles POST /v1/ingest/cognition.
func (h *Handlers) Cognition(w http.ResponseWriter, r *http.Request) {
	var req CognitionRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.IngestCognition(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, resp)
}

// Commit handles POST /v1/ingest/{id}/commit for operator promotion after review (M7).
func (h *Handlers) Commit(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil || id == uuid.Nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid ingestion id")
		return
	}
	resp, err := h.Service.CommitIngestion(r.Context(), id)
	if err != nil {
		switch {
		case errors.Is(err, ErrIngestionNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, ErrIngestionNotPromotable):
			httpx.WriteError(w, http.StatusConflict, err.Error())
		default:
			httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	httpx.WriteJSON(w, resp)
}
