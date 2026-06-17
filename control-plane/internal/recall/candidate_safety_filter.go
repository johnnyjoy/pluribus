package recall

import (
	"control-plane/internal/memory"
	"control-plane/pkg/api"
	"time"
)

// applyCandidateSafetyFilter enforces lifecycle/status eligibility and date bounds on merged candidates.
// Belt-and-suspenders layer after lexical + semantic merge; do not trust vector SQL alone.
func applyCandidateSafetyFilter(
	objs []memory.MemoryObject,
	mode RecallMode,
	allowedStatuses []api.Status,
	situationQuery string,
	after, before *time.Time,
) []memory.MemoryObject {
	if len(objs) == 0 {
		return objs
	}
	allowed := map[api.Status]struct{}{}
	for _, st := range allowedStatuses {
		allowed[st] = struct{}{}
	}
	legacySuperseded := mode == RecallModeCurrent && includeSupersededCandidates(situationQuery)

	out := objs[:0]
	for _, o := range objs {
		st := o.Status
		if st == "" {
			st = api.StatusActive
		}
		if st == api.StatusPending || st == api.StatusRejected {
			continue
		}
		if _, ok := allowed[st]; !ok {
			if !(legacySuperseded && st == api.StatusSuperseded) {
				continue
			}
		}
		et := memoryEffectiveTime(o)
		if after != nil && et.Before(after.UTC()) {
			continue
		}
		if before != nil && !et.Before(before.UTC()) {
			continue
		}
		out = append(out, o)
	}
	return out
}
