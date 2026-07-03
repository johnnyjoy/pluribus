package chores

import (
	"net/http"
	"strconv"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers provides HTTP handlers for curation chores.
type Handlers struct {
	Service *Service
}

// List handles GET /v1/curation/chores?limit=N — open chores oldest-first.
func (h *Handlers) List(w http.ResponseWriter, r *http.Request) {
	limit := 20
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	list, err := h.Service.List(r.Context(), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if list == nil {
		list = []Chore{}
	}
	httpx.WriteJSON(w, ListResponse{Chores: list})
}

// Resolve handles POST /v1/curation/chores/{id}/resolve — one agent's vote.
func (h *Handlers) Resolve(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid chore id: expected a UUID")
		return
	}
	var req ResolveRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.Resolve(r.Context(), id, req)
	if err != nil {
		if err.Error() == "chore not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, resp)
}
