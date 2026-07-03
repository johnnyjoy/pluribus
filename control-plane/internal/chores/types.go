// Package chores implements agent-driven curation: the server packages review
// work it detected deterministically (unresolved contradictions, quarantined
// rows, embedding near-duplicate pairs) as small "chores" that visiting agents
// judge. A resolution applies only after min_resolvers DISTINCT agents
// (memory.AgentUsageKey hash; the memory's own author never counts) vote for
// the same action. The backend stays LLM-free.
package chores

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Chore types.
const (
	TypeContradiction    = "contradiction"
	TypeQuarantineReview = "quarantine_review"
	TypeDuplicatePair    = "duplicate_pair"
)

// Chore states.
const (
	StateOpen      = "open"
	StateResolved  = "resolved"
	StateDismissed = "dismissed"
)

// Actions per chore type.
const (
	// contradiction: keep one side (the other is superseded) or let both stand.
	ActionKeepSubject = "keep_subject"
	ActionKeepRelated = "keep_related"
	ActionCoexist     = "coexist"
	// quarantine_review: release back to pending, or soft-delete.
	ActionRelease = "release"
	ActionDelete  = "delete"
	// duplicate_pair: consolidate (lower-authority/older row superseded) or keep both.
	ActionConsolidate = "consolidate"
	ActionDistinct    = "distinct"
)

// AllowedActions maps chore type to the actions agents may vote for.
var AllowedActions = map[string][]string{
	TypeContradiction:    {ActionKeepSubject, ActionKeepRelated, ActionCoexist},
	TypeQuarantineReview: {ActionRelease, ActionDelete},
	TypeDuplicatePair:    {ActionConsolidate, ActionDistinct},
}

// Chore is one unit of review work offered to visiting agents.
type Chore struct {
	ID               uuid.UUID       `json:"id"`
	Type             string          `json:"chore_type"`
	SubjectMemoryID  uuid.UUID       `json:"subject_memory_id"`
	RelatedMemoryID  *uuid.UUID      `json:"related_memory_id,omitempty"`
	Evidence         json.RawMessage `json:"evidence,omitempty"`
	State            string          `json:"state"`
	ResolutionAction string          `json:"resolution_action,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
	ResolvedAt       *time.Time      `json:"resolved_at,omitempty"`
	// Statement snippets so the voting agent can judge without extra lookups.
	SubjectStatement string   `json:"subject_statement,omitempty"`
	RelatedStatement string   `json:"related_statement,omitempty"`
	Actions          []string `json:"actions,omitempty"`
	VoteCount        int      `json:"vote_count"`
}

// ResolveRequest is the body for POST /v1/curation/chores/{id}/resolve.
type ResolveRequest struct {
	AgentID string `json:"agent_id"`
	Action  string `json:"action"`
	Reason  string `json:"reason,omitempty"`
}

// ResolveResponse reports what a vote did.
type ResolveResponse struct {
	ChoreID        uuid.UUID `json:"chore_id"`
	Recorded       bool      `json:"recorded"`
	Counted        bool      `json:"counted"`
	Note           string    `json:"note,omitempty"`
	VotesForAction int       `json:"votes_for_action"`
	MinResolvers   int       `json:"min_resolvers"`
	Applied        bool      `json:"applied"`
	State          string    `json:"state"`
}

// ListResponse wraps GET /v1/curation/chores.
type ListResponse struct {
	Chores []Chore `json:"chores"`
}

func actionAllowed(choreType, action string) bool {
	for _, a := range AllowedActions[choreType] {
		if a == action {
			return true
		}
	}
	return false
}
