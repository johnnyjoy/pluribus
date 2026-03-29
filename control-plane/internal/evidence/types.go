package evidence

import (
	"time"

	"github.com/google/uuid"
)

// Record is the metadata for a stored evidence artifact (file on disk).
type Record struct {
	ID        uuid.UUID `json:"id"`
	Digest    string    `json:"digest"`
	Path      string    `json:"path"`
	Kind      string    `json:"kind,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateRequest is the payload for POST /v1/evidence.
// Content is base64-encoded body; Digest optional (computed from content if empty).
type CreateRequest struct {
	Kind      string    `json:"kind"`
	Digest    string    `json:"digest,omitempty"`
	Content   string    `json:"content"` // base64-encoded
}
