package recall

import (
	"errors"
	"fmt"
	"strings"

	"control-plane/pkg/api"
)

// RecallMode controls lifecycle-aware candidate retrieval for compile.
type RecallMode string

const (
	RecallModeCurrent    RecallMode = "current"
	RecallModeHistorical RecallMode = "historical"
)

var (
	ErrInvalidRecallMode = errors.New("invalid recall_mode: must be current or historical")
	ErrInvalidIncludeStatus = errors.New("invalid include_status: allowed values are active, pending, superseded, archived")
)

// LifecycleRecallMeta is attached to RecallBundle when lifecycle recall is active.
type LifecycleRecallMeta struct {
	RecallMode     string   `json:"recall_mode"`
	IncludeStatus  []string `json:"include_status,omitempty"`
	HistoricalMode bool     `json:"historical_mode,omitempty"`
}

// ResolveLifecycleRecall interprets recall_mode and include_status on a compile request.
// Default: current guidance (active + rare pending; same pool, pending weighted lower).
func ResolveLifecycleRecall(req CompileRequest) (RecallMode, []api.Status, LifecycleRecallMeta, error) {
	meta := LifecycleRecallMeta{}

	if len(req.IncludeStatus) > 0 {
		statuses, err := parseIncludeStatus(req.IncludeStatus)
		if err != nil {
			return "", nil, meta, err
		}
		meta.IncludeStatus = append([]string(nil), req.IncludeStatus...)
		mode := inferModeFromStatuses(statuses)
		meta.RecallMode = string(mode)
		meta.HistoricalMode = mode == RecallModeHistorical
		return mode, statuses, meta, nil
	}

	mode := RecallModeCurrent
	if s := strings.ToLower(strings.TrimSpace(req.RecallMode)); s != "" {
		mode = RecallMode(s)
	}
	switch mode {
	case RecallModeCurrent:
		meta.RecallMode = string(RecallModeCurrent)
		return RecallModeCurrent, []api.Status{api.StatusActive, api.StatusPending}, meta, nil
	case RecallModeHistorical:
		meta.RecallMode = string(RecallModeHistorical)
		meta.HistoricalMode = true
		return RecallModeHistorical, []api.Status{api.StatusActive, api.StatusSuperseded, api.StatusArchived}, meta, nil
	default:
		return "", nil, meta, ErrInvalidRecallMode
	}
}

func parseIncludeStatus(raw []string) ([]api.Status, error) {
	allowed := map[string]api.Status{
		"active":     api.StatusActive,
		"pending":    api.StatusPending,
		"superseded": api.StatusSuperseded,
		"archived":   api.StatusArchived,
	}
	seen := map[api.Status]struct{}{}
	var out []api.Status
	for _, s := range raw {
		s = strings.ToLower(strings.TrimSpace(s))
		if s == "" {
			continue
		}
		st, ok := allowed[s]
		if !ok {
			return nil, fmt.Errorf("%w: %q", ErrInvalidIncludeStatus, s)
		}
		if _, dup := seen[st]; dup {
			continue
		}
		seen[st] = struct{}{}
		out = append(out, st)
	}
	if len(out) == 0 {
		return nil, ErrInvalidIncludeStatus
	}
	return out, nil
}

func inferModeFromStatuses(statuses []api.Status) RecallMode {
	hasHistorical := false
	for _, st := range statuses {
		if st == api.StatusSuperseded || st == api.StatusArchived {
			hasHistorical = true
			break
		}
	}
	if hasHistorical {
		return RecallModeHistorical
	}
	return RecallModeCurrent
}
