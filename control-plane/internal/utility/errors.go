package utility

import "errors"

var (
	ErrNoRepo              = errors.New("utility: no repository")
	ErrMemoryNotFound      = errors.New("memory not found")
	ErrInvalidEventType    = errors.New("invalid event_type")
	ErrReasonRequired      = errors.New("reason required for negative feedback")
	ErrInvalidPayload      = errors.New("invalid payload")
)
