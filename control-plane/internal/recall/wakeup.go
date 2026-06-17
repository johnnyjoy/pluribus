package recall

import (
	"context"
	"strings"

	"control-plane/pkg/api"
)

// Wake-up recall is a compiled view over the same memory pool and compiler as POST /v1/recall/compile.
// It does not add storage, ranking engines, or alternate truth semantics.

const (
	defaultWakeupMaxState          = 4
	defaultWakeupMaxPerKind        = 2
	defaultWakeupMaxGoverningTotal = 12
)

// WakeupRequest is the body for POST /v1/recall/wakeup.
type WakeupRequest struct {
	AgentID       string   `json:"agent_id,omitempty"`
	Tags          []string `json:"tags,omitempty"`
	CorrelationID string   `json:"correlation_id,omitempty"`
	// MaxState caps L0 identity rows (kind state only); default defaultWakeupMaxState.
	MaxState int `json:"max_state,omitempty"`
	// MaxPerKind caps each primary bucket before governing merge; default defaultWakeupMaxPerKind.
	MaxPerKind int `json:"max_per_kind,omitempty"`
	// MaxGoverningTotal caps L1 rows after governing applicability filter; default defaultWakeupMaxGoverningTotal.
	MaxGoverningTotal int `json:"max_governing_total,omitempty"`
	TelemetryOptions
}

// WakeupResponse is a compact session-start payload: L0 identity + L1 governing memory only.
type WakeupResponse struct {
	Identity          []MemoryItem         `json:"identity"`
	GoverningMemory   []MemoryItem         `json:"governing_memory"`
	RecallPreamble    string               `json:"recall_preamble,omitempty"`
	LimitsApplied     WakeupLimitsApplied  `json:"limits_applied,omitempty"`
	Telemetry         *RecallTelemetry     `json:"telemetry,omitempty"`
}

// WakeupLimitsApplied records the effective caps for this response.
type WakeupLimitsApplied struct {
	MaxState          int `json:"max_state"`
	MaxPerKind        int `json:"max_per_kind"`
	MaxGoverningTotal int `json:"max_governing_total"`
}

// Wakeup runs one compile (empty retrieval query, no experience prepend) and projects L0/L1 slices.
// Skips compile cache and usage reinforcement so session starts stay cheap and do not distort global reinforcement.
func (s *Service) Wakeup(ctx context.Context, req WakeupRequest) (*WakeupResponse, error) {
	if s.Compiler == nil {
		return nil, ErrNoCompiler
	}
	maxState := req.MaxState
	if maxState <= 0 {
		maxState = defaultWakeupMaxState
	}
	maxPerKind := req.MaxPerKind
	if maxPerKind <= 0 {
		maxPerKind = defaultWakeupMaxPerKind
	}
	maxGov := req.MaxGoverningTotal
	if maxGov <= 0 {
		maxGov = defaultWakeupMaxGoverningTotal
	}

	cr := CompileRequest{
		AgentID:                 req.AgentID,
		Tags:                    req.Tags,
		CorrelationID:           req.CorrelationID,
		RetrievalQuery:          "",
		ProposalText:            "",
		EnableTriggeredRecall:   false,
		MaxPerKind:              maxPerKind,
		MaxTotal:                0,
		MaxTokens:               0,
		Mode:                    "continuity",
		SkipExperienceHydration: true,
	}

	bundle, err := s.Compiler.Compile(ctx, cr)
	if err != nil {
		return nil, err
	}
	if err := s.hydrateEvidence(ctx, bundle); err != nil {
		return nil, err
	}
	AugmentWhyMattersWithEvidence(bundle)

	out := BuildWakeupResponse(bundle, maxState, maxGov)
	out.LimitsApplied = WakeupLimitsApplied{
		MaxState:          maxState,
		MaxPerKind:        maxPerKind,
		MaxGoverningTotal: maxGov,
	}
	return out, nil
}

// BuildWakeupResponse derives L0/L1 from an existing RecallBundle (same shapes as compile; selection only).
func BuildWakeupResponse(b *RecallBundle, maxState, maxGoverningTotal int) *WakeupResponse {
	if b == nil {
		return &WakeupResponse{}
	}
	resp := &WakeupResponse{RecallPreamble: b.RecallPreamble}

	stateKind := string(api.MemoryKindState)
	for _, it := range b.Continuity {
		if !strings.EqualFold(it.Kind, stateKind) {
			continue
		}
		resp.Identity = append(resp.Identity, it)
		if len(resp.Identity) >= maxState {
			break
		}
	}

	seen := make(map[string]struct{})
	appendGov := func(slice []MemoryItem) {
		for _, it := range slice {
			if !applicabilityIncludedInWakeupGoverning(it.Applicability) {
				continue
			}
			if _, ok := seen[it.ID]; ok {
				continue
			}
			seen[it.ID] = struct{}{}
			resp.GoverningMemory = append(resp.GoverningMemory, it)
			if len(resp.GoverningMemory) >= maxGoverningTotal {
				return
			}
		}
	}
	appendGov(b.GoverningConstraints)
	if len(resp.GoverningMemory) >= maxGoverningTotal {
		return resp
	}
	appendGov(b.Decisions)
	if len(resp.GoverningMemory) >= maxGoverningTotal {
		return resp
	}
	appendGov(b.KnownFailures)
	if len(resp.GoverningMemory) >= maxGoverningTotal {
		return resp
	}
	appendGov(b.ApplicablePatterns)
	return resp
}

// applicabilityIncludedInWakeupGoverning treats empty applicability like durable defaults (governing when unset at persist).
func applicabilityIncludedInWakeupGoverning(a api.Applicability) bool {
	switch a {
	case "", api.ApplicabilityGoverning:
		return true
	default:
		return false
	}
}
