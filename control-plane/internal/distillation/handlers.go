package distillation

import (
	"errors"
	"net/http"

	"control-plane/internal/httpx"
)

// Handlers exposes POST /v1/episodes/distill.
type Handlers struct {
	Service *Service
}

// Distill handles POST /v1/episodes/distill.
func (h *Handlers) Distill(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "distillation is unavailable")
		return
	}
	var req DistillRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.Distill(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrDistillationDisabled):
			httpx.WriteError(w, http.StatusForbidden, err.Error())
		case errors.Is(err, ErrEpisodeNotFound):
			httpx.WriteError(w, http.StatusNotFound, err.Error())
		default:
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
		}
		return
	}
	httpx.WriteJSON(w, resp)
}
