package memory

import (
	"errors"
	"net/http"
	"time"

	"control-plane/internal/httpx"
	"control-plane/pkg/api"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers provides HTTP handlers for memory objects.
type Handlers struct {
	Service *Service
}

// Create handles POST /v1/memory.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	obj, err := h.Service.Create(r.Context(), req)
	if err != nil {
		var dup *ErrDuplicateMemory
		if errors.As(err, &dup) {
			httpx.WriteJSONStatus(w, http.StatusConflict, map[string]string{
				"error":     "duplicate memory",
				"memory_id": dup.ExistingID.String(),
			})
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, obj)
}

// PatternElevationRunRequest is the body for POST /v1/memory/pattern-elevation/run (optional).
type PatternElevationRunRequest struct {
	Tags []string `json:"tags,omitempty"`
}

// RunPatternElevation runs TryElevatePatterns for optional tag scope (empty = all patterns in DB scan limit).
func (h *Handlers) RunPatternElevation(w http.ResponseWriter, r *http.Request) {
	var req PatternElevationRunRequest
	_ = httpx.DecodeJSON(r, &req)
	list, err := h.Service.TryElevatePatterns(r.Context(), req.Tags)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if list == nil {
		list = []MemoryObject{}
	}
	httpx.WriteJSON(w, list)
}

// Search handles POST /v1/memory/search.
func (h *Handlers) Search(w http.ResponseWriter, r *http.Request) {
	var req SearchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := h.Service.Search(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if list == nil {
		list = []MemoryObject{}
	}
	httpx.WriteJSON(w, list)
}

// CreateMemories handles POST /v1/memories.
func (h *Handlers) CreateMemories(w http.ResponseWriter, r *http.Request) {
	var req MemoriesCreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if err := ValidateMemoriesCreate(&req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	auth := req.Authority
	if auth <= 0 {
		auth = 5
	}
	cr := CreateRequest{
		Kind:          req.Kind,
		Authority:     auth,
		Applicability: api.ApplicabilityGoverning,
		Statement:     req.Statement,
		Tags:          req.Tags,
		Payload:       req.Payload,
		Status:        req.Status,
	}
	obj, err := h.Service.Create(r.Context(), cr)
	if err != nil {
		var dup *ErrDuplicateMemory
		if errors.As(err, &dup) {
			httpx.WriteJSONStatus(w, http.StatusConflict, map[string]string{
				"error":     "duplicate memory",
				"memory_id": dup.ExistingID.String(),
			})
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, obj)
}

// SearchMemories handles POST /v1/memories/search.
func (h *Handlers) SearchMemories(w http.ResponseWriter, r *http.Request) {
	var req MemoriesSearchRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	list, err := h.Service.SearchMemories(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if list == nil {
		list = []MemoryObject{}
	}
	httpx.WriteJSON(w, list)
}

// AuthorityEventRequest is the body for POST /v1/memory/{id}/authority/event (Task 74).
type AuthorityEventRequest struct {
	Type string `json:"type"` // "validation", "contradiction", or "failure"
}

// ApplyAuthorityEvent handles POST /v1/memory/{id}/authority/event.
func (h *Handlers) ApplyAuthorityEvent(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	if idStr == "" {
		httpx.WriteError(w, http.StatusBadRequest, "memory id required")
		return
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id: expected a UUID")
		return
	}
	var req AuthorityEventRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Type != "validation" && req.Type != "contradiction" && req.Type != "failure" {
		httpx.WriteError(w, http.StatusBadRequest, "type must be validation, contradiction, or failure")
		return
	}
	obj, err := h.Service.ApplyAuthorityEvent(r.Context(), id, req.Type)
	if err != nil {
		if err.Error() == "memory lifecycle not configured" {
			httpx.WriteError(w, http.StatusServiceUnavailable, err.Error())
			return
		}
		if err.Error() == "memory not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, obj)
}

// ExpireMemoriesResponse is the response for POST /v1/memory/expire (Task 75).
type ExpireMemoriesResponse struct {
	Archived int `json:"archived"`
}

// ExpireMemories handles POST /v1/memory/expire (run nightly or via cron).
func (h *Handlers) ExpireMemories(w http.ResponseWriter, r *http.Request) {
	count, err := h.Service.ExpireMemories(r.Context(), time.Now())
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSON(w, ExpireMemoriesResponse{Archived: count})
}

// SetAttributesRequest is the body for PUT /v1/memory/{id}/attributes (Task 78: constraint attributes).
type SetAttributesRequest struct {
	Attributes map[string]string `json:"attributes"`
}

// SetAttributes handles PUT /v1/memory/{id}/attributes.
func (h *Handlers) SetAttributes(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id: expected a UUID")
		return
	}
	var req SetAttributesRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.Attributes == nil {
		req.Attributes = make(map[string]string)
	}
	if err := h.Service.SetAttributes(r.Context(), id, req.Attributes); err != nil {
		if err.Error() == "memory not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Promote handles POST /v1/memory/promote (Pluribus Phase A contract).
func (h *Handlers) Promote(w http.ResponseWriter, r *http.Request) {
	var req PromoteRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := h.Service.Promote(r.Context(), req)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, resp)
}
