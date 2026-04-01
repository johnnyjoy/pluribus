package similarity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"time"

	"control-plane/internal/httpx"
)

// Handlers exposes advisory episode HTTP API.
type Handlers struct {
	Service *Service
	// AutoDistill when set runs after successful POST / (same distillation as explicit distill); failures are logged only.
	AutoDistill AutoDistiller
	// AfterAdvisoryCreate runs after insert (inline probationary memory formation); failures logged inside hook.
	AfterAdvisoryCreate func(ctx context.Context, rec *Record, deduplicated bool)
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
	tags := rec.Tags
	if tags == nil {
		tags = []string{}
	}
	ents := rec.Entities
	if ents == nil {
		ents = []string{}
	}
	out := map[string]interface{}{
		"id":            rec.ID.String(),
		"summary_text":  rec.SummaryText,
		"source":        rec.Source,
		"tags":          tags,
		"entities":      ents,
		"created_at":    rec.CreatedAt,
		"advisory":      true,
		"non_canonical": true,
	}
	if rec.OccurredAt != nil {
		out["occurred_at"] = rec.OccurredAt.UTC().Format(time.RFC3339Nano)
	}
	if rec.Deduplicated {
		out["deduplicated"] = true
	}
	if h.AutoDistill != nil && !rec.Deduplicated {
		if err := h.AutoDistill.DistillAfterAdvisoryIngest(r.Context(), rec.ID); err != nil {
			slog.Warn("[DISTILL AUTO] advisory episode ingest distill failed", "episode_id", rec.ID.String(), "error", err.Error())
		}
	}
	if h.AfterAdvisoryCreate != nil {
		h.AfterAdvisoryCreate(r.Context(), rec, rec.Deduplicated)
	}
	if h.Service != nil && h.Service.Repo != nil {
		if up, err := h.Service.Repo.GetByID(r.Context(), rec.ID); err == nil && up != nil {
			rec = up
		}
	}
	out["memory_formation_status"] = rec.MemoryFormationStatus
	if rec.RejectionReason != "" {
		out["rejection_reason"] = rec.RejectionReason
	}
	if rec.RelatedMemoryID != nil {
		out["related_memory_id"] = rec.RelatedMemoryID.String()
		out["probationary_memory_id"] = rec.RelatedMemoryID.String()
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

// PruneRejectedRequest is POST /v1/advisory-episodes/prune-rejected (triggered cleanup).
type PruneRejectedRequest struct {
	OlderThanHours int `json:"older_than_hours"`
	Limit          int `json:"limit,omitempty"`
}

// PruneRejected deletes rejected advisory_experiences older than a cutoff.
func (h *Handlers) PruneRejected(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil || h.Service.Repo == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "similarity: repo not configured")
		return
	}
	var req PruneRejectedRequest
	raw, _ := io.ReadAll(r.Body)
	if len(bytes.TrimSpace(raw)) == 0 {
		req.OlderThanHours = 24 * 30
	} else if err := json.Unmarshal(raw, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if req.OlderThanHours <= 0 {
		req.OlderThanHours = 24 * 30
	}
	limit := req.Limit
	if limit <= 0 {
		limit = 10_000
	}
	cutoff := time.Now().Add(-time.Duration(req.OlderThanHours) * time.Hour)
	n, err := h.Service.Repo.DeleteRejectedOlderThan(r.Context(), cutoff, limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	slog.Info("[ADVISORY_PRUNE]", "phase", "manual", "deleted", n, "older_than_hours", req.OlderThanHours)
	httpx.WriteJSON(w, map[string]any{"deleted": n})
}
