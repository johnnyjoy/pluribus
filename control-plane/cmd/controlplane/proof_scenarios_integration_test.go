//go:build integration
// +build integration

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/enforcement"
	"control-plane/internal/proofscenarios"
	"control-plane/internal/recall"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

// TestIntegration_proofScenarioSuite runs YAML-defined integration proof scenarios (benefit receipts).
// Requires TEST_PG_DSN (same as make regression). Set RECALL_PROOF_RESULTS_OUT to write a markdown summary.
func TestIntegration_proofScenarioSuite(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn

	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	defer container.DB.Close()

	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()
	base := srv.URL

	dir := filepath.Join(controlPlaneModuleRoot(t), "proof-scenarios")
	scenarios, err := proofscenarios.LoadDir(dir)
	if err != nil {
		t.Fatalf("LoadDir: %v", err)
	}
	if err := proofscenarios.ValidateUniqueIDs(scenarios); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, sc := range scenarios {
		if err := proofscenarios.Validate(&sc); err != nil {
			t.Fatalf("validate %s: %v", sc.ID, err)
		}
		if sc.Mode == proofscenarios.ModeIntegration {
			ids = append(ids, sc.ID)
		}
	}
	sort.Strings(ids)

	runners := map[string]func(*testing.T, string){
		"anti-drift-known-bad-pattern":        runProofAntiDriftKnownBad,
		"continuity-second-step-from-first":   runProofContinuitySecondStep,
		"curation-digest-materialize-durable": runProofCurationDigestMaterialize,
		"curation-then-recall-continuity":     runProofCurationThenRecall,
		"enforcement-sqlite-forbidden":        runProofEnforcementSQLiteForbidden,
		"enforcement-unrelated-allow":         runProofEnforcementUnrelatedAllow,
		"recall-binding-constraint-surfaces":  runProofRecallBindingConstraint,
		"recall-decision-relevant-to-work":    runProofRecallDecisionRelevant,
		"simulated-multi-agent-continuity":    runProofSimulatedMultiAgentContinuity,
	}

	var rows []proofscenarios.ResultRow
	for _, id := range ids {
		fn, ok := runners[id]
		if !ok {
			t.Fatalf("no integration runner for scenario id %q (add to runners map)", id)
		}
		start := time.Now()
		okRun := t.Run(id, func(t *testing.T) {
			fn(t, base)
		})
		rows = append(rows, proofscenarios.ResultRow{
			ScenarioID: id,
			Pass:       okRun,
			Duration:   time.Since(start),
		})
	}

	if p := strings.TrimSpace(os.Getenv("RECALL_PROOF_RESULTS_OUT")); p != "" {
		env := "integration TEST_PG_DSN"
		if err := proofscenarios.WriteMarkdownSummary(p, env, rows); err != nil {
			t.Logf("write proof results: %v", err)
		}
	}
}

// --- HTTP helpers ---

func postJSON(t *testing.T, url string, body string) *http.Response {
	t.Helper()
	return postJSONClient(t, http.DefaultClient, url, body)
}

func postJSONClient(t *testing.T, client *http.Client, urlStr string, body string) *http.Response {
	t.Helper()
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Post(urlStr, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", urlStr, err)
	}
	return resp
}

func getJSON(t *testing.T, url string) *http.Response {
	t.Helper()
	return getJSONClient(t, http.DefaultClient, url)
}

func getJSONClient(t *testing.T, client *http.Client, urlStr string) *http.Response {
	t.Helper()
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodGet, urlStr, nil)
	if err != nil {
		t.Fatalf("NewRequest GET %s: %v", urlStr, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("GET %s: %v", urlStr, err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) []byte {
	t.Helper()
	b, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return b
}

func createConstraintPostgres(t *testing.T, base string) {
	t.Helper()
	body := `{"kind":"constraint","authority":9,"statement":"All durable project data must use Postgres; SQLite is not permitted."}`
	resp := postJSON(t, base+"/v1/memory", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("POST memory status=%d body=%s", resp.StatusCode, b)
	}
}

func postEnforcement(t *testing.T, base string, proposal, intent string) enforcement.EvaluateResponse {
	t.Helper()
	body := fmt.Sprintf(`{"proposal_text":%q,"intent":%q}`, proposal, intent)
	resp := postJSON(t, base+"/v1/enforcement/evaluate", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("enforcement status=%d body=%s", resp.StatusCode, b)
	}
	var out enforcement.EvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode enforcement: %v", err)
	}
	return out
}

func compileRecall(t *testing.T, base string, taskID uuid.UUID) recall.RecallBundle {
	t.Helper()
	return compileRecallClient(t, http.DefaultClient, base, taskID)
}

func compileRecallClient(t *testing.T, client *http.Client, base string, taskID uuid.UUID) recall.RecallBundle {
	t.Helper()
	return compileRecallClientWithRetrieval(t, client, base, taskID, "")
}

func compileRecallClientWithRetrieval(t *testing.T, client *http.Client, base string, _ uuid.UUID, retrievalQuery string) recall.RecallBundle {
	t.Helper()
	var body string
	if retrievalQuery != "" {
		body = fmt.Sprintf(`{"retrieval_query":%q}`, retrievalQuery)
	} else {
		body = `{}`
	}
	resp := postJSONClient(t, client, base+"/v1/recall/compile", body)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("recall compile status=%d body=%s", resp.StatusCode, b)
	}
	var bundle recall.RecallBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	return bundle
}

func bundleHasKindSubstring(b recall.RecallBundle, kind, substr string) bool {
	substr = strings.ToLower(substr)
	check := func(items []recall.MemoryItem) bool {
		for _, it := range items {
			if it.Kind == kind && strings.Contains(strings.ToLower(it.Statement), substr) {
				return true
			}
		}
		return false
	}
	return check(b.GoverningConstraints) || check(b.Decisions) || check(b.KnownFailures) ||
		check(b.ApplicablePatterns)
}

// --- Scenario runners ---

func runProofEnforcementSQLiteForbidden(t *testing.T, base string) {
	createConstraintPostgres(t, base)
	out := postEnforcement(t, base, "We will migrate durable storage to SQLite.", "datastore")
	if out.Decision != enforcement.DecisionBlock {
		t.Fatalf("decision=%q want block", out.Decision)
	}
	if len(out.TriggeredMemories) < 1 {
		t.Fatalf("expected triggered_memories")
	}
	if out.TriggeredMemories[0].ReasonCode != "normative_conflict" {
		t.Fatalf("reason_code=%q want normative_conflict", out.TriggeredMemories[0].ReasonCode)
	}
}

func runProofEnforcementUnrelatedAllow(t *testing.T, base string) {
	createConstraintPostgres(t, base)
	out := postEnforcement(t, base, "We will add a metrics dashboard.", "change")
	if out.Decision != enforcement.DecisionAllow {
		t.Fatalf("decision=%q want allow", out.Decision)
	}
}

func runProofAntiDriftKnownBad(t *testing.T, base string) {
	// Same behavioral receipt as enforcement-sqlite-forbidden; distinct benefit_claim in YAML.
	runProofEnforcementSQLiteForbidden(t, base)
}

func runProofRecallBindingConstraint(t *testing.T, base string) {
	createConstraintPostgres(t, base)
	b := compileRecall(t, base, uuid.Nil)
	if !bundleHasKindSubstring(b, "constraint", "Postgres") {
		t.Fatalf("expected constraint about Postgres in bundle, got constraints=%+v", b.GoverningConstraints)
	}
}

func runProofRecallDecisionRelevant(t *testing.T, base string) {
	body := `{"kind":"decision","authority":7,"statement":"We will use feature flags for rollout of the new API surface."}`
	resp2 := postJSON(t, base+"/v1/memory", body)
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		b := readBody(t, resp2)
		t.Fatalf("POST decision memory status=%d body=%s", resp2.StatusCode, b)
	}
	b := compileRecallClientWithRetrieval(t, http.DefaultClient, base, uuid.Nil, "feature flags rollout API")
	if !bundleHasKindSubstring(b, "decision", "feature flags") {
		t.Fatalf("expected decision about feature flags in bundle, got decisions=%+v", b.Decisions)
	}
}

func runProofCurationDigestMaterialize(t *testing.T, base string) {
	digestBody := `{"work_summary":"Proof scenario work summary for digest pipeline minimum length.","curation_answers":{"decision":"We will use feature flags for rollout of the new API surface."}}`
	resp := postJSON(t, base+"/v1/curation/digest", digestBody)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("digest status=%d body=%s", resp.StatusCode, b)
	}
	var dr struct {
		Proposals []struct {
			CandidateID string `json:"candidate_id"`
			Kind        string `json:"kind"`
		} `json:"proposals"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		t.Fatalf("decode digest: %v", err)
	}
	if len(dr.Proposals) < 1 || dr.Proposals[0].CandidateID == "" {
		t.Fatalf("expected at least one proposal with candidate_id")
	}
	candID := dr.Proposals[0].CandidateID
	if dr.Proposals[0].Kind != string(api.MemoryKindDecision) {
		t.Fatalf("kind=%q want decision", dr.Proposals[0].Kind)
	}
	matURL := fmt.Sprintf("%s/v1/curation/candidates/%s/materialize", base, candID)
	mresp := postJSON(t, matURL, "{}")
	defer mresp.Body.Close()
	if mresp.StatusCode != http.StatusCreated {
		b := readBody(t, mresp)
		t.Fatalf("materialize status=%d body=%s", mresp.StatusCode, b)
	}
	var mobj struct {
		Kind string `json:"kind"`
	}
	if err := json.NewDecoder(mresp.Body).Decode(&mobj); err != nil {
		t.Fatalf("decode memory: %v", err)
	}
	if mobj.Kind != string(api.MemoryKindDecision) {
		t.Fatalf("memory kind=%q want decision", mobj.Kind)
	}
}

func runProofCurationThenRecall(t *testing.T, base string) {
	digestBody := `{"work_summary":"Proof scenario work summary for digest pipeline minimum length.","curation_answers":{"decision":"We will use feature flags for rollout of the new API surface."}}`
	resp := postJSON(t, base+"/v1/curation/digest", digestBody)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("digest status=%d body=%s", resp.StatusCode, b)
	}
	var dr struct {
		Proposals []struct {
			CandidateID string `json:"candidate_id"`
		} `json:"proposals"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		t.Fatalf("decode digest: %v", err)
	}
	candID := dr.Proposals[0].CandidateID
	mresp := postJSON(t, base+"/v1/curation/candidates/"+candID+"/materialize", "{}")
	defer mresp.Body.Close()
	if mresp.StatusCode != http.StatusCreated {
		b := readBody(t, mresp)
		t.Fatalf("materialize status=%d body=%s", mresp.StatusCode, b)
	}
	b := compileRecall(t, base, uuid.Nil)
	if !bundleHasKindSubstring(b, "decision", "feature flags") {
		t.Fatalf("expected materialized decision in recall bundle")
	}
}

// runProofSimulatedMultiAgentContinuity proves two distinct HTTP clients share the same global memory pool:
// A writes a tagged marker constraint; B recalls from a fresh HTTP client and still surfaces it (global memory pool).
func runProofSimulatedMultiAgentContinuity(t *testing.T, base string) {
	agentA := &http.Client{}
	agentB := &http.Client{}
	marker := fmt.Sprintf("SIM-MA-CONTINUITY-%s", uuid.New().String())
	tag := fmt.Sprintf("proof-multi-agent-%s", strings.ReplaceAll(uuid.New().String(), "-", ""))

	memBody := fmt.Sprintf(`{"kind":"constraint","authority":9,"statement":%q,"tags":["%s"]}`, marker, tag)
	respMem := postJSONClient(t, agentA, base+"/v1/memory", memBody)
	defer respMem.Body.Close()
	if respMem.StatusCode != http.StatusOK {
		b := readBody(t, respMem)
		t.Fatalf("Agent A POST memory status=%d body=%s", respMem.StatusCode, b)
	}
	bundleA := compileRecallClient(t, agentA, base, uuid.Nil)
	if !bundleHasKindSubstring(bundleA, "constraint", marker) {
		t.Fatalf("Agent A recall: expected marker in bundle, constraints=%+v", bundleA.GoverningConstraints)
	}

	// POST /v1/memory/search uses memory.SearchRequest (tags/status/max/kinds only — no query field).
	// Tag is unique to this proof run; sufficient to retrieve the marker memory.
	searchBody := fmt.Sprintf(`{"tags":["%s"]}`, tag)
	respSearch := postJSONClient(t, agentB, base+"/v1/memory/search", searchBody)
	defer respSearch.Body.Close()
	if respSearch.StatusCode != http.StatusOK {
		b := readBody(t, respSearch)
		t.Fatalf("Agent B memory search status=%d body=%s", respSearch.StatusCode, b)
	}
	var found []struct {
		Statement string `json:"statement"`
	}
	if err := json.NewDecoder(respSearch.Body).Decode(&found); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	ok := false
	for _, m := range found {
		if strings.Contains(m.Statement, marker) {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("Agent B search: expected marker memory, got %+v", found)
	}

	bundleB := compileRecallClient(t, agentB, base, uuid.Nil)
	if !bundleHasKindSubstring(bundleB, "constraint", marker) {
		t.Fatalf("Agent B recall: expected same marker in bundle, constraints=%+v", bundleB.GoverningConstraints)
	}
}

func runProofContinuitySecondStep(t *testing.T, base string) {
	digestBody := `{"work_summary":"Proof scenario work summary for digest pipeline minimum length.","curation_answers":{"decision":"We will use feature flags for rollout of the new API surface."}}`
	resp := postJSON(t, base+"/v1/curation/digest", digestBody)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("digest status=%d body=%s", resp.StatusCode, b)
	}
	var dr struct {
		Proposals []struct {
			CandidateID string `json:"candidate_id"`
		} `json:"proposals"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dr); err != nil {
		t.Fatalf("decode digest: %v", err)
	}
	candID := dr.Proposals[0].CandidateID
	mresp := postJSON(t, base+"/v1/curation/candidates/"+candID+"/materialize", "{}")
	defer mresp.Body.Close()
	if mresp.StatusCode != http.StatusCreated {
		b := readBody(t, mresp)
		t.Fatalf("materialize status=%d body=%s", mresp.StatusCode, b)
	}
	b1 := compileRecall(t, base, uuid.Nil)
	if !bundleHasKindSubstring(b1, "decision", "feature flags") {
		t.Fatalf("first recall: expected decision")
	}
	b2 := compileRecall(t, base, uuid.Nil)
	if !bundleHasKindSubstring(b2, "decision", "feature flags") {
		t.Fatalf("second recall: expected decision (continuity)")
	}
}
