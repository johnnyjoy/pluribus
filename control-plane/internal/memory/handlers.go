package memory

import (
	"context"
	"errors"
	"net/http"
	"time"

	"control-plane/internal/formation"
	"control-plane/internal/httpx"
	"control-plane/pkg/api"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers provides HTTP handlers for memory objects.
type Handlers struct {
	Service         *Service
	Relationships *RelationshipRepo // optional: POST/GET memory relationships
}

// Create handles POST /v1/memory.
func (h *Handlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	req.FormationPath = formation.PathDirectCreate
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
		OccurredAt:    req.OccurredAt,
		FormationPath: formation.PathMemoriesCreate,
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

// BackfillEmbeddingsRequest is the optional body for POST /v1/memory/embeddings/backfill.
type BackfillEmbeddingsRequest struct {
	Limit int `json:"limit,omitempty"`
}

// BackfillEmbeddings handles POST /v1/memory/embeddings/backfill: embed rows
// created before the local embedder was enabled (or under a previous model).
func (h *Handlers) BackfillEmbeddings(w http.ResponseWriter, r *http.Request) {
	var req BackfillEmbeddingsRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	res, err := h.Service.BackfillEmbeddings(r.Context(), req.Limit)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, res)
}

// RemediationRequest is the optional body for DELETE /v1/memory/{id} and POST /v1/memory/{id}/quarantine.
type RemediationRequest struct {
	Reason string `json:"reason,omitempty"`
}

// Quarantine handles POST /v1/memory/{id}/quarantine (C3 remediation).
// The row is preserved with status=quarantined and never surfaced by recall.
func (h *Handlers) Quarantine(w http.ResponseWriter, r *http.Request) {
	h.remediate(w, r, h.Service.Quarantine)
}

// Delete handles DELETE /v1/memory/{id} (C3 remediation). Soft delete:
// the row is preserved as a tombstone (status=deleted) and excluded from all
// recall including historical mode. Canonical rows are never hard-deleted.
func (h *Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	h.remediate(w, r, h.Service.SoftDelete)
}

func (h *Handlers) remediate(w http.ResponseWriter, r *http.Request, apply func(ctx context.Context, id uuid.UUID, reason string) (*MemoryObject, error)) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id: expected a UUID")
		return
	}
	var req RemediationRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := httpx.DecodeJSON(r, &req); err != nil {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
	}
	obj, err := apply(r.Context(), id, req.Reason)
	if err != nil {
		if err.Error() == "memory not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, obj)
}

// MergeTagsRequest is the body for PUT /v1/memory/{id}/tags (additive tag merge).
type MergeTagsRequest struct {
	Tags []string `json:"tags"`
}

// MergeTags handles PUT /v1/memory/{id}/tags — merges tags without replacing existing.
func (h *Handlers) MergeTags(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id: expected a UUID")
		return
	}
	var req MergeTagsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	obj, err := h.Service.MergeTags(r.Context(), id, req.Tags)
	if err != nil {
		if err.Error() == "memory not found" {
			httpx.WriteError(w, http.StatusNotFound, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSON(w, obj)
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
