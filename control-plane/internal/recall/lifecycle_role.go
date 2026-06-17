package recall

import (
	"control-plane/internal/memory"
	"control-plane/internal/utility"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// LifecycleRole values exposed on recall bundle items (Phase 8 contract).
const (
	LifecycleCurrentGuidance   = "current_guidance"
	LifecycleHistoricalContext = "historical_context"
	LifecycleSupersededContext = "superseded_context"
	LifecycleArchivedContext   = "archived_context"
	LifecycleRefutedContext    = "refuted_context"
	LifecycleOutdatedContext   = "outdated_context"
	LifecyclePendingContext    = "pending_context"
)

// deriveLifecycleRole labels a memory for agent reasoning without inferring from score alone.
func deriveLifecycleRole(mode RecallMode, obj memory.MemoryObject, util *utility.Score) string {
	switch obj.Status {
	case api.StatusSuperseded:
		return LifecycleSupersededContext
	case api.StatusArchived:
		return LifecycleArchivedContext
	case api.StatusPending:
		return LifecyclePendingContext
	case api.StatusRejected:
		return LifecyclePendingContext
	case api.StatusActive:
		if util != nil {
			if util.RefutedCount > 0 || util.WrongCount > 0 {
				if mode == RecallModeHistorical {
					return LifecycleRefutedContext
				}
			}
			if util.OutdatedCount > 0 {
				if mode == RecallModeHistorical {
					return LifecycleOutdatedContext
				}
			}
		}
		return LifecycleCurrentGuidance
	default:
		if mode == RecallModeHistorical {
			return LifecycleHistoricalContext
		}
		return LifecycleCurrentGuidance
	}
}

// applyHistoricalScoreCap prevents historical rows from masquerading as top current guidance within historical mode.
func applyHistoricalScoreCap(mode RecallMode, obj memory.MemoryObject, score float64) float64 {
	if mode != RecallModeHistorical {
		return score
	}
	switch obj.Status {
	case api.StatusSuperseded:
		if score > 0.72 {
			return 0.72
		}
	case api.StatusArchived:
		if score > 0.68 {
			return 0.68
		}
	}
	return score
}

func supersededByString(superMap map[uuid.UUID]uuid.UUID, id uuid.UUID) string {
	if superMap == nil {
		return ""
	}
	if newID, ok := superMap[id]; ok {
		return newID.String()
	}
	return ""
}
