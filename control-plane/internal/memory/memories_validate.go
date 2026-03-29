package memory

import "strings"

// ValidateMemoriesCreate checks POST /v1/memories bodies.
func ValidateMemoriesCreate(req *MemoriesCreateRequest) error {
	if req == nil {
		return errMemoriesInvalid("request body required")
	}
	if strings.TrimSpace(req.Statement) == "" {
		return errMemoriesInvalid("statement is required")
	}
	if !validKind(req.Kind) {
		return errMemoriesInvalid("invalid kind")
	}
	if req.Status != "" && !validCreateStatus(req.Status) {
		return errMemoriesInvalid("invalid status")
	}
	return nil
}

func errMemoriesInvalid(msg string) error {
	return &ValidationError{Msg: msg}
}
