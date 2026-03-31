package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestInferContextStrategy_override(t *testing.T) {
	s, mode, lim := inferContextStrategy("anything", "thread")
	if s != ctxStrategyEpisodic || mode != "thread" || lim != nil {
		t.Fatalf("got strategy=%s mode=%s lim=%v", s, mode, lim)
	}
}

func TestInferContextStrategy_constraintHeuristic(t *testing.T) {
	s, _, lim := inferContextStrategy("We must not deploy without policy review and compliance check", "")
	if s != ctxStrategyConstraint {
		t.Fatalf("want constraint focus, got %s", s)
	}
	if lim == nil || lim["constraints"] == nil {
		t.Fatalf("want constraint limits: %v", lim)
	}
}

func TestInferContextStrategy_continuityTieredLimits(t *testing.T) {
	_, _, lim := inferContextStrategy("generic work item without strong cues", "")
	if lim == nil {
		t.Fatal("expected default continuity tiered limits")
	}
	if lim["constraints"] == nil || lim["failures"] == nil || lim["patterns"] == nil {
		t.Fatalf("expected constraints/failures/patterns in limits: %v", lim)
	}
}

func TestBuildMemoryContextResolveCompileBody_correlationID(t *testing.T) {
	b, meta, err := buildMemoryContextResolveCompileBody(json.RawMessage(`{"task":"x","correlation_id":"sess-99"}`))
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["correlation_id"] != "sess-99" {
		t.Fatalf("compile body: %v", out["correlation_id"])
	}
	if meta["correlation_id"] != "sess-99" {
		t.Fatalf("meta: %v", meta["correlation_id"])
	}
}

func TestBuildMemoryContextResolveCompileBody_taskAlias(t *testing.T) {
	b, _, err := buildMemoryContextResolveCompileBody(json.RawMessage(`{"task":"fix the flaky timeout test"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["retrieval_query"] != "fix the flaky timeout test" {
		t.Fatalf("retrieval_query: %v", m["retrieval_query"])
	}
}

func TestBuildMemoryContextResolveCompileBody(t *testing.T) {
	b, meta, err := buildMemoryContextResolveCompileBody(json.RawMessage(`{"task_description":"error timeout incident","entities":["api"]}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["retrieval_query"] == nil {
		t.Fatalf("missing retrieval_query: %s", string(b))
	}
	if meta["strategy"] == nil {
		t.Fatalf("meta: %v", meta)
	}
	tags, _ := m["tags"].([]any)
	if len(tags) < 1 || tags[0] != "api" {
		t.Fatalf("want entities merged to tags: %v", m["tags"])
	}
}

func TestTextBlockHasRelevantSignals(t *testing.T) {
	if !textBlockHasRelevantSignals(strings.Repeat("x", 20) + " error during deploy") {
		t.Fatal("expected signal")
	}
	if textBlockHasRelevantSignals("short") {
		t.Fatal("expected skip short")
	}
	if !textBlockHasRelevantSignals(strings.Repeat("a", 12) + " we tried again and again") {
		t.Fatal("expected repetition phrase")
	}
}

func TestEnrichMCPContextFromRecallBundle(t *testing.T) {
	meta := map[string]any{"retrieval_query": "q"}
	bundle := json.RawMessage(`{
		"governing_constraints":[{"statement":"Never deploy Friday","justification":{"reason":"tag_match","score":1}}],
		"known_failures":[],
		"applicable_patterns":[]
	}`)
	enrichMCPContextFromRecallBundle(meta, bundle)
	if meta["why_now"] == nil {
		t.Fatal("expected why_now")
	}
	why := meta["why_now"].(string)
	if why == "" || !strings.Contains(why, "Constraint") {
		t.Fatalf("why_now: %q", why)
	}
	if meta["primary_signal"] != "tag_match" {
		t.Fatalf("primary_signal: %v", meta["primary_signal"])
	}
}

func TestApplyMCPRecallBehaviorHints_weakPool(t *testing.T) {
	meta := map[string]any{}
	bundle := json.RawMessage(`{"governing_constraints":[],"known_failures":[],"applicable_patterns":[]}`)
	enrichMCPContextFromRecallBundle(meta, bundle)
	applyMCPRecallBehaviorHints(meta)
	if meta["decision_hint"] != mcpDecisionHintAlways {
		t.Fatalf("decision_hint: %v", meta["decision_hint"])
	}
	if meta["relevance_hint"] != nil {
		t.Fatalf("expected no relevance_hint when pool empty, got %v", meta["relevance_hint"])
	}
	if meta["after_work_hint"] != mcpAfterWorkHintWeakPool {
		t.Fatalf("after_work_hint: %v", meta["after_work_hint"])
	}
}

func TestApplyMCPRecallBehaviorHints_strongPool(t *testing.T) {
	meta := map[string]any{}
	bundle := json.RawMessage(`{"governing_constraints":[{"statement":"Use Postgres"}],"known_failures":[]}`)
	enrichMCPContextFromRecallBundle(meta, bundle)
	applyMCPRecallBehaviorHints(meta)
	if meta["decision_hint"] != mcpDecisionHintAlways {
		t.Fatalf("decision_hint: %v", meta["decision_hint"])
	}
	if meta["relevance_hint"] != mcpRelevanceHintStrong {
		t.Fatalf("relevance_hint: %v", meta["relevance_hint"])
	}
	if meta["after_work_hint"] != mcpAfterWorkHintStrongPool {
		t.Fatalf("after_work_hint: %v", meta["after_work_hint"])
	}
}

func TestApplyMCPRecallBehaviorHints_singleMcpContextKeys(t *testing.T) {
	meta := map[string]any{}
	bundle := json.RawMessage(`{"continuity":[{"statement":"resume"}]}`)
	enrichMCPContextFromRecallBundle(meta, bundle)
	applyMCPRecallBehaviorHints(meta)
	for _, k := range []string{"decision_hint", "relevance_hint", "after_work_hint"} {
		if _, ok := meta[k]; !ok {
			t.Fatalf("missing %q in mcp_context fields", k)
		}
	}
}
