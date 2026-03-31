package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"control-plane/internal/mcp"
)

func TestToolDefinitions_includeCurationTools(t *testing.T) {
	tools := mcp.ToolDefinitions()
	want := map[string]bool{
		"recall_context":                 false,
		"record_experience":              false,
		"mcp_episode_ingest":             false,
		"curation_digest":                false,
		"curation_pending":               false,
		"curation_promotion_suggestions": false,
		"curation_strengthened":          false,
		"curation_materialize":           false,
		"enforcement_evaluate":           false,
	}
	for _, row := range tools {
		n, _ := row["name"].(string)
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for name, ok := range want {
		if !ok {
			t.Errorf("tools/list missing %q", name)
		}
	}
}

func TestParseCandidateID(t *testing.T) {
	good := json.RawMessage(`{"candidate_id":"22222222-2222-2222-2222-222222222222"}`)
	id, err := mcp.ParseCandidateID(good)
	if err != nil || id != "22222222-2222-2222-2222-222222222222" {
		t.Fatalf("got %q %v", id, err)
	}
	alias := json.RawMessage(`{"id":"33333333-3333-3333-3333-333333333333"}`)
	id, err = mcp.ParseCandidateID(alias)
	if err != nil || id != "33333333-3333-3333-3333-333333333333" {
		t.Fatalf("alias: got %q %v", id, err)
	}
	_, err = mcp.ParseCandidateID(json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for missing id")
	}
	_, err = mcp.ParseCandidateID(json.RawMessage(`{"candidate_id":"not-a-uuid"}`))
	if err == nil {
		t.Fatal("expected error for invalid uuid")
	}
}

func TestHandleToolsCall_curationDigest(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"proposals":[],"truncated":false}`))
	}))
	defer ts.Close()

	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"curation_digest","arguments":{"work_summary":"Enough text for validation and digest."}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/curation/digest" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(string(gotBody), "work_summary") {
		t.Fatalf("body: %s", gotBody)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result type %T", res)
	}
	if m["isError"] == true {
		t.Fatalf("unexpected error: %+v", res)
	}
}

func TestHandleToolsCall_enforcementEvaluate(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"decision":"allow","explanation":"ok","triggered_memories":[]}`))
	}))
	defer ts.Close()

	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"enforcement_evaluate","arguments":{"proposal_text":"We will add logging."}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/enforcement/evaluate" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if !strings.Contains(string(gotBody), "proposal_text") {
		t.Fatalf("body: %s", gotBody)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestValidateEnforcementArguments(t *testing.T) {
	err := mcp.ValidateEnforcementArguments(json.RawMessage(`{"tags":["x"],"proposal_text":"ok"}`))
	if err != nil {
		t.Fatalf("extra fields should be ignored by enforcement validation, got: %v", err)
	}
	err = mcp.ValidateEnforcementArguments(json.RawMessage(`{"proposal_text":""}`))
	if err == nil || !strings.Contains(err.Error(), "proposal_text") {
		t.Fatalf("expected proposal_text error: %v", err)
	}
	long := strings.Repeat("a", mcp.EnforcementMaxProposalBytes+1)
	payload, _ := json.Marshal(map[string]string{
		"proposal_text": long,
	})
	err = mcp.ValidateEnforcementArguments(payload)
	if err == nil || !strings.Contains(err.Error(), "max length") {
		t.Fatalf("expected max length error: %v", err)
	}
}

func TestHandleToolsCall_enforcementEvaluate_requiresArguments(t *testing.T) {
	params := json.RawMessage(`{"name":"enforcement_evaluate","arguments":{}}`)
	_, err := mcp.HandleToolsCall(http.DefaultClient, "http://127.0.0.1:9", "", params, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "proposal_text") {
		t.Fatalf("expected proposal_text error, got %v", err)
	}
}

func TestHandleToolsCall_curationDigest_requiresArguments(t *testing.T) {
	params := json.RawMessage(`{"name":"curation_digest","arguments":{}}`)
	_, err := mcp.HandleToolsCall(http.DefaultClient, "http://127.0.0.1:9", "", params, nil)
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "work_summary") {
		t.Fatalf("expected work_summary error, got %v", err)
	}
}

func TestValidateDigestArguments_workSummaryOnly(t *testing.T) {
	err := mcp.ValidateDigestArguments(json.RawMessage(`{"work_summary":"Enough text for validation and digest minimum length here."}`))
	if err != nil {
		t.Fatalf("expected ok with work_summary only: %v", err)
	}
}

func TestHandleToolsCall_curationMaterialize(t *testing.T) {
	cand := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	var gotMethod, gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"` + cand + `","kind":"constraint"}`))
	}))
	defer ts.Close()

	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"curation_materialize","arguments":{"candidate_id":"` + cand + `"}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/curation/candidates/"+cand+"/materialize" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	if len(gotBody) != 0 {
		t.Fatalf("materialize should not send body, got %q", gotBody)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestHandleToolsCall_curationMaterialize_http404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"candidate not found"}`, http.StatusNotFound)
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"curation_materialize","arguments":{"candidate_id":"11111111-1111-1111-1111-111111111111"}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := res.(map[string]any)
	if !ok {
		t.Fatalf("result type %T", res)
	}
	if m["isError"] != true {
		t.Fatalf("expected isError true: %+v", m)
	}
	content, _ := m["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatal("expected content")
	}
	text, _ := content[0]["text"].(string)
	if !strings.Contains(text, "404") || !strings.Contains(text, "candidate not found") {
		t.Fatalf("expected 404 body in text: %q", text)
	}
}

func TestHandleToolsCall_curationMaterialize_validationError(t *testing.T) {
	params := json.RawMessage(`{"name":"curation_materialize","arguments":{}}`)
	_, err := mcp.HandleToolsCall(http.DefaultClient, "http://127.0.0.1:9", "", params, nil)
	if err == nil || !strings.Contains(err.Error(), "candidate_id") {
		t.Fatalf("expected validation error, got %v", err)
	}
}

func TestHandleToolsCall_mcpEpisodeIngest_validationError(t *testing.T) {
	params := json.RawMessage(`{"name":"mcp_episode_ingest","arguments":{"summary":"short"}}`)
	res, err := mcp.HandleToolsCall(http.DefaultClient, "http://127.0.0.1:9", "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] != true {
		t.Fatalf("expected tool error result: %+v", res)
	}
}

func TestHandleToolsCall_mcpEpisodeIngest_postsAdvisory(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"11111111-1111-1111-1111-111111111111","advisory":true}`))
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	sum := "This deployment failed after we chose a rollback; learned to gate releases."
	params := json.RawMessage(`{"name":"mcp_episode_ingest","arguments":{"summary":` + jsonEscape(sum) + `,"event_kind":"failure","correlation_id":"sess-1"}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/advisory-episodes" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	var payload map[string]any
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["source"] != "mcp" {
		t.Fatalf("source: %v", payload["source"])
	}
	tags, _ := payload["tags"].([]any)
	found := false
	for _, x := range tags {
		if s, ok := x.(string); ok && strings.Contains(s, "mcp:event:") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected mcp:event tag, got %v", tags)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
	content, _ := m["content"].([]map[string]any)
	if len(content) == 0 {
		t.Fatal("expected content")
	}
	text, _ := content[0]["text"].(string)
	var body map[string]any
	if err := json.Unmarshal([]byte(text), &body); err != nil {
		t.Fatalf("parse tool result json: %v text=%q", err, text)
	}
	if _, ok := body["mcp_affordance"]; !ok {
		t.Fatalf("expected mcp_affordance in advisory response: %v", body)
	}
}

func TestHandleToolsCall_recordExperience_alias(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"22222222-2222-2222-2222-222222222222","advisory":true}`))
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	sum := "This deployment failed after we chose a rollback; learned to gate releases with enough signal tokens here."
	params := json.RawMessage(`{"name":"record_experience","arguments":{"summary":` + jsonEscape(sum) + `}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/v1/advisory-episodes" {
		t.Fatalf("got path %s", gotPath)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestHandleToolsCall_recallContext_alias(t *testing.T) {
	var gotPath, gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"recall_preamble":"ok","governing_constraints":[]}`))
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"recall_context","arguments":{"task_description":"ship the API with constraints"}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPost || gotPath != "/v1/recall/compile" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
	content, _ := m["content"].([]map[string]any)
	text, _ := content[0]["text"].(string)
	var wrap map[string]any
	if err := json.Unmarshal([]byte(text), &wrap); err != nil {
		t.Fatalf("parse: %v", err)
	}
	mc, _ := wrap["mcp_context"].(map[string]any)
	if mc["decision_hint"] == nil {
		t.Fatalf("expected decision_hint in mcp_context: %v", mc)
	}
	if mc["relevance_hint"] != nil {
		t.Fatalf("expected no relevance_hint for empty recall buckets: %v", mc["relevance_hint"])
	}
	wantWeak := "No strong prior memory found. Consider recording the outcome after completing this task."
	if mc["after_work_hint"] != wantWeak {
		t.Fatalf("after_work_hint: got %q want %q", mc["after_work_hint"], wantWeak)
	}
}

func TestHandleToolsCall_recallContext_strongPoolHints(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"governing_constraints":[{"statement":"Bind to TLS 1.2+"}]}`))
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"recall_context","arguments":{"task":"secure the edge API"}}`)

	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
	content, _ := m["content"].([]map[string]any)
	text, _ := content[0]["text"].(string)
	var wrap map[string]any
	if err := json.Unmarshal([]byte(text), &wrap); err != nil {
		t.Fatalf("parse: %v", err)
	}
	mc, _ := wrap["mcp_context"].(map[string]any)
	if mc["decision_hint"] == nil || mc["relevance_hint"] == nil || mc["after_work_hint"] == nil {
		t.Fatalf("expected decision_hint, relevance_hint, after_work_hint: %v", mc)
	}
	wantStrong := "After completing meaningful work, consider recording the outcome."
	if mc["after_work_hint"] != wantStrong {
		t.Fatalf("after_work_hint: got %q want %q", mc["after_work_hint"], wantStrong)
	}
}

func jsonEscape(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func TestHandleToolsCall_curationPending_get(t *testing.T) {
	var gotMethod, gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()
	base := strings.TrimSuffix(ts.URL, "/")
	params := json.RawMessage(`{"name":"curation_pending","arguments":{}}`)
	res, err := mcp.HandleToolsCall(ts.Client(), base, "", params, nil)
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodGet || gotPath != "/v1/curation/pending" {
		t.Fatalf("got %s %s", gotMethod, gotPath)
	}
	m, ok := res.(map[string]any)
	if !ok || m["isError"] == true {
		t.Fatalf("unexpected: %+v", res)
	}
}

func TestBuildRecallGetURL(t *testing.T) {
	base := "http://127.0.0.1:8123"
	arg := json.RawMessage(`{"retrieval_query":"ship feature","tags":["x","y"],"max_per_kind":3}`)
	u, err := mcp.BuildRecallGetURL(base, arg)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(u)
	if err != nil {
		t.Fatal(err)
	}
	q := parsed.Query()
	if q.Get("retrieval_query") == "" {
		t.Errorf("retrieval_query: got %q", q.Get("retrieval_query"))
	}
	tags := q["tags"]
	if len(tags) != 2 {
		t.Errorf("tags: %v", tags)
	}
	if q.Get("max_per_kind") != "3" {
		t.Errorf("max_per_kind: %v", q.Get("max_per_kind"))
	}
}
