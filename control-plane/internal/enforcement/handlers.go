package enforcement

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"control-plane/internal/httpx"
	"control-plane/internal/memory"

	"github.com/google/uuid"
)

// Handlers exposes POST /v1/enforcement/evaluate.
type Handlers struct {
	Service *Service
}

// Evaluate handles POST /v1/enforcement/evaluate.
func (h *Handlers) Evaluate(w http.ResponseWriter, r *http.Request) {
	if h.Service == nil {
		httpx.WriteError(w, http.StatusServiceUnavailable, "enforcement is unavailable: service is not wired (internal configuration error)")
		return
	}
	var req EvaluateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	ctx := r.Context()
	out, err := h.Service.Evaluate(ctx, req)
	if err != nil {
		if errors.Is(err, ErrDisabled) {
			httpx.WriteError(w, http.StatusForbidden, err.Error())
			return
		}
		if strings.Contains(err.Error(), "project not found") {
			httpx.WriteError(w, http.StatusBadRequest, err.Error())
			return
		}
		httpx.WriteError(w, http.StatusBadRequest, err.Error())
		return
	}
	if h.Service.SuccessReinforcer != nil && out.Validation.NextAction == "proceed" && out.Validation.Passed && len(out.TriggeredMemories) > 0 {
		ids := make([]uuid.UUID, 0, len(out.TriggeredMemories))
		for _, tm := range out.TriggeredMemories {
			ids = append(ids, tm.MemoryID)
		}
		type recallMetaReinforcer interface {
			ReinforceRecallUsageWithMeta(context.Context, []uuid.UUID, memory.ReinforceMeta) error
		}
		if m, ok := h.Service.SuccessReinforcer.(recallMetaReinforcer); ok {
			_ = m.ReinforceRecallUsageWithMeta(ctx, ids, memory.ReinforceMeta{
				Reason:     "success_enforcement_proceed",
				AgentKey:   memory.AgentUsageKey(req.AgentID),
			})
		} else {
			_ = h.Service.SuccessReinforcer.ReinforceSuccess(ctx, ids, "success_enforcement_proceed")
		}
	}
	httpx.WriteJSON(w, out)
}
