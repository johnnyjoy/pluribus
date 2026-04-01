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
)

// TestREST_advisory_ingest_immediate_memory asserts signal-rich ingest creates a probationary memory in one request (no follow-up).
func TestREST_advisory_ingest_immediate_memory(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = app.BoolPtr(true)
	cfg.Similarity.MinResemblance = 0.01
	cfg.Distillation.Enabled = false
	cfg.Distillation.AutoFromAdvisoryEpisodes = false

	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	defer container.DB.Close()

	var memBefore int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&memBefore); err != nil {
		t.Fatalf("count memories: %v", err)
	}

	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	body := `{"summary":"deployment failure timeout rollback decision during peak traffic incident learning fix","source":"manual"}`
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201 got %d body=%s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["memory_formation_status"] != "linked" {
		t.Fatalf("want linked, got %#v", out["memory_formation_status"])
	}
	if out["probationary_memory_id"] == nil && out["related_memory_id"] == nil {
		t.Fatalf("expected probationary_memory_id or related_memory_id: %#v", out)
	}

	var memAfter int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&memAfter); err != nil {
		t.Fatalf("count memories: %v", err)
	}
	if memAfter <= memBefore {
		t.Fatalf("expected new memory row after ingest, before=%d after=%d", memBefore, memAfter)
	}
}

// TestREST_advisory_ingest_rejected_no_memory asserts low-signal ingest does not create memory and marks advisory rejected.
func TestREST_advisory_ingest_rejected_no_memory(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = app.BoolPtr(true)
	cfg.Similarity.MinResemblance = 0.01
	cfg.Distillation.Enabled = false
	cfg.Distillation.AutoFromAdvisoryEpisodes = false

	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	defer container.DB.Close()

	var memBefore int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&memBefore); err != nil {
		t.Fatalf("count memories: %v", err)
	}

	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	// Long enough for min runes, no distillation keywords or mcp:event tags.
	body := fmt.Sprintf(`{"summary":%q,"source":"manual"}`, strings.Repeat("z", 24))
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("want 201 got %d body=%s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["memory_formation_status"] != "rejected" {
		t.Fatalf("want rejected, got %#v full=%#v", out["memory_formation_status"], out)
	}
	rr, _ := out["rejection_reason"].(string)
	if rr == "" {
		t.Fatalf("expected rejection_reason: %#v", out)
	}

	var memAfter int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&memAfter); err != nil {
		t.Fatalf("count memories: %v", err)
	}
	if memAfter != memBefore {
		t.Fatalf("low-signal ingest must not create memory: before=%d after=%d", memBefore, memAfter)
	}
}
