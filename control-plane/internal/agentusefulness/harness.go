package agentusefulness

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"control-plane/internal/mcp"
	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/internal/recall/benchmark"
	"control-plane/internal/utility"

	"github.com/google/uuid"
)

var historicalRoles = map[string]struct{}{
	recall.LifecycleSupersededContext: {},
	recall.LifecycleArchivedContext:   {},
	recall.LifecycleRefutedContext:    {},
	recall.LifecycleOutdatedContext:   {},
	recall.LifecycleHistoricalContext: {},
}

// NewCompiler builds a recall compiler for task memory pool.
func NewCompiler(objects []memory.MemoryObject, utilScores map[string]utility.Score, labelToID map[string]string) *recall.Compiler {
	w := recall.DefaultRankingWeights()
	c := &recall.Compiler{
		Memory:        &benchmark.MemoryStub{All: objects},
		Ranking:       &w,
		UtilityWeight: 0.12,
	}
	if len(utilScores) > 0 {
		scores := map[uuid.UUID]float64{}
		summaries := map[uuid.UUID]utility.Score{}
		for label, us := range utilScores {
			if idStr, ok := labelToID[label]; ok {
				if u, err := uuid.Parse(idStr); err == nil {
					scores[u] = us.UtilityScore
					summaries[u] = us
				}
			}
		}
		c.Utility = &fixtureUtilityProvider{scores: scores, summaries: summaries}
	}
	return c
}

type fixtureUtilityProvider struct {
	scores    map[uuid.UUID]float64
	summaries map[uuid.UUID]utility.Score
}

func (f *fixtureUtilityProvider) GetScoresForMemories(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]float64, error) {
	out := map[uuid.UUID]float64{}
	for _, id := range ids {
		if v, ok := f.scores[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

func (f *fixtureUtilityProvider) GetUtilitySummaries(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]utility.Score, error) {
	out := map[uuid.UUID]utility.Score{}
	for _, id := range ids {
		if v, ok := f.summaries[id]; ok {
			out[id] = v
		}
	}
	return out, nil
}

// BuildCompileRequest maps task fixture to compile request.
func BuildCompileRequest(t TaskFixture) recall.CompileRequest {
	req := recall.CompileRequest{
		RetrievalQuery: t.RecallQuery,
		Tags:           append([]string(nil), t.DomainTags...),
		MaxPerKind:     10,
		MaxTotal:       40,
	}
	if t.RecallMode != "" {
		req.RecallMode = t.RecallMode
	}
	if len(t.IncludeStatus) > 0 {
		req.IncludeStatus = append([]string(nil), t.IncludeStatus...)
	}
	if t.OccurredAfter != "" {
		req.OccurredAfter = t.OccurredAfter
	}
	if t.OccurredBefore != "" {
		req.OccurredBefore = t.OccurredBefore
	}
	return req
}

// RecallREST compiles via direct REST-equivalent compile path.
func RecallREST(ctx context.Context, compiler *recall.Compiler, req recall.CompileRequest) (*recall.RecallBundle, error) {
	return compiler.Compile(ctx, req)
}

// RecallMCP builds MCP compile body then compiles (same path as recall_context).
func RecallMCP(ctx context.Context, compiler *recall.Compiler, t TaskFixture) (*recall.CompileRequest, *recall.RecallBundle, error) {
	// MCP maps `task` → compile retrieval_query (see context_resolve.go).
	taskText := strings.TrimSpace(t.RecallQuery)
	if taskText == "" {
		taskText = t.TaskPrompt
	}
	args := map[string]any{
		"task": taskText,
	}
	if len(t.DomainTags) > 0 {
		args["tags"] = t.DomainTags
	}
	if t.RecallMode != "" {
		args["recall_mode"] = t.RecallMode
	}
	if len(t.IncludeStatus) > 0 {
		args["include_status"] = t.IncludeStatus
	}
	if t.OccurredAfter != "" {
		args["occurred_after"] = t.OccurredAfter
	}
	if t.OccurredBefore != "" {
		args["occurred_before"] = t.OccurredBefore
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return nil, nil, err
	}
	body, _, err := mcp.BuildMemoryContextResolveCompileBodyForTest(raw)
	if err != nil {
		return nil, nil, err
	}
	var req recall.CompileRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return nil, nil, err
	}
	if req.RetrievalQuery == "" {
		req.RetrievalQuery = t.RecallQuery
	}
	if len(req.Tags) == 0 {
		req.Tags = append([]string(nil), t.DomainTags...)
	}
	req.MaxPerKind = 10
	req.MaxTotal = 40
	bundle, err := compiler.Compile(ctx, req)
	return &req, bundle, err
}

// FlattenBundle collects all memory items from a bundle in stable order.
func FlattenBundle(b *recall.RecallBundle) []recall.MemoryItem {
	if b == nil {
		return nil
	}
	var items []recall.MemoryItem
	for _, s := range [][]recall.MemoryItem{
		b.GoverningConstraints, b.Decisions, b.KnownFailures, b.ApplicablePatterns,
		b.Continuity, b.Constraints, b.Experience,
	} {
		items = append(items, s...)
	}
	sort.Slice(items, func(i, j int) bool {
		si, sj := scoreOf(items[i]), scoreOf(items[j])
		if si != sj {
			return si > sj
		}
		return items[i].ID < items[j].ID
	})
	return items
}

func scoreOf(it recall.MemoryItem) float64 {
	if it.Justification != nil {
		return it.Justification.Score
	}
	return 0
}

// BuildManifest constructs recall manifest from bundle.
func BuildManifest(taskID, runID, iface, mode string, req recall.CompileRequest, b *recall.RecallBundle, lc *LoadedCorpus) *RecallManifest {
	m := &RecallManifest{
		TaskID:        taskID,
		RunID:         runID,
		Interface:     iface,
		Mode:          mode,
		RecallRequest: req,
	}
	items := FlattenBundle(b)
	for i, it := range items {
		entry := RecalledMemoryEntry{
			MemoryID: it.ID,
			Label:    lc.LabelForID(it.ID),
			Rank:     i + 1,
			LifecycleRole: it.LifecycleRole,
			Status:        it.Status,
			Why:           it.WhyMatters,
		}
		if it.Justification != nil {
			entry.Score = it.Justification.Score
			if entry.Why == "" {
				entry.Why = it.Justification.Reason
			}
		}
		if it.UtilityScore != nil {
			entry.UtilityScore = *it.UtilityScore
		}
		m.Recalled = append(m.Recalled, entry)
	}
	return m
}

// RecalledLabels returns sorted labels from manifest.
func RecalledLabels(m *RecallManifest) []string {
	if m == nil {
		return nil
	}
	var out []string
	for _, e := range m.Recalled {
		if e.Label != "" {
			out = append(out, e.Label)
		}
	}
	sort.Strings(out)
	return out
}

// RecalledLabelSet returns set of recalled labels.
func RecalledLabelSet(m *RecallManifest) map[string]struct{} {
	out := map[string]struct{}{}
	for _, lbl := range RecalledLabels(m) {
		out[lbl] = struct{}{}
	}
	return out
}

// SameRecalledIDs compares REST and MCP manifests (order tolerant).
func SameRecalledIDs(a, b *RecallManifest) (bool, string) {
	if a == nil || b == nil {
		return false, "missing manifest"
	}
	setA := map[string]struct{}{}
	setB := map[string]struct{}{}
	for _, e := range a.Recalled {
		setA[e.MemoryID] = struct{}{}
	}
	for _, e := range b.Recalled {
		setB[e.MemoryID] = struct{}{}
	}
	if len(setA) != len(setB) {
		return false, fmt.Sprintf("recalled count rest=%d mcp=%d", len(setA), len(setB))
	}
	for id := range setA {
		if _, ok := setB[id]; !ok {
			return false, fmt.Sprintf("mcp missing recalled id %s", id)
		}
	}
	return true, ""
}

// domainOverlap checks task domain tags against memory item tags.
func domainOverlap(taskTags []string, memTags []string) bool {
	if len(taskTags) == 0 {
		return true
	}
	set := map[string]struct{}{}
	for _, t := range memTags {
		set[t] = struct{}{}
	}
	for _, t := range taskTags {
		if _, ok := set[t]; ok {
			return true
		}
	}
	return false
}

// isCurrentMode reports whether recall mode treats rows as current guidance.
func isCurrentMode(mode string) bool {
	mode = strings.ToLower(strings.TrimSpace(mode))
	return mode == "" || mode == "current"
}

// itemByLabel finds recalled item by label.
func itemByLabel(items []recall.MemoryItem, lc *LoadedCorpus, label string) *recall.MemoryItem {
	id := lc.IDForLabel(label)
	for i := range items {
		if items[i].ID == id {
			return &items[i]
		}
	}
	return nil
}

// containsLabel checks slice membership.
func containsLabel(list []string, label string) bool {
	for _, l := range list {
		if l == label {
			return true
		}
	}
	return false
}
