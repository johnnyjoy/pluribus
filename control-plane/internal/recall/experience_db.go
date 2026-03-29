package recall

import (
	"context"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

// DBExperienceLister sources experience-like memories from durable DB storage.
// It replaces file-backed experience authority for authoritative recall paths.
type DBExperienceLister struct {
	Memory MemorySearcher
}

// NewDBExperienceLister constructs a DB-backed experience lister.
func NewDBExperienceLister(mem MemorySearcher) *DBExperienceLister {
	return &DBExperienceLister{Memory: mem}
}

// ListForCompile returns promoted/global decision/object-lesson memories tagged as experience/promoted.
func (d *DBExperienceLister) ListForCompile(ctx context.Context, limit int) ([]memory.MemoryObject, error) {
	if d == nil || d.Memory == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	list, err := d.Memory.Search(ctx, memory.SearchRequest{
		Tags:   []string{"experience", "promoted"},
		Status: string(api.StatusActive),
		Max:    limit,
	})
	if err != nil {
		return nil, err
	}
	return list, nil
}

