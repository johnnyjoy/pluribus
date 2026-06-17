package benchmark

import (
	"context"
	"strings"

	"control-plane/internal/memory"
)

// MemoryStub is an in-memory MemorySearcher for benchmark runs.
type MemoryStub struct {
	All []memory.MemoryObject
}

func (s *MemoryStub) Search(_ context.Context, req memory.SearchRequest) ([]memory.MemoryObject, error) {
	statuses := req.Statuses
	if len(statuses) == 0 {
		want := string(req.Status)
		if want == "" {
			want = "active"
		}
		statuses = []string{want}
	}
	allowed := map[string]struct{}{}
	for _, st := range statuses {
		allowed[st] = struct{}{}
	}
	var out []memory.MemoryObject
	for _, o := range s.All {
		st := string(o.Status)
		if st == "" {
			st = "active"
		}
		if _, ok := allowed[st]; !ok {
			continue
		}
		if len(req.Tags) == 0 || tagOverlap(o.Tags, req.Tags) {
			out = append(out, o)
		}
	}
	return append([]memory.MemoryObject(nil), out...), nil
}

func (s *MemoryStub) SearchMemories(_ context.Context, req memory.MemoriesSearchRequest) ([]memory.MemoryObject, error) {
	want := string(req.Status)
	if want == "" {
		want = "active"
	}
		q := strings.ToLower(strings.TrimSpace(req.Query))
		var out []memory.MemoryObject
		for _, o := range s.All {
			if string(o.Status) != want {
				continue
			}
			if len(req.Tags) > 0 && !tagOverlap(o.Tags, req.Tags) {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(o.Statement), q) {
				continue
			}
		out = append(out, o)
	}
	return append([]memory.MemoryObject(nil), out...), nil
}

func tagOverlap(have, want []string) bool {
	if len(want) == 0 {
		return true
	}
	set := map[string]bool{}
	for _, t := range have {
		set[t] = true
	}
	for _, t := range want {
		if set[t] {
			return true
		}
	}
	return false
}
