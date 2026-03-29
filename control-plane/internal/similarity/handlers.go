package similarity

import (
	"errors"
	"net/http"

	"control-plane/internal/httpx"
)

// Handlers exposes advisory episode HTTP API.
type Handlers struct {
	Service *Service
}

// Create handles POST /v1/advisory-episodes.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "similarity is unavailable: service is not configured")
		return
	}
	if h.Service.Repo == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "similarity is unavailable: database repo is not configured")
		return
	}
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	rec, err := h.Service.Create(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrSimilarityDisabled):
			httpx.WriteError(w, http.StatusForbidden, err.Error())
		default:
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	out := map[string]interface{}{
		"id":            rec.ID.String(),
		"summary_text":  rec.SummaryText,
		"source":        rec.Source,
		"tags":          rec.Tags,
		"created_at":    rec.CreatedAt,
		"advisory":      true,
		"non_canonical": true,
	}
	if rec.RelatedMemoryID != nil {
		out["related_memory_id"] = rec.RelatedMemoryID.String()
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, out)
}

// Similar handles POST /v1/advisory-episodes/similar.
func (h *Handlers) Similar(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "similarity is unavailable: service is not configured")
		return
	}
	var req SimilarRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.FindSimilar(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, resp)
}
