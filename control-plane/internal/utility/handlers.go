package utility

import (
	"errors"
	"net/http"
	"strconv"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers exposes REST endpoints for memory utility feedback.
type Handlers struct {
	Service *Service
}

// PostFeedback handles POST /v1/memory/{id}/feedback.
func (h *Handlers) PostFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := parseMemoryID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id")
		return
	}
	var req FeedbackRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.RecordFeedback(r.Context(), id, req)
	if err != nil {
		writeFeedbackError(w, err)
		return
	}
	httpx.WriteJSON(w, resp)
}

// ListFeedback handles GET /v1/memory/{id}/feedback.
func (h *Handlers) ListFeedback(w http.ResponseWriter, r *http.Request) {
	id, err := parseMemoryID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id")
		return
	}
	limit := 50
	if q := r.URL.Query().Get("limit"); q != "" {
		if n, err := strconv.Atoi(q); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.Service.ListUtilityEvents(r.Context(), id, limit)
	if err != nil {
		writeFeedbackError(w, err)
		return
	}
	httpx.WriteJSON(w, list)
}

// GetUtility handles GET /v1/memory/{id}/utility.
func (h *Handlers) GetUtility(w http.ResponseWriter, r *http.Request) {
	id, err := parseMemoryID(r)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id")
		return
	}
	score, err := h.Service.GetUtilityScore(r.Context(), id)
	if err != nil {
		writeFeedbackError(w, err)
		return
	}
	httpx.WriteJSON(w, score)
}

func parseMemoryID(r *http.Request) (uuid.UUID, error) {
	return uuid.Parse(chi.URLParam(r, "id"))
}

func writeFeedbackError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrMemoryNotFound):
		httpx.WriteError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrInvalidEventType), errors.Is(err, ErrReasonRequired), errors.Is(err, ErrInvalidPayload):
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
	default:
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
	}
}
