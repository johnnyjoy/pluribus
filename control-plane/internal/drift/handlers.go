package drift

import (
	"net/http"

	"control-plane/internal/httpx"
)

// Handlers provides HTTP handlers for drift check.
type Handlers struct {
	Service *Service
}

// Check handles POST /v1/drift/check.
func (h *Handlers) Check(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.Service.Check(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, result)
}
