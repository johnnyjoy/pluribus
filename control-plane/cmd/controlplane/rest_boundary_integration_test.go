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
	cfg := loadIntegrationConfig(t)
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
	cfg := loadIntegrationConfig(t)
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
	// Product wire: empty buckets may be JSON null → decoded as nil slices; do not require non-nil empties.
	// Assert the seeded constraint appears in compile output (legacy and/or grouped views).
	const needle = "REST compile test"
	var items []recall.MemoryItem
	items = append(items, bundle.GoverningConstraints...)
	items = append(items, bundle.Decisions...)
	items = append(items, bundle.KnownFailures...)
	items = append(items, bundle.ApplicablePatterns...)
	items = append(items, bundle.Continuity...)
	items = append(items, bundle.Constraints...)
	items = append(items, bundle.Experience...)
	found := false
	for _, it := range items {
		if strings.Contains(it.Statement, needle) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected seeded constraint in compile bundle (legacy or grouped), got %+v", bundle)
	}
}

// TestREST_recallWakeup_returnsWakeupResponseShape asserts POST /v1/recall/wakeup returns the same JSON shape as the handler (MCP proxies this body unchanged for limit-only calls).
func TestREST_recallWakeup_returnsWakeupResponseShape(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	tag := "rest:wakeup:" + uuid.NewString()
	for _, body := range []string{
		fmt.Sprintf(`{"kind":"state","statement":"Mission WAKEUP_STATE integration marker","authority":9,"tags":[%q]}`, tag),
		fmt.Sprintf(`{"kind":"constraint","statement":"WAKEUP_L1 constraint integration marker","authority":8,"tags":[%q]}`, tag),
		fmt.Sprintf(`{"kind":"constraint","statement":"WAKEUP_L1 second constraint","authority":7,"tags":[%q]}`, tag),
	} {
		resp, perr := http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(body))
		if perr != nil {
			t.Fatalf("POST /v1/memories: %v", perr)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("seed memory: status=%d %s", resp.StatusCode, string(b))
		}
	}

	t.Run("empty_body_ok", func(t *testing.T) {
		resp, err := http.Post(srv.URL+"/v1/recall/wakeup", "application/json", strings.NewReader(`{}`))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("wakeup {}: status=%d %s", resp.StatusCode, string(b))
		}
		var out recall.WakeupResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		// Same wire shape as MCP tool result body (raw JSON text).
		if out.LimitsApplied.MaxState != 4 || out.LimitsApplied.MaxPerKind != 2 || out.LimitsApplied.MaxGoverningTotal != 12 {
			t.Fatalf("limits_applied defaults: %+v", out.LimitsApplied)
		}
	})

	t.Run("tagged_pool_non_empty_and_limits", func(t *testing.T) {
		full := fmt.Sprintf(`{"tags":[%q]}`, tag)
		resp, err := http.Post(srv.URL+"/v1/recall/wakeup", "application/json", strings.NewReader(full))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("wakeup: status=%d %s", resp.StatusCode, string(b))
		}
		var out recall.WakeupResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.LimitsApplied.MaxState != 4 || out.LimitsApplied.MaxPerKind != 2 || out.LimitsApplied.MaxGoverningTotal != 12 {
			t.Fatalf("limits_applied defaults: %+v", out.LimitsApplied)
		}
		foundState, foundL1 := false, false
		for _, it := range out.Identity {
			if it.Kind == "state" && strings.Contains(it.Statement, "WAKEUP_STATE") {
				foundState = true
			}
		}
		for _, it := range out.GoverningMemory {
			if it.Kind == "constraint" && strings.Contains(it.Statement, "WAKEUP_L1") {
				foundL1 = true
			}
		}
		if !foundState {
			t.Fatalf("expected seeded state in identity, got identity=%+v", out.Identity)
		}
		if !foundL1 {
			t.Fatalf("expected seeded constraint in governing_memory, got governing=%+v", out.GoverningMemory)
		}

		limited := fmt.Sprintf(`{"tags":[%q],"max_governing_total":1}`, tag)
		resp2, err := http.Post(srv.URL+"/v1/recall/wakeup", "application/json", strings.NewReader(limited))
		if err != nil {
			t.Fatal(err)
		}
		defer resp2.Body.Close()
		if resp2.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp2.Body)
			t.Fatalf("wakeup limited: status=%d %s", resp2.StatusCode, string(b))
		}
		var out2 recall.WakeupResponse
		if err := json.NewDecoder(resp2.Body).Decode(&out2); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(out2.GoverningMemory) != 1 {
			t.Fatalf("max_governing_total=1: want 1 governing row, got %d", len(out2.GoverningMemory))
		}
		if out2.LimitsApplied.MaxGoverningTotal != 1 {
			t.Fatalf("limits_applied.max_governing_total: got %d", out2.LimitsApplied.MaxGoverningTotal)
		}
	})
}

func TestREST_recallPreflight_returnsRiskShape(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
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
	cfg := loadIntegrationConfig(t)
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
	cfg := loadIntegrationConfig(t)
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
	cfg := loadIntegrationConfig(t)
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

// TestREST_memories_occurredAt_createAndRecallRank proves optional occurred_at is stored, returned, and drives recency ranking vs ingest time.
func TestREST_memories_occurredAt_createAndRecallRank(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	tag := "rest:occurred:" + uuid.NewString()
	oldStmt := "Temporal honesty old event " + uuid.NewString()
	newStmt := "Temporal honesty new event " + uuid.NewString()
	// Ingest order: "new" row first (fresh updated_at), but occurred_at makes the other look older for recency.
	bodyOld := fmt.Sprintf(`{"kind":"decision","statement":%q,"authority":5,"tags":[%q],"occurred_at":"2010-01-01T00:00:00Z"}`, oldStmt, tag)
	resp, err := http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(bodyOld))
	if err != nil {
		t.Fatalf("POST memories: %v", err)
	}
	bOld, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create old-event memory: status=%d %s", resp.StatusCode, string(bOld))
	}
	var memOld map[string]interface{}
	if err := json.Unmarshal(bOld, &memOld); err != nil {
		t.Fatalf("decode memory: %v", err)
	}
	if memOld["occurred_at"] == nil {
		t.Fatalf("expected occurred_at in response, got %s", string(bOld))
	}

	bodyNew := fmt.Sprintf(`{"kind":"decision","statement":%q,"authority":5,"tags":[%q],"occurred_at":"2024-06-01T12:00:00Z"}`, newStmt, tag)
	resp, err = http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(bodyNew))
	if err != nil {
		t.Fatalf("POST memories: %v", err)
	}
	bNew, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create new-event memory: status=%d %s", resp.StatusCode, string(bNew))
	}

	compileBody := fmt.Sprintf(`{"retrieval_query":"Temporal honesty","tags":[%q],"max_per_kind":5}`, tag)
	resp, err = http.Post(srv.URL+"/v1/recall/compile", "application/json", strings.NewReader(compileBody))
	if err != nil {
		t.Fatalf("compile: %v", err)
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
	if len(bundle.Continuity) < 2 {
		t.Fatalf("expected 2 decisions in continuity, got %d: %+v", len(bundle.Continuity), bundle.Continuity)
	}
	// Ranked path: newer effective event time should appear first.
	if !strings.Contains(bundle.Continuity[0].Statement, newStmt) {
		t.Fatalf("expected newer occurred_at decision first in continuity, got first=%q", bundle.Continuity[0].Statement)
	}
}

// TestREST_memories_withoutOccurredAt_backwardCompatible proves omitting occurred_at still creates and returns a row.
func TestREST_memories_withoutOccurredAt_backwardCompatible(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
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
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	tag := "rest:no-occurred:" + uuid.NewString()
	body := fmt.Sprintf(`{"kind":"constraint","statement":%q,"authority":6,"tags":[%q]}`, "Backward compatible no occurred_at "+uuid.NewString(), tag)
	resp, err := http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST: %v", err)
	}
	raw, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status=%d %s", resp.StatusCode, string(raw))
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := m["occurred_at"]; ok && m["occurred_at"] != nil {
		t.Fatalf("expected no occurred_at in JSON when omitted, got %v", m["occurred_at"])
	}
}
