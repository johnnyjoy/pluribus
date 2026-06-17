package benchmark

import (
	"strings"
	"time"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

// ViolationMetrics tracks lifecycle/date/utility/mode violations for hybrid gates.
type ViolationMetrics struct {
	LifecycleViolationCount int     `json:"lifecycle_violation_count"`
	LifecycleViolationRate  float64 `json:"lifecycle_violation_rate"`
	DateBoundViolationCount int     `json:"date_bound_violation_count"`
	DateBoundViolationRate  float64 `json:"date_bound_violation_rate"`
	UtilityViolationCount   int     `json:"utility_violation_count"`
	UtilityViolationRate    float64 `json:"utility_violation_rate"`
	ModeViolationCount      int     `json:"mode_violation_count"`
	ModeViolationRate       float64 `json:"mode_violation_rate"`
	LifecycleViolators      []string `json:"lifecycle_violators,omitempty"`
	DateBoundViolators      []string `json:"date_bound_violators,omitempty"`
	UtilityViolators        []string `json:"utility_violators,omitempty"`
}

func memoryEffectiveTime(m memory.MemoryObject) time.Time {
	if m.OccurredAt != nil {
		return m.OccurredAt.UTC()
	}
	return m.CreatedAt.UTC()
}

func resolveAllowedStatuses(c FixtureCase) []api.Status {
	if len(c.IncludeStatus) > 0 {
		var out []api.Status
		for _, s := range c.IncludeStatus {
			switch strings.ToLower(strings.TrimSpace(s)) {
			case "active":
				out = append(out, api.StatusActive)
			case "superseded":
				out = append(out, api.StatusSuperseded)
			case "archived":
				out = append(out, api.StatusArchived)
			}
		}
		return out
	}
	mode := strings.ToLower(strings.TrimSpace(c.RecallMode))
	if mode == "historical" {
		return []api.Status{api.StatusActive, api.StatusSuperseded, api.StatusArchived}
	}
	return []api.Status{api.StatusActive}
}

func parseOptionalTime(s string) *time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	layouts := []string{time.RFC3339, time.RFC3339Nano, "2006-01-02"}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			tt := t.UTC()
			return &tt
		}
	}
	return nil
}

// ComputeViolations evaluates lifecycle/date/utility/mode violations for ranked hits.
func ComputeViolations(c FixtureCase, lc *LoadedCorpus, hits []RankedHit) ViolationMetrics {
	allowed := map[api.Status]struct{}{}
	for _, st := range resolveAllowedStatuses(c) {
		allowed[st] = struct{}{}
	}
	after := parseOptionalTime(c.OccurredAfter)
	before := parseOptionalTime(c.OccurredBefore)

	var vm ViolationMetrics
	for _, h := range hits {
		id := lc.LabelToID[h.Label]
		fix, ok := lc.IDToFixture[id]
		if !ok {
			continue
		}
		st := api.Status(strings.TrimSpace(fix.Status))
		if st == "" {
			st = api.StatusActive
		}
		// Lifecycle / mode violations
		if st == api.StatusPending || st == api.StatusRejected {
			vm.LifecycleViolationCount++
			vm.ModeViolationCount++
			vm.LifecycleViolators = append(vm.LifecycleViolators, h.Label)
			continue
		}
		if _, ok := allowed[st]; !ok {
			vm.LifecycleViolationCount++
			vm.ModeViolationCount++
			vm.LifecycleViolators = append(vm.LifecycleViolators, h.Label)
		}
		// Date-bound violations
		for _, o := range lc.Objects {
			if o.ID != id {
				continue
			}
			et := memoryEffectiveTime(o)
			if after != nil && et.Before(after.UTC()) {
				vm.DateBoundViolationCount++
				vm.DateBoundViolators = append(vm.DateBoundViolators, h.Label)
			}
			if before != nil && !et.Before(before.UTC()) {
				vm.DateBoundViolationCount++
				vm.DateBoundViolators = append(vm.DateBoundViolators, h.Label)
			}
			break
		}
	}
	// Utility rank violations
	for lbl, maxRank := range c.MaxRankForLabel {
		if r, ok := hitRank(hits, lbl); ok && r > maxRank {
			vm.UtilityViolationCount++
			vm.UtilityViolators = append(vm.UtilityViolators, lbl)
		}
	}
	for mustLead, mustTrail := range c.RequiredBeforeLabel {
		leadRank, leadOK := hitRank(hits, mustLead)
		trailRank, trailOK := hitRank(hits, mustTrail)
		if leadOK && trailOK && leadRank > trailRank {
			vm.UtilityViolationCount++
			vm.UtilityViolators = append(vm.UtilityViolators, mustLead+">"+mustTrail)
		}
	}
	n := float64(len(hits))
	if n > 0 {
		vm.LifecycleViolationRate = float64(vm.LifecycleViolationCount) / n
		vm.DateBoundViolationRate = float64(vm.DateBoundViolationCount) / n
		vm.UtilityViolationRate = float64(vm.UtilityViolationCount) / n
		vm.ModeViolationRate = float64(vm.ModeViolationCount) / n
	}
	return vm
}

func hitRank(hits []RankedHit, label string) (int, bool) {
	for _, h := range hits {
		if h.Label == label {
			return h.Rank, true
		}
	}
	return 0, false
}
