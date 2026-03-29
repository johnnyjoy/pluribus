package api

// MemoryKind is the type of a memory object.
type MemoryKind string

const (
	MemoryKindDecision     MemoryKind = "decision"
	MemoryKindConstraint   MemoryKind = "constraint"
	MemoryKindFailure      MemoryKind = "failure"
	MemoryKindPattern      MemoryKind = "pattern"
	// MemoryKindState is operational / resume context (recall continuity bucket).
	MemoryKindState MemoryKind = "state"
)

// Applicability indicates how a memory applies to the current context.
type Applicability string

const (
	ApplicabilityGoverning   Applicability = "governing"
	ApplicabilityAdvisory   Applicability = "advisory"
	ApplicabilityAnalogical Applicability = "analogical"
	ApplicabilityExperimental Applicability = "experimental"
)

// Status is the lifecycle status of a memory object or entity.
type Status string

const (
	StatusActive     Status = "active"
	StatusSuperseded Status = "superseded"
	StatusArchived   Status = "archived"
	StatusPending    Status = "pending"
	StatusRejected   Status = "rejected"
)
