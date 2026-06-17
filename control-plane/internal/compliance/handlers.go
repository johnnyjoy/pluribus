package compliance

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"control-plane/internal/httpx"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handlers exposes /v1/compliance/* REST API.
type Handlers struct {
	Service *Service
}

// ListSessions GET /v1/compliance/sessions
func (h *Handlers) ListSessions(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "compliance unavailable")
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	sessions, err := h.Service.ListSessions(r.Context(), limit)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, map[string]any{"sessions": sessions})
}

// GetSession GET /v1/compliance/sessions/{id}
func (h *Handlers) GetSession(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "compliance unavailable")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	sess, err := h.Service.GetSession(r.Context(), id)
	if err != nil || sess == nil {
		httpx.WriteError(w, http.StatusNotFound, "session not found")
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, sess)
}

// SessionEvents GET /v1/compliance/sessions/{id}/events
func (h *Handlers) SessionEvents(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "compliance unavailable")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "invalid session id")
		return
	}
	events, err := h.Service.ListEvents(r.Context(), id)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, map[string]any{"events": events})
}

// Evaluate POST /v1/compliance/evaluate
func (h *Handlers) Evaluate(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "compliance unavailable")
		return
	}
	var req EvaluateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	id, err := uuid.Parse(req.SessionID)
	if err != nil {
		httpx.WriteError(w, http.StatusBadRequest, "session_id must be UUID")
		return
	}
	maxAge := DefaultRecallMaxAge
	if req.RecallMaxAgeMS != nil && *req.RecallMaxAgeMS > 0 {
		maxAge = time.Duration(*req.RecallMaxAgeMS) * time.Millisecond
	}
	out, err := h.Service.EvaluatePersisted(r.Context(), id, maxAge)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpx.WriteJSONStatus(w, http.StatusOK, out)
}

// Summary GET /v1/compliance/summary
func (h *Handlers) Summary(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "compliance unavailable")
		return
	}
	sessions, err := h.Service.ListSessions(r.Context(), 100)
	if err != nil {
		httpx.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}
	byStatus := map[string]int{}
	for _, sess := range sessions {
		ev, _ := h.Service.ListEvents(r.Context(), sess.ID)
		eval := EvaluateSession(ev, DefaultRecallMaxAge)
		byStatus[eval.Status]++
	}
	httpx.WriteJSONStatus(w, http.StatusOK, SummaryResponse{
		TotalSessions:   len(sessions),
		ByStatus:        byStatus,
		EvaluatedWindow: "recent_sessions_heuristic",
	})
}

// EvaluateSessionID helper for MCP tools.
func (h *Handlers) EvaluateSessionID(ctx context.Context, sessionID uuid.UUID) (Evaluation, error) {
	return h.Service.EvaluatePersisted(ctx, sessionID, DefaultRecallMaxAge)
}
