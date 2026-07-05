package mcp

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToolDefinitions_recallRecordLoopDescriptions(t *testing.T) {
	tools := ToolDefinitions()
	var recallDesc, recordDesc string
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		switch name {
		case "recall_context":
			recallDesc, _ = tool["description"].(string)
		case "record_experience":
			recordDesc, _ = tool["description"].(string)
		}
	}
	for _, pair := range []struct {
		desc, needle string
	}{
		{recallDesc, "start of a substantive task"},
		{recallDesc, "prior constraints"},
		{recordDesc, "after solving"},
		{recordDesc, "meaningful outcomes"},
	} {
		if !strings.Contains(pair.desc, pair.needle) {
			t.Fatalf("description must contain %q, got %q", pair.needle, pair.desc)
		}
	}
}

func TestInitializeResult_memoryLoopInstructions(t *testing.T) {
	res := InitializeResult("test", "0.0.0")
	inst, _ := res["instructions"].(string)
	if !strings.Contains(inst, "wakeup_context") || !strings.Contains(inst, "recall_context") || !strings.Contains(inst, "record_experience") || !strings.Contains(inst, "resolve_chore") || !strings.Contains(inst, "memory_feedback") {
		t.Fatalf("instructions: %q", inst)
	}
}

func TestToolDefinitions_noProjectCRUDTools(t *testing.T) {
	tools := ToolDefinitions()
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		if name == "project_get_by_slug" || name == "project_list" || name == "project_create" {
			t.Fatalf("project-first MCP tool should not be exposed: %s", name)
		}
	}
}

func TestToolDefinitions_behaviorFirstAndCompatAliases(t *testing.T) {
	tools := ToolDefinitions()
	found := map[string]string{}
	order := []string{}
	for _, tool := range tools {
		name, _ := tool["name"].(string)
		desc, _ := tool["description"].(string)
		if strings.TrimSpace(desc) == "" {
			t.Fatalf("tool %q has empty description", name)
		}
		found[name] = desc
		order = append(order, name)
	}
	for _, name := range []string{"recall_context", "record_experience", "memory_context_resolve", "mcp_episode_ingest"} {
		if _, ok := found[name]; !ok {
			t.Fatalf("tools/list must include %q", name)
		}
	}
	// Primary loop tools should appear before compatibility aliases in tools/list.
	iRecall := indexOf(order, "recall_context")
	iCompatRecall := indexOf(order, "memory_context_resolve")
	iRecord := indexOf(order, "record_experience")
	iCompatRecord := indexOf(order, "mcp_episode_ingest")
	if iRecall < 0 || iCompatRecall < 0 || iRecord < 0 || iCompatRecord < 0 {
		t.Fatal("missing tool in order slice")
	}
	if iRecall >= iCompatRecall {
		t.Fatalf("recall_context should sort before memory_context_resolve: %v", order)
	}
	if iRecord >= iCompatRecord {
		t.Fatalf("record_experience should sort before mcp_episode_ingest: %v", order)
	}
	if indexOf(order, "health") != len(order)-1 {
		t.Fatalf("health should be last tool in list, got order: %v", order)
	}
}

func indexOf(slice []string, s string) int {
	for i, v := range slice {
		if v == s {
			return i
		}
	}
	return -1
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

func TestBuildWakeupContextBody(t *testing.T) {
	for _, tc := range []struct {
		raw  string
		want string
	}{
		{``, `{}`},
		{`   `, `{}`},
		{`null`, `{}`},
		{`{}`, `{}`},
		{`{"max_state":3}`, `{"max_state":3}`},
		{`{"max_per_kind":2,"max_governing_total":8}`, `{"max_per_kind":2,"max_governing_total":8}`},
	} {
		b, err := buildWakeupContextBody(json.RawMessage(tc.raw))
		if err != nil {
			t.Fatalf("raw=%q: %v", tc.raw, err)
		}
		if string(b) != tc.want {
			t.Fatalf("raw=%q: got %s want %s", tc.raw, b, tc.want)
		}
	}
}

func TestHandleToolsCall_wakeupContext_forwardsPOST(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The wakeup wrapper also fetches the optional housekeeping chore;
		// record only the wakeup call itself.
		if r.URL.Path == "/v1/curation/chores" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"chores":[]}`))
			return
		}
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"identity":[],"governing_memory":[]}`))
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"wakeup_context","arguments":{"max_state":2}}`)
	res, err := HandleToolsCall(ts.Client(), base, "", params, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != "POST" || gotPath != "/v1/recall/wakeup" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if string(gotBody) != `{"max_state":2}` {
		t.Fatalf("body: %s", gotBody)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result type %T", res)
	}
	if m["isError"] == true {
		t.Fatalf("isError: %+v", m)
	}
}

func TestToolDefinitions_wakeupContext_schemaStrict(t *testing.T) {
	for _, tool := range ToolDefinitions() {
		name, _ := tool["name"].(string)
		if name != "wakeup_context" {
			continue
		}
		schema, _ := tool["inputSchema"].(map[string]any)
		if schema["additionalProperties"] != false {
			t.Fatalf("wakeup_context inputSchema.additionalProperties want false, got %v", schema["additionalProperties"])
		}
		props, _ := schema["properties"].(map[string]any)
		if len(props) != 3 {
			t.Fatalf("expected 3 properties, got %d", len(props))
		}
		for _, k := range []string{"max_state", "max_per_kind", "max_governing_total"} {
			if props[k] == nil {
				t.Fatalf("missing property %q", k)
			}
		}
		return
	}
	t.Fatal("wakeup_context not in ToolDefinitions")
}
