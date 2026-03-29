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
	"strings"
	"testing"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/enforcement"
	"control-plane/internal/recall"

	"github.com/google/uuid"
)

// TestREST_memoryCreate_rejectsContainerOntologyJSON proves DisallowUnknownFields rejects dead container keys on POST /v1/memory and POST /v1/memories.
func TestREST_memoryCreate_rejectsContainerOntologyJSON(t *testing.T) {
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	badFragments := []struct {
		name, jsonField string
	}{
		{"project_id", `"project_id":"550e8400-e29b-41d4-a716-446655440000"`},
		{"task_id", `"task_id":"550e8400-e29b-41d4-a716-446655440001"`},
		{"hive_id", `"hive_id":"550e8400-e29b-41d4-a716-446655440002"`},
		{"workspace_id", `"workspace_id":"acme"`},
		{"scope_id", `"scope_id":"global"`},
	}
	for _, tc := range badFragments {
		t.Run(tc.name, func(t *testing.T) {
			base := `{"kind":"decision","statement":"ontology guard","authority":5,"tags":["rest:boundary"]`
			body := base + `,` + tc.jsonField + `}`
			for _, ep := range []string{"/v1/memory", "/v1/memories"} {
				resp, err := http.Post(srv.URL+ep, "application/json", strings.NewReader(body))
				if err != nil {
					t.Fatalf("POST %s: %v", ep, err)
				}
				b, _ := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if resp.StatusCode != http.StatusBadRequest {
					t.Fatalf("POST %s with %s: want 400, got %d body=%s", ep, tc.jsonField, resp.StatusCode, string(b))
				}
				if !strings.Contains(string(b), "invalid JSON") {
					t.Fatalf("expected invalid JSON error in body: %s", string(b))
				}
			}
		})
	}
}

// TestREST_recallCompile_returnsShapedBundle asserts POST /v1/recall/compile returns RecallBundle-shaped JSON after seeding memory via REST.
func TestREST_recallCompile_returnsShapedBundle(t *testing.T) {
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	tag := "rest:compile:" + uuid.NewString()
	createBody := fmt.Sprintf(`{"kind":"constraint","statement":"Use Postgres for durable data in REST compile test","authority":8,"tags":[%q]}`, tag)
	resp, err := http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /v1/memories: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create memory: status=%d %s", resp.StatusCode, string(b))
	}

	compileBody := fmt.Sprintf(`{"retrieval_query":"Postgres durable","tags":[%q],"max_per_kind":3}`, tag)
	resp, err = http.Post(srv.URL+"/v1/recall/compile", "application/json", strings.NewReader(compileBody))
	if err != nil {
		t.Fatalf("POST compile: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("compile: status=%d %s", resp.StatusCode, string(b))
	}
	var bundle recall.RecallBundle
	if err := json.NewDecoder(resp.Body).Decode(&bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	// Shaped product boundary: grouped slices exist (may be empty).
	if bundle.GoverningConstraints == nil || bundle.Decisions == nil || bundle.KnownFailures == nil || bundle.ApplicablePatterns == nil {
		t.Fatalf("expected non-nil legacy buckets, got %+v", bundle)
	}
	if bundle.Continuity == nil || bundle.Constraints == nil || bundle.Experience == nil {
		t.Fatalf("expected non-nil continuity/constraints/experience groups, got cont=%v cons=%v exp=%v",
			bundle.Continuity, bundle.Constraints, bundle.Experience)
	}
}

// TestREST_recallPreflight_returnsRiskShape exercises POST /v1/recall/preflight on the full router.
func TestREST_recallPreflight_returnsRiskShape(t *testing.T) {
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/v1/recall/preflight", "application/json", strings.NewReader(`{}`))
	if err != nil {
		t.Fatalf("preflight: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("preflight status=%d %s", resp.StatusCode, string(b))
	}
	var out recall.PreflightResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.RiskLevel == "" {
		t.Fatal("expected risk_level")
	}
	if out.RequiredActions == nil {
		t.Fatal("expected required_actions slice (may be empty)")
	}
}

// TestREST_recallCompileMulti_minimal exercises POST /v1/recall/compile-multi returns variant bundles.
func TestREST_recallCompileMulti_minimal(t *testing.T) {
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	body := `{"retrieval_query":"smoke compile multi","variants":2,"max_per_kind":2}`
	resp, err := http.Post(srv.URL+"/v1/recall/compile-multi", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("compile-multi: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("compile-multi status=%d %s", resp.StatusCode, string(b))
	}
	var out recall.CompileMultiResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out.Bundles) < 1 {
		t.Fatalf("expected at least one variant bundle, got %d", len(out.Bundles))
	}
}

// TestREST_enforcementEvaluate_fullRouter seeds memory via POST /v1/memories then evaluates via full apiserver router.
func TestREST_enforcementEvaluate_fullRouter(t *testing.T) {
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	tag := "rest:enforce:" + uuid.NewString()
	// Enforcement normativeConflict (rule-based v1) detects Postgres-vs-SQLite clashes; keep the test aligned with evaluator.go.
	stmt := "Production must use Postgres only for durable data."
	createBody := fmt.Sprintf(`{"kind":"constraint","statement":%q,"authority":9,"tags":[%q]}`, stmt, tag)
	resp, err := http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST memories: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create: status=%d %s", resp.StatusCode, string(b))
	}

	evBody := `{"proposal_text":"We will use SQLite for durable data in production.","intent":"datastore"}`
	resp, err = http.Post(srv.URL+"/v1/enforcement/evaluate", "application/json", strings.NewReader(evBody))
	if err != nil {
		t.Fatalf("POST evaluate: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("evaluate status=%d %s", resp.StatusCode, string(b))
	}
	var out enforcement.EvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Decision != enforcement.DecisionBlock && out.Decision != enforcement.DecisionRequireReview && out.Decision != enforcement.DecisionBlockOverrideable {
		t.Fatalf("expected non-allow decision for conflicting proposal, got %q validation=%+v", out.Decision, out.Validation)
	}
	if len(out.TriggeredMemories) < 1 {
		t.Fatalf("expected triggered_memories, got %+v", out)
	}
}

// TestREST_curationDigest_dryRun exercises POST /v1/curation/digest with dry_run (no persistence required for proposals path).
func TestREST_curationDigest_dryRun(t *testing.T) {
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	body := `{"work_summary":"Shipped curation digest REST test; learned to keep dry_run for CI.","curation_answers":{"decision":"Use dry_run in tests"},"options":{"dry_run":true}}`
	resp, err := http.Post(srv.URL+"/v1/curation/digest", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("digest: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("digest status=%d %s", resp.StatusCode, string(b))
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := m["proposals"]; !ok {
		t.Fatalf("digest JSON missing proposals key: %s", string(raw))
	}
}
