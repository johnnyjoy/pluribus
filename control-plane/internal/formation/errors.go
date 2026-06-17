package formation

import (
	"fmt"

	"control-plane/internal/formationquality"
)

// ErrRejected indicates the formation gate rejected the write.
type ErrRejected struct {
	Reason string
	Path   Path
}

func (e *ErrRejected) Error() string {
	if e == nil {
		return "formation rejected"
	}
	if e.Path != "" {
		return fmt.Sprintf("formation rejected (%s): %s", e.Path, e.Reason)
	}
	return fmt.Sprintf("formation rejected: %s", e.Reason)
}

// Path identifies which memory formation entrypoint applied the gate.
type Path string

const (
	PathDirectCreate      Path = "direct_create"
	PathMemoriesCreate    Path = "memories_create"
	PathPromote           Path = "promote"
	PathProbationaryIngest Path = "probationary_ingest"
)

// Outcome is the gate decision for a write.
type Outcome string

const (
	OutcomeAllow   Outcome = "allow"
	OutcomeReject  Outcome = "reject"
	OutcomePending Outcome = "pending"
)

// Decision is the result of evaluating a memory create request.
type Decision struct {
	Outcome       Outcome
	Reason        string
	CapAuthority  int // 0 = no cap applied
	ForcePending  bool
	ForceAdvisory bool // downgrade governing → advisory (probationary path only)
	Quality       *formationquality.Result
}
