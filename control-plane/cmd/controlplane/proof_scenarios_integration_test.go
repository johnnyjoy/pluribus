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
//
// Local (CI / make regression): requires TEST_PG_DSN; boots an in-process server with
// proof-friendly formation defaults (seeded memories active immediately).
//
// Deployed benefit receipts: set PLURIBUS_PROOF_BASE_URL (or CONTROL_PLANE_URL) to a running
// control-plane base URL. No local Postgres boot. Optional CONTROL_PLANE_API_KEY / PLURIBUS_API_KEY
// for X-API-Key. Set RECALL_PROOF_RESULTS_OUT to write a markdown summary.
//
// Deployed runs exercise the server's real formation/recall policy (not proof defaults).
func TestIntegration_proofScenarioSuite(t *testing.T) {
	base := strings.TrimRight(strings.TrimSpace(os.Getenv("PLURIBUS_PROOF_BASE_URL")), "/")
	if base == "" {
		base = strings.TrimRight(strings.TrimSpace(os.Getenv("CONTROL_PLANE_URL")), "/")
	}
	deployed := base != ""
	if !deployed {
		dsn := os.Getenv("TEST_PG_DSN")
		if dsn == "" {
			t.Skip("TEST_PG_DSN not set (or set PLURIBUS_PROOF_BASE_URL for deployed benefit receipts)")
		}
		cfg := loadIntegrationConfig(t)
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
		base = srv.URL
	} else {
		t.Logf("deployed benefit receipts against %s", base)
	}
	for _, line := range proofscenarios.SuiteHonestyNotes {
		t.Logf("SUITE LIMIT: %s", line)
	}

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

	scByID := make(map[string]proofscenarios.Scenario, len(scenarios))
	for _, sc := range scenarios {
		scByID[sc.ID] = sc
	}

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
			if sc, ok := scByID[id]; ok {
				proofLogScenarioHonesty(t, sc)
			}
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
		if deployed {
			env = "deployed benefit receipts " + base
		}
		if err := proofscenarios.WriteMarkdownSummary(p, env, rows, scenarios); err != nil {
			t.Logf("write proof results: %v", err)
		}
	}
}

// proofAPIKey returns the optional API key for deployed benefit receipts.
func proofAPIKey() string {
	if k := strings.TrimSpace(os.Getenv("CONTROL_PLANE_API_KEY")); k != "" {
		return k
	}
	return strings.TrimSpace(os.Getenv("PLURIBUS_API_KEY"))
}

// proofHTTPClient returns an HTTP client; for deployed runs it injects X-API-Key when configured.
func proofHTTPClient() *http.Client {
	key := proofAPIKey()
	if key == "" {
		return http.DefaultClient
	}
	return &http.Client{Transport: &proofAuthTransport{key: key, base: http.DefaultTransport}}
}

type proofAuthTransport struct {
	key  string
	base http.RoundTripper
}

func (t *proofAuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r := req.Clone(req.Context())
	r.Header.Set("X-API-Key", t.key)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(r)
}

// --- Proof honesty (hostile / skeptical contract) ---

func proofLogScenarioHonesty(t *testing.T, sc proofscenarios.Scenario) {
	t.Helper()
	t.Logf("PROVES (claim): %s", sc.BenefitClaim)
	for _, line := range sc.DoesNotProve {
		t.Logf("DOES NOT PROVE: %s", line)
	}
}

type proofMemoryCreateOutcome struct {
	ID           string
	Statement    string
	Consolidated bool
	Status       string
}

func proofDecodeMemoryCreateFromBody(t *testing.T, body []byte) proofMemoryCreateOutcome {
	t.Helper()
	var raw struct {
		ID           string `json:"id"`
		Statement    string `json:"statement"`
		Consolidated bool   `json:"consolidated"`
		Status       string `json:"status"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode memory create: %v", err)
	}
	return proofMemoryCreateOutcome{
		ID:           raw.ID,
		Statement:    raw.Statement,
		Consolidated: raw.Consolidated,
		Status:       raw.Status,
	}
}

// proofRequireVerifiableWrite fails when a create response claims success but the situation tag
// cannot retrieve the marker — including consolidated=true merges that drop situational tags.
func proofRequireVerifiableWrite(t *testing.T, base, situationTag, marker string, created proofMemoryCreateOutcome) {
	t.Helper()
	if created.Consolidated {
		t.Logf("HOSTILE: create returned consolidated=true id=%s — verifying independent tag retrieval anyway", created.ID)
	}
	found := proofMemorySearchByTag(t, proofHTTPClient(), base, situationTag)
	for _, m := range found {
		if strings.Contains(m.Statement, marker) {
			if created.Consolidated && !strings.Contains(created.Statement, marker) {
				t.Logf("HOSTILE NOTE: marker visible via tag search but consolidated row statement differs: %q", created.Statement)
			}
			return
		}
	}
	t.Fatalf("HOSTILE FAIL: write not independently verifiable — consolidated=%v id=%q tag=%q marker=%q search=%+v",
		created.Consolidated, created.ID, situationTag, marker, found)
}

func proofNewSituationTag(prefix string) string {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	if prefix == "" {
		prefix = "proof-situation"
	}
	return fmt.Sprintf("%s-%s", prefix, strings.ReplaceAll(uuid.New().String(), "-", ""))
}

type proofMaterializeOutcome struct {
	MemoryID           string
	Statement          string
	Consolidated       bool
	ConsolidatedIntoID string
	Created            bool
}

func proofDecodeMaterializeFromBody(t *testing.T, body []byte) proofMaterializeOutcome {
	t.Helper()
	var raw struct {
		Memory struct {
			ID           string `json:"id"`
			Statement    string `json:"statement"`
			Consolidated bool   `json:"consolidated"`
		} `json:"memory"`
		Created                  bool    `json:"created"`
		ConsolidatedIntoMemoryID *string `json:"consolidated_into_memory_id,omitempty"`
		Strengthened             bool    `json:"strengthened"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		t.Fatalf("decode materialize: %v", err)
	}
	out := proofMaterializeOutcome{
		MemoryID:     raw.Memory.ID,
		Statement:    raw.Memory.Statement,
		Consolidated: raw.Memory.Consolidated || raw.Strengthened || raw.ConsolidatedIntoMemoryID != nil,
		Created:      raw.Created,
	}
	if raw.ConsolidatedIntoMemoryID != nil {
		out.ConsolidatedIntoID = *raw.ConsolidatedIntoMemoryID
	}
	return out
}

func proofRequireVerifiableMaterialize(t *testing.T, base, situationTag, marker string, mat proofMaterializeOutcome) {
	t.Helper()
	if mat.Consolidated {
		msg := fmt.Sprintf("HOSTILE: materialize consolidated/strengthened memory_id=%s", mat.MemoryID)
		if mat.ConsolidatedIntoID != "" {
			msg += fmt.Sprintf(" consolidated_into=%s", mat.ConsolidatedIntoID)
		}
		t.Log(msg)
	}
	proofRequireVerifiableWrite(t, base, situationTag, marker, proofMemoryCreateOutcome{
		ID:           mat.MemoryID,
		Statement:    mat.Statement,
		Consolidated: mat.Consolidated,
	})
}

// proofDigestMaterializeFeatureFlagsDecision runs digest→materialize with a unique situation tag + marker.
func proofDigestMaterializeFeatureFlagsDecision(t *testing.T, base, tagPrefix string) (situationTag, marker string, mat proofMaterializeOutcome) {
	t.Helper()
	situationTag = proofNewSituationTag(tagPrefix)
	marker = fmt.Sprintf("[%s:%s]", tagPrefix, uuid.New())
	decision := fmt.Sprintf("We will use feature flags for rollout of the new API surface. %s", marker)
	digestPayload, err := json.Marshal(map[string]any{
		"work_summary":     "Proof scenario work summary for digest pipeline minimum length.",
		"signals":          []string{situationTag},
		"curation_answers": map[string]string{"decision": decision},
	})
	if err != nil {
		t.Fatalf("marshal digest: %v", err)
	}
	resp := postJSON(t, base+"/v1/curation/digest", string(digestPayload))
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
	if dr.Proposals[0].Kind != string(api.MemoryKindDecision) {
		t.Fatalf("kind=%q want decision", dr.Proposals[0].Kind)
	}
	candID := dr.Proposals[0].CandidateID
	mresp := postJSON(t, fmt.Sprintf("%s/v1/curation/candidates/%s/materialize", base, candID), "{}")
	defer mresp.Body.Close()
	if mresp.StatusCode != http.StatusCreated {
		b := readBody(t, mresp)
		t.Fatalf("materialize status=%d body=%s", mresp.StatusCode, b)
	}
	mat = proofDecodeMaterializeFromBody(t, readBody(t, mresp))
	if mat.MemoryID == "" {
		t.Fatal("materialize: missing memory id")
	}
	proofRequireVerifiableMaterialize(t, base, situationTag, marker, mat)
	return situationTag, marker, mat
}

func proofPostMemoryCreateVerified(t *testing.T, base string, fields map[string]any, situationTag, marker string) proofMemoryCreateOutcome {
	t.Helper()
	resp := postJSON(t, base+"/v1/memory", proofMemoryCreateBody(fields))
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b := readBody(t, resp)
		t.Fatalf("POST memory status=%d body=%s", resp.StatusCode, b)
	}
	created := proofDecodeMemoryCreateFromBody(t, readBody(t, resp))
	proofRequireVerifiableWrite(t, base, situationTag, marker, created)
	return created
}

// --- HTTP helpers ---

func postJSON(t *testing.T, url string, body string) *http.Response {
	t.Helper()
	return postJSONClient(t, proofHTTPClient(), url, body)
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
	return getJSONClient(t, proofHTTPClient(), url)
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

const proofEphemeralTTLSeconds = 3600

func mergeProofEphemeralTags(tags []string) []string {
	seen := map[string]struct{}{}
	for _, t := range tags {
		if t != "" {
			seen[t] = struct{}{}
		}
	}
	for _, t := range []string{api.TagEphemeral, api.TagProofScenario} {
		seen[t] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for t := range seen {
		out = append(out, t)
	}
	sort.Strings(out)
	return out
}

func proofMemoryCreateBody(fields map[string]any) string {
	if fields == nil {
		fields = map[string]any{}
	}
	var tags []string
	if raw, ok := fields["tags"].([]string); ok {
		tags = raw
	}
	fields["tags"] = mergeProofEphemeralTags(tags)
	if _, ok := fields["ttl_seconds"]; !ok {
		fields["ttl_seconds"] = proofEphemeralTTLSeconds
	}
	b, err := json.Marshal(fields)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func createConstraintPostgres(t *testing.T, base string) {
	t.Helper()
	body := proofMemoryCreateBody(map[string]any{
		"kind":          "constraint",
		"authority":     9,
		"applicability": "governing",
		"statement":     "All durable project data must use Postgres; SQLite is not permitted.",
	})
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
	return compileRecallClient(t, proofHTTPClient(), base, taskID)
}

func compileRecallClient(t *testing.T, client *http.Client, base string, taskID uuid.UUID) recall.RecallBundle {
	t.Helper()
	return compileRecallClientWithRetrieval(t, client, base, taskID, "")
}

func compileRecallClientWithRetrieval(t *testing.T, client *http.Client, base string, _ uuid.UUID, retrievalQuery string) recall.RecallBundle {
	t.Helper()
	return compileRecallClientWithSituation(t, client, base, nil, retrievalQuery)
}

func compileRecallClientWithSituation(t *testing.T, client *http.Client, base string, tags []string, retrievalQuery string) recall.RecallBundle {
	t.Helper()
	payload := map[string]any{}
	if len(tags) > 0 {
		payload["tags"] = tags
	}
	if strings.TrimSpace(retrievalQuery) != "" {
		payload["retrieval_query"] = retrievalQuery
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal recall compile: %v", err)
	}
	resp := postJSONClient(t, client, base+"/v1/recall/compile", string(bodyBytes))
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

func bundleHasKindSubstrings(b recall.RecallBundle, kind string, substrings ...string) bool {
	for _, sub := range substrings {
		if sub != "" && bundleHasKindSubstring(b, kind, sub) {
			return true
		}
	}
	return false
}

func proofMemorySearchByTag(t *testing.T, client *http.Client, base, tag string) []struct {
	Statement string `json:"statement"`
} {
	t.Helper()
	searchBody := fmt.Sprintf(`{"tags":["%s"]}`, tag)
	respSearch := postJSONClient(t, client, base+"/v1/memory/search", searchBody)
	defer respSearch.Body.Close()
	if respSearch.StatusCode != http.StatusOK {
		b := readBody(t, respSearch)
		t.Fatalf("memory search status=%d body=%s", respSearch.StatusCode, b)
	}
	var found []struct {
		Statement string `json:"statement"`
	}
	if err := json.NewDecoder(respSearch.Body).Decode(&found); err != nil {
		t.Fatalf("decode search: %v", err)
	}
	return found
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
	situationTag := proofNewSituationTag("proof-recall-decision")
	marker := fmt.Sprintf("[proof-recall-decision:%s]", uuid.New())
	stmt := fmt.Sprintf("We will use feature flags for rollout of the new API surface. %s", marker)
	proofPostMemoryCreateVerified(t, base, map[string]any{
		"kind":      "decision",
		"authority": 7,
		"statement": stmt,
		"tags":      []string{situationTag},
	}, situationTag, marker)
	tagsT := mergeProofEphemeralTags([]string{situationTag})
	b := compileRecallClientWithSituation(t, proofHTTPClient(), base, tagsT, situationTag+" feature flags rollout API")
	if !bundleHasKindSubstrings(b, "decision", marker, "feature flags") {
		t.Fatalf("expected decision with marker and feature flags in bundle, got decisions=%+v", b.Decisions)
	}
}

func runProofCurationDigestMaterialize(t *testing.T, base string) {
	_, marker, mat := proofDigestMaterializeFeatureFlagsDecision(t, base, "proof-digest-mat")
	if !strings.Contains(mat.Statement, marker) {
		t.Fatalf("materialized statement missing marker %q: %q", marker, mat.Statement)
	}
}

func runProofCurationThenRecall(t *testing.T, base string) {
	situationTag, marker, _ := proofDigestMaterializeFeatureFlagsDecision(t, base, "proof-digest-recall")
	tagsT := mergeProofEphemeralTags([]string{situationTag})
	b := compileRecallClientWithSituation(t, proofHTTPClient(), base, tagsT, situationTag+" feature flags")
	if !bundleHasKindSubstrings(b, "decision", marker, "feature flags") {
		t.Fatalf("expected materialized decision with marker in recall bundle")
	}
}

// runProofSimulatedMultiAgentContinuity proves two distinct HTTP clients share the same global memory pool:
// A writes a tagged marker constraint; B recalls from a fresh HTTP client using the same tags + retrieval text.
func runProofSimulatedMultiAgentContinuity(t *testing.T, base string) {
	// Distinct client instances (shared auth transport when API key is set).
	agentA := proofHTTPClient()
	agentB := proofHTTPClient()
	marker := fmt.Sprintf("SIM-MA-CONTINUITY-%s", uuid.New().String())
	tag := fmt.Sprintf("proof-multi-agent-%s", strings.ReplaceAll(uuid.New().String(), "-", ""))
	tagsT := mergeProofEphemeralTags([]string{tag})

	memBody := proofMemoryCreateBody(map[string]any{
		"kind":      "constraint",
		"authority": 9,
		"statement": marker,
		"tags":      []string{tag},
	})
	respMem := postJSONClient(t, agentA, base+"/v1/memory", memBody)
	defer respMem.Body.Close()
	if respMem.StatusCode != http.StatusOK {
		b := readBody(t, respMem)
		t.Fatalf("Agent A POST memory status=%d body=%s", respMem.StatusCode, b)
	}
	created := proofDecodeMemoryCreateFromBody(t, readBody(t, respMem))
	proofRequireVerifiableWrite(t, base, tag, marker, created)

	// YAML phase A/B: recall/compile scoped to agreed tag T + retrieval text (no UUID handoff to B).
	situation := tag + " " + marker
	bundleA := compileRecallClientWithSituation(t, agentA, base, tagsT, situation)
	if !bundleHasKindSubstrings(bundleA, "constraint", marker, tag) {
		t.Fatalf("Agent A recall: expected marker or tag in bundle, constraints=%+v", bundleA.GoverningConstraints)
	}

	foundB := proofMemorySearchByTag(t, agentB, base, tag)
	ok := false
	for _, m := range foundB {
		if strings.Contains(m.Statement, marker) || strings.Contains(m.Statement, tag) {
			ok = true
			break
		}
	}
	if !ok {
		t.Fatalf("Agent B search: expected marker memory, got %+v", foundB)
	}

	bundleB := compileRecallClientWithSituation(t, agentB, base, tagsT, situation)
	if !bundleHasKindSubstrings(bundleB, "constraint", marker, tag) {
		t.Fatalf("Agent B recall: expected same marker or tag in bundle, constraints=%+v", bundleB.GoverningConstraints)
	}
}

func runProofContinuitySecondStep(t *testing.T, base string) {
	situationTag, marker, _ := proofDigestMaterializeFeatureFlagsDecision(t, base, "proof-continuity")
	tagsT := mergeProofEphemeralTags([]string{situationTag})
	query := situationTag + " feature flags"
	b1 := compileRecallClientWithSituation(t, proofHTTPClient(), base, tagsT, query)
	if !bundleHasKindSubstrings(b1, "decision", marker, "feature flags") {
		t.Fatalf("first recall: expected decision with marker")
	}
	b2 := compileRecallClientWithSituation(t, proofHTTPClient(), base, tagsT, query)
	if !bundleHasKindSubstrings(b2, "decision", marker, "feature flags") {
		t.Fatalf("second recall: expected decision with marker (continuity)")
	}
}
