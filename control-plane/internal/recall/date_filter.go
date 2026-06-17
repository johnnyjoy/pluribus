package recall

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"control-plane/internal/memory"
)

var (
	ErrInvalidOccurredAfter  = errors.New("invalid occurred_after: use RFC3339 timestamp")
	ErrInvalidOccurredBefore = errors.New("invalid occurred_before: use RFC3339 timestamp")
	ErrInvalidDateRange      = errors.New("occurred_after must be before occurred_before")
)

// ParseCompileDateBounds validates optional occurred_after / occurred_before on a compile request.
func ParseCompileDateBounds(req CompileRequest) (after, before *time.Time, err error) {
	if s := strings.TrimSpace(req.OccurredAfter); s != "" {
		t, perr := parseISO8601(s)
		if perr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrInvalidOccurredAfter, perr)
		}
		after = &t
	}
	if s := strings.TrimSpace(req.OccurredBefore); s != "" {
		t, perr := parseISO8601(s)
		if perr != nil {
			return nil, nil, fmt.Errorf("%w: %v", ErrInvalidOccurredBefore, perr)
		}
		before = &t
	}
	if after != nil && before != nil && !after.Before(*before) {
		return nil, nil, ErrInvalidDateRange
	}
	return after, before, nil
}

func parseISO8601(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	var lastErr error
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.UTC(), nil
		} else {
			lastErr = err
		}
	}
	return time.Time{}, lastErr
}

// memoryEffectiveTime returns occurred_at when set, otherwise created_at (documented fallback).
func memoryEffectiveTime(m memory.MemoryObject) time.Time {
	if m.OccurredAt != nil {
		return m.OccurredAt.UTC()
	}
	return m.CreatedAt.UTC()
}

// filterByDateBounds keeps memories whose effective time falls within [after, before) when bounds are set.
func filterByDateBounds(objs []memory.MemoryObject, after, before *time.Time) []memory.MemoryObject {
	if after == nil && before == nil {
		return objs
	}
	out := objs[:0]
	for _, o := range objs {
		et := memoryEffectiveTime(o)
		if after != nil && et.Before(after.UTC()) {
			continue
		}
		if before != nil && !et.Before(before.UTC()) {
			continue
		}
		out = append(out, o)
	}
	return out
}
