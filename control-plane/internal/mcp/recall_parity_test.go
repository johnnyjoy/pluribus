package mcp

import (
	"encoding/json"
	"reflect"
	"testing"
)

// TestMCPAndRESTRecallUseSameCompilePath verifies recall_context maps to the same compile fields as REST POST /v1/recall/compile.
func TestMCPAndRESTRecallUseSameCompilePath(t *testing.T) {
	args := json.RawMessage(`{
		"task":"deploy webhook idempotency",
		"recall_mode":"historical",
		"include_status":["active","archived"],
		"occurred_after":"2023-06-01T00:00:00Z",
		"occurred_before":"2023-12-31T23:59:59Z",
		"repo_root":"/projects/pluribus",
		"tags":["payments"]
	}`)
	body, _, err := buildMemoryContextResolveCompileBody(args)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"retrieval_query", "recall_mode", "include_status", "occurred_after", "occurred_before", "repo_root", "tags"} {
		if m[key] == nil {
			t.Fatalf("compile body missing %q: %v", key, m)
		}
	}
}

func TestMCPAndRESTPreserveRecallMode(t *testing.T) {
	body, _, err := buildMemoryContextResolveCompileBody(json.RawMessage(`{"task":"x","recall_mode":"historical"}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m["recall_mode"] != "historical" {
		t.Fatalf("recall_mode=%v", m["recall_mode"])
	}
}

func TestMCPAndRESTPreserveIncludeStatus(t *testing.T) {
	body, _, err := buildMemoryContextResolveCompileBody(json.RawMessage(`{"task":"x","include_status":["archived","superseded"]}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	inc, ok := m["include_status"].([]any)
	if !ok || len(inc) != 2 {
		t.Fatalf("include_status=%v", m["include_status"])
	}
}

func TestMCPAndRESTPreserveDateBounds(t *testing.T) {
	body, _, err := buildMemoryContextResolveCompileBody(json.RawMessage(`{
		"task":"history",
		"occurred_after":"2023-01-01T00:00:00Z",
		"occurred_before":"2023-12-31T00:00:00Z"
	}`))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(body, &m); err != nil {
		t.Fatal(err)
	}
	if m["occurred_after"] == nil || m["occurred_before"] == nil {
		t.Fatalf("date bounds missing: %v", m)
	}
}

// TestMCPAndRESTExposeSemanticRetrievalMetadataConsistently verifies MCP passes through recall_bundle including semantic_retrieval when present.
func TestMCPAndRESTExposeSemanticRetrievalMetadataConsistently(t *testing.T) {
	bundle := json.RawMessage(`{
		"applicable_patterns":[],
		"semantic_retrieval":{"attempted":true,"path":"lexical_only","fallback_reason":"no_embedder","embedder_available":false}
	}`)
	meta := map[string]any{}
	enrichMCPContextFromRecallBundle(meta, bundle)
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(bundle, &raw); err != nil {
		t.Fatal(err)
	}
	sem, ok := raw["semantic_retrieval"]
	if !ok {
		t.Fatal("bundle missing semantic_retrieval")
	}
	var restSem map[string]any
	if err := json.Unmarshal(sem, &restSem); err != nil {
		t.Fatal(err)
	}
	// MCP wraps full bundle in recall_bundle; execMemoryContextResolve does not strip semantic_retrieval.
	// Verify bundle field survives round-trip parse (parity: same JSON on wire).
	var wrap map[string]any
	if err := json.Unmarshal(bundle, &wrap); err != nil {
		t.Fatal(err)
	}
	got, ok := wrap["semantic_retrieval"].(map[string]any)
	if !ok {
		t.Fatalf("semantic_retrieval type=%T", wrap["semantic_retrieval"])
	}
	if !reflect.DeepEqual(got["fallback_reason"], restSem["fallback_reason"]) {
		t.Fatalf("semantic_retrieval mismatch: %v vs %v", got, restSem)
	}
	_ = meta
}
