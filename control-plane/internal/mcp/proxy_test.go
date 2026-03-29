package mcp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestToolDefinitions_noProjectCRUDTools(t *testing.T) {
	tools := ToolDefinitions()
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == "project_get_by_slug" || name == "project_list" || name == "project_create" {
			t.Fatalf("project-first MCP tool should not be exposed: %s", name)
		}
	}
}

func TestBuildRecallGetURL_tagsWithoutLegacyIDs(t *testing.T) {
	url, err := BuildRecallGetURL("http://localhost:8123", json.RawMessage(`{"retrieval_query":"ship API","tags":["go"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(url, "context_id=") || strings.Contains(url, "target_id=") {
		t.Fatalf("recall URL must not include legacy correlation query keys: %s", url)
	}
	if !strings.Contains(url, "tags=go") {
		t.Fatalf("expected tags in URL: %s", url)
	}
	if !strings.Contains(url, "retrieval_query=") {
		t.Fatalf("expected retrieval_query in URL: %s", url)
	}
}

func TestBuildRecallGetURL_retrievalQuery(t *testing.T) {
	u, err := BuildRecallGetURL("http://localhost:8123", json.RawMessage(`{"retrieval_query":"ship the API safely","tags":["go"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "retrieval_query=") {
		t.Fatalf("expected retrieval_query in URL: %s", u)
	}
}

func TestBuildRecallGetURL_queryAlias(t *testing.T) {
	u, err := BuildRecallGetURL("http://localhost:8123", json.RawMessage(`{"query":"same as retrieval","tags":["x"]}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(u, "retrieval_query=same+as+retrieval") && !strings.Contains(u, "retrieval_query=same%20as%20retrieval") {
		t.Fatalf("expected query mapped to retrieval_query param: %s", u)
	}
}
