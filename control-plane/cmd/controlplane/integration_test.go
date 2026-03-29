//go:build integration
// +build integration

package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/curation"
	"control-plane/internal/enforcement"
	"control-plane/internal/httpx"
	"control-plane/internal/memory"
	"control-plane/pkg/api"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	_ "github.com/lib/pq"
)

// integrationConfigPath returns a path to the example config. go test runs with cwd = the
// package directory (e.g. cmd/controlplane), so relative "configs/..." from LoadConfig's default
// would fail; we walk up to the module root (go.mod). Override with CONFIG=/abs/path.yaml if needed.
func integrationConfigPath(t *testing.T) string {
	t.Helper()
	if p := strings.TrimSpace(os.Getenv("CONFIG")); p != "" {
		return p
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return filepath.Join(dir, "configs", "config.example.yaml")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from cwd %s (set CONFIG to an absolute config path)", wd)
		}
		dir = parent
	}
}

// controlPlaneModuleRoot returns the directory containing go.mod (control-plane module root).
func controlPlaneModuleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found from cwd %s", wd)
		}
		dir = parent
	}
}

// Integration test: create candidate, promote into behavior memory, verify memory and candidate status.
func TestIntegration_promoteCandidateToPattern(t *testing.T) {
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

	ctx := context.Background()
	curationRepo := &curation.Repo{DB: container.DB}
	memoryRepo := &memory.Repo{DB: container.DB}
	memorySvc := &memory.Service{Repo: memoryRepo}
	curationSvc := &curation.Service{Repo: curationRepo, Memory: memorySvc}

	cand, err := curationRepo.Create(ctx, "We must always run tests before deploy.", 0.7)
	if err != nil {
		t.Fatalf("create candidate: %v", err)
	}
	if cand.PromotionStatus != "pending" {
		t.Errorf("candidate status = %q, want pending", cand.PromotionStatus)
	}

	payload := &memory.PatternPayload{
		Polarity:   "negative",
		Experience: "Deployed without tests.",
		Decision:   "Require tests before deploy",
		Outcome:    "Regression in production",
		Impact:     memory.PatternImpact{Severity: "high"},
		Directive:  "Always run tests before deploy.",
	}
	obj, err := curationSvc.PromoteToPattern(ctx, cand.ID, payload)
	if err != nil {
		t.Fatalf("PromoteToPattern: %v", err)
	}
	if obj == nil || obj.Kind != api.MemoryKindPattern {
		t.Errorf("created memory = %+v, want kind pattern", obj)
	}
	if obj.Statement != "Always run tests before deploy." {
		t.Errorf("memory.Statement = %q", obj.Statement)
	}

	cand2, err := curationRepo.GetByID(ctx, cand.ID)
	if err != nil {
		t.Fatalf("get candidate: %v", err)
	}
	if cand2 == nil || cand2.PromotionStatus != "promoted" {
		t.Errorf("candidate after promote = %+v, want status promoted", cand2)
	}

	mem, err := memoryRepo.GetByID(ctx, obj.ID)
	if err != nil {
		t.Fatalf("get memory: %v", err)
	}
	if mem == nil {
		t.Fatal("memory not found after promote")
	}
	if len(mem.Payload) == 0 {
		t.Error("memory payload empty")
	}
	var pl memory.PatternPayload
	if json.Unmarshal(mem.Payload, &pl) != nil {
		t.Fatal("memory payload not valid PatternPayload")
	}
	if pl.Polarity != "negative" || pl.Impact.Severity != "high" {
		t.Errorf("payload = %+v", pl)
	}
}

// Integration test: real HTTP handlers (/healthz, /readyz, GET /v1/recall/) against TEST_PG_DSN.
func TestIntegration_restHealthReadyAndRecall(t *testing.T) {
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

	resp, err := http.Get(srv.URL + "/healthz")
	if err != nil {
		t.Fatalf("GET /healthz: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/healthz status=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp, err = http.Get(srv.URL + "/readyz")
	if err != nil {
		t.Fatalf("GET /readyz: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("/readyz status=%d", resp.StatusCode)
	}
	_ = resp.Body.Close()

	resp, err = http.Get(srv.URL + "/v1/recall/?retrieval_query=integration%20smoke&max_per_kind=2")
	if err != nil {
		t.Fatalf("GET /v1/recall/: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		t.Fatalf("GET /v1/recall/ status=%d body=%s", resp.StatusCode, string(body))
	}
	_ = resp.Body.Close()
}

// TestIntegration_memories_createSearch exercises POST /v1/memories and POST /v1/memories/search.
// Requires TEST_PG_DSN.
func TestIntegration_memories_createSearch(t *testing.T) {
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

	createBody := `{"kind":"constraint","statement":"Never commit secrets","authority":7,"tags":["rules:security"]}`
	resp, err := http.Post(srv.URL+"/v1/memories", "application/json", strings.NewReader(createBody))
	if err != nil {
		t.Fatalf("POST /v1/memories: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /v1/memories status=%d body=%s", resp.StatusCode, string(b))
	}
	var created memory.MemoryObject
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if created.ID == uuid.Nil {
		t.Fatal("expected id")
	}
	var hasRule bool
	for _, tg := range created.Tags {
		if tg == "rules:security" {
			hasRule = true
			break
		}
	}
	if !hasRule {
		t.Fatalf("expected rules:security in tags, got %v", created.Tags)
	}

	searchBody := `{"query":"secrets","tags":["rules:security"]}`
	resp, err = http.Post(srv.URL+"/v1/memories/search", "application/json", strings.NewReader(searchBody))
	if err != nil {
		t.Fatalf("POST /v1/memories/search: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("POST /v1/memories/search status=%d body=%s", resp.StatusCode, string(b))
	}
	var list []memory.MemoryObject
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("want 1 result, got %d", len(list))
	}
	if list[0].ID != created.ID {
		t.Fatalf("id mismatch")
	}
}

// TestIntegration_enforcementEvaluate_postgresVsSqlite exercises POST /v1/enforcement/evaluate against real DB rows
// (binding constraint memory → block on SQLite proposal; unrelated proposal → allow).
// Requires TEST_PG_DSN and migrations applied (same as other integration tests).
func TestIntegration_enforcementEvaluate_postgresVsSqlite(t *testing.T) {
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

	ctx := context.Background()
	memoryRepo := &memory.Repo{DB: container.DB}
	memorySvc := &memory.Service{Repo: memoryRepo}

	_, err = memorySvc.Create(ctx, memory.CreateRequest{
		Kind:      api.MemoryKindConstraint,
		Authority: 9,
		Statement: "All durable project data must use Postgres; SQLite is not permitted.",
	})
	if err != nil {
		t.Fatalf("create constraint memory: %v", err)
	}

	enforcementSvc := &enforcement.Service{
		Repo:     memoryRepo,
		Evidence: nil,
		Config:   &cfg.Enforcement,
	}
	enforcementHandlers := &enforcement.Handlers{Service: enforcementSvc}

	r := httpx.NewRouter()
	r.Route("/v1", func(rr chi.Router) {
		rr.Route("/enforcement", func(rrr chi.Router) {
			rrr.Post("/evaluate", enforcementHandlers.Evaluate)
		})
	})
	srv := httptest.NewServer(r)
	defer srv.Close()

	t.Run("block_sqlite_proposal", func(t *testing.T) {
		body := `{"proposal_text":"We will migrate durable storage to SQLite.","intent":"datastore"}`
		resp, err := http.Post(srv.URL+"/v1/enforcement/evaluate", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST enforcement: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status=%d body=%s", resp.StatusCode, b)
		}
		var out enforcement.EvaluateResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Decision != enforcement.DecisionBlock {
			t.Fatalf("decision=%q want %q", out.Decision, enforcement.DecisionBlock)
		}
		if len(out.TriggeredMemories) < 1 {
			t.Fatalf("expected triggered_memories, got %+v", out)
		}
		if out.TriggeredMemories[0].ReasonCode != "normative_conflict" {
			t.Fatalf("reason_code=%q want normative_conflict", out.TriggeredMemories[0].ReasonCode)
		}
		if out.EvaluationEngine != enforcement.EvaluationEngineRuleBasedHeuristicV1 {
			t.Fatalf("evaluation_engine=%q", out.EvaluationEngine)
		}
	})

	t.Run("allow_unrelated", func(t *testing.T) {
		body := `{"proposal_text":"We will add a metrics dashboard.","intent":"change"}`
		resp, err := http.Post(srv.URL+"/v1/enforcement/evaluate", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST enforcement: %v", err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			b, _ := io.ReadAll(resp.Body)
			t.Fatalf("status=%d body=%s", resp.StatusCode, b)
		}
		var out enforcement.EvaluateResponse
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if out.Decision != enforcement.DecisionAllow {
			t.Fatalf("decision=%q want %q", out.Decision, enforcement.DecisionAllow)
		}
		if out.EvaluationEngine != enforcement.EvaluationEngineRuleBasedHeuristicV1 {
			t.Fatalf("evaluation_engine=%q", out.EvaluationEngine)
		}
		if out.EvaluationNote == "" {
			t.Fatal("expected non-empty evaluation_note")
		}
	})
}

