package memory

import (
	"net/http"
	"strings"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CreateRelationshipRequest is POST /v1/memory/relationships.
type CreateRelationshipRequest struct {
	FromMemoryID     string `json:"from_memory_id"`
	ToMemoryID       string `json:"to_memory_id"`
	RelationshipType string `json:"relationship_type"`
	Reason           string `json:"reason,omitempty"`
	Source           string `json:"source,omitempty"`
}

// MemoryRelationshipsResponse is GET /v1/memory/{id}/relationships.
type MemoryRelationshipsResponse struct {
	MemoryID  string               `json:"memory_id"`
	Outbound  []MemoryRelationship `json:"outbound"`
	Inbound   []MemoryRelationship `json:"inbound"`
}

// CreateRelationship handles POST /v1/memory/relationships.
func (h *Handlers) CreateRelationship(w http.ResponseWriter, r *http.Request) {
	if h.Relationships == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "memory relationships not configured")
		return
	}
	var req CreateRelationshipRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	from, err := uuid.Parse(strings.TrimSpace(req.FromMemoryID))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid from_memory_id")
		return
	}
	to, err := uuid.Parse(strings.TrimSpace(req.ToMemoryID))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid to_memory_id")
		return
	}
	typ := RelationshipType(strings.TrimSpace(req.RelationshipType))
	out, err := h.Relationships.CreateRelationship(r.Context(), from, to, typ, req.Reason, req.Source)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	httpx.WriteJSONStatus(w, http.StatusCreated, out)
}

// ListRelationships handles GET /v1/memory/{id}/relationships.
func (h *Handlers) ListRelationships(w http.ResponseWriter, r *http.Request) {
	if h.Relationships == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "memory relationships not configured")
		return
	}
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid memory id: expected a UUID")
		return
	}
	outb, inb, err := h.Relationships.ListForMemory(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if outb == nil {
		outb = []MemoryRelationship{}
	}
	if inb == nil {
		inb = []MemoryRelationship{}
	}
	httpx.WriteJSON(w, MemoryRelationshipsResponse{
		MemoryID: id.String(),
		Outbound: outb,
		Inbound:  inb,
	})
}
