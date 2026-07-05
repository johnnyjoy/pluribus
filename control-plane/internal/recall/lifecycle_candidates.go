package recall

import (
	"context"
	"strings"

	"control-plane/internal/memory"
	"control-plane/pkg/api"
)

func includeSupersededCandidates(query string) bool {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return false
	}
	for _, term := range []string{"deprecated", "superseded", "obsolete", "legacy", "stal"} {
		if strings.Contains(q, term) {
			return true
		}
	}
	if strings.Contains(q, "sqlite") && (strings.Contains(q, "still") || strings.Contains(q, "use")) {
		return true
	}
	return false
}

// fetchLifecycleCandidates loads compile candidates according to recall_mode / include_status.
// Current mode: active + pending (pending weighted lower in ranking); legacy keyword merge for superseded.
// Historical mode: active + pending + superseded + archived; rejected never included.
func (c *Compiler) fetchLifecycleCandidates(ctx context.Context, req CompileRequest, mode RecallMode, statuses []api.Status, situationQuery string) ([]memory.MemoryObject, error) {
	if c.Memory == nil {
		return nil, nil
	}
	var objs []memory.MemoryObject
	for _, st := range statuses {
		batch, err := c.Memory.Search(ctx, memory.SearchRequest{
			Tags:   req.Tags,
			Status: string(st),
			Max:    100,
		})
		if err != nil {
			return nil, err
		}
		objs = mergeUniqueMemoryObjects(objs, batch)
	}
	// Legacy current-mode path: merge superseded when query explicitly asks about lifecycle change.
	if mode == RecallModeCurrent && includeSupersededCandidates(situationQuery) {
		supReq := memory.SearchRequest{
			Tags:   req.Tags,
			Status: string(api.StatusSuperseded),
			Max:    50,
		}
		if superseded, serr := c.Memory.Search(ctx, supReq); serr == nil {
			objs = mergeUniqueMemoryObjects(objs, superseded)
		}
	}
	if strings.TrimSpace(situationQuery) != "" {
		searchStatuses := statuses
		if mode == RecallModeCurrent && includeSupersededCandidates(situationQuery) {
			searchStatuses = append(append([]api.Status{}, statuses...), api.StatusSuperseded)
		}
		// Hybrid slice 2/3 (H1): Postgres full-text top-K over the whole situation
		// query (GIN tsvector index), so the pool no longer depends solely on the
		// authority-top slice + per-keyword ILIKE bridges as the corpus grows.
		// Optional interface: real service implements it; stubs without it skip.
		type fullTextSearcher interface {
			SearchFullText(ctx context.Context, query, status string, max int) ([]memory.MemoryObject, error)
		}
		if fts, ok := c.Memory.(fullTextSearcher); ok {
			for _, st := range searchStatuses {
				extra, err := fts.SearchFullText(ctx, situationQuery, string(st), 50)
				if err != nil {
					return nil, err
				}
				objs = mergeUniqueMemoryObjects(objs, extra)
			}
		}
		keywords := situationKeywords(situationQuery)
		for _, kw := range keywords {
			for _, st := range searchStatuses {
				// Keyword bridge intentionally omits req.Tags: tag-filtered status search
				// already narrows the pool; keyword expansion must surface lexically relevant
				// memories whose tags differ (e.g. mobile credential decision under housing query).
				extra, err := c.Memory.SearchMemories(ctx, memory.MemoriesSearchRequest{
					Query:  kw,
					Status: string(st),
					Max:    50,
				})
				if err != nil {
					return nil, err
				}
				extra = filterKeywordBridgeCandidates(req.Tags, kw, keywords, extra)
				objs = mergeUniqueMemoryObjects(objs, extra)
			}
		}
	}
	return filterLifecycleCandidates(mode, objs), nil
}

// filterKeywordBridgeCandidates gates tag-less keyword hits when the caller supplied tags.
// Require tag overlap or a distinctive (>=6 char) search keyword match so short bridges
// (e.g. "proof" matching noise fillers) do not bypass tag-focused compile while cross-tag
// decisions (e.g. "mobile", "credential") still enter via their dedicated keyword searches.
func filterKeywordBridgeCandidates(reqTags []string, searchKeyword string, _ []string, objs []memory.MemoryObject) []memory.MemoryObject {
	if len(reqTags) == 0 || len(objs) == 0 {
		return objs
	}
	searchKeyword = strings.ToLower(strings.TrimSpace(searchKeyword))
	distinctive := len(searchKeyword) >= 6
	out := make([]memory.MemoryObject, 0, len(objs))
	for _, o := range objs {
		if tagMatchScore(o.Tags, reqTags) > 0 {
			out = append(out, o)
			continue
		}
		if distinctive && strings.Contains(strings.ToLower(o.Statement), searchKeyword) {
			out = append(out, o)
		}
	}
	return out
}

func filterLifecycleCandidates(mode RecallMode, objs []memory.MemoryObject) []memory.MemoryObject {
	if len(objs) == 0 {
		return objs
	}
	allowed := map[api.Status]struct{}{
		api.StatusActive:     {},
		api.StatusPending:    {},
		api.StatusSuperseded: {},
		api.StatusArchived:   {},
	}
	if mode == RecallModeCurrent {
		allowed = map[api.Status]struct{}{
			api.StatusActive:  {},
			api.StatusPending: {},
		}
	}
	out := objs[:0]
	for _, o := range objs {
		st := o.Status
		if st == "" {
			st = api.StatusActive
		}
		if st == api.StatusRejected || st == api.StatusQuarantined || st == api.StatusDeleted {
			continue
		}
		if mode == RecallModeCurrent {
			if st == api.StatusActive || st == api.StatusPending {
				out = append(out, o)
				continue
			}
			// Legacy keyword merge may have added superseded rows.
			if st == api.StatusSuperseded {
				out = append(out, o)
			}
			continue
		}
		if _, ok := allowed[st]; ok {
			out = append(out, o)
		}
	}
	return out
}
