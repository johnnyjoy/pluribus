package utilitypolicy

import "errors"

var (
	ErrNoService           = errors.New("utility policy service unavailable")
	ErrCandidateNotFound   = errors.New("utility candidate not found")
	ErrEvaluationMissing   = errors.New("evaluation_id required")
	ErrSessionMissing      = errors.New("session_id required")
	ErrMemoryMissing       = errors.New("memory_id required")
	ErrAlreadyApplied      = errors.New("candidate already applied")
	ErrStaleCandidate      = errors.New("candidate stale beyond policy window")
	ErrSessionCapExceeded  = errors.New("session positive apply cap exceeded")
	ErrAgentCapExceeded    = errors.New("agent positive apply cap exceeded")
	ErrTamperedCandidate   = errors.New("candidate failed integrity check")
	ErrApplicationNotFound = errors.New("utility application not found")
	ErrAlreadyReverted     = errors.New("application already reverted")
	ErrInvalidRollback     = errors.New("invalid rollback token")
	ErrScoreMutationDenied = errors.New("policy decision does not permit score mutation")
)
