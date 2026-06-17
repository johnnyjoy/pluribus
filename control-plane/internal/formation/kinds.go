package formation

import "control-plane/pkg/api"

// ValidDirectCreateKinds are memory kinds allowed on direct create paths (no experience).
var ValidDirectCreateKinds = []api.MemoryKind{
	api.MemoryKindState,
	api.MemoryKindDecision,
	api.MemoryKindFailure,
	api.MemoryKindPattern,
	api.MemoryKindConstraint,
}

// ValidDirectCreateKind reports whether kind is allowed for direct memory creation.
func ValidDirectCreateKind(k api.MemoryKind) bool {
	switch k {
	case api.MemoryKindState, api.MemoryKindDecision, api.MemoryKindFailure,
		api.MemoryKindPattern, api.MemoryKindConstraint:
		return true
	default:
		return false
	}
}
