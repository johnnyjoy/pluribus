//go:build integration
// +build integration

package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/memory"
	"control-plane/internal/recall"
	"control-plane/pkg/api"

	"github.com/google/uuid"
)

func TestTTLArchiveHistoricalRecallIntegration(t *testing.T) {
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

	ctx := context.Background()
	memRepo := &memory.Repo{DB: container.DB}
	memSvc := &memory.Service{Repo: memRepo, Lifecycle: &memory.LifecycleConfig{ExpirationAuthorityThreshold: 2}}

	tag := "p9b-ttl-expire-" + uuid.New().String()[:8]
	statement := "P9B TTL expire archive integration proof " + uuid.New().String()
	ttl := 60
	obj, err := memSvc.Create(ctx, memory.CreateRequest{
		Kind:          api.MemoryKindState,
		Authority:     1,
		Statement:     statement,
		Tags:          []string{tag},
		TTLSeconds:    ttl,
		Applicability: api.ApplicabilityAdvisory,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	past := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := container.DB.ExecContext(ctx,
		`UPDATE memories SET created_at = $1 WHERE id = $2`, past, obj.ID); err != nil {
		t.Fatalf("backdate created_at: %v", err)
	}
	var ttlDB sql.NullInt64
	var createdAt time.Time
	if err := container.DB.QueryRowContext(ctx,
		`SELECT ttl_seconds, created_at FROM memories WHERE id = $1`, obj.ID).Scan(&ttlDB, &createdAt); err != nil {
		t.Fatalf("verify row: %v", err)
	}
	if !ttlDB.Valid || ttlDB.Int64 != int64(ttl) {
		t.Fatalf("ttl_seconds in DB=%v want %d", ttlDB, ttl)
	}
	if time.Since(createdAt) < time.Hour {
		t.Fatalf("created_at backdate failed: created_at=%v (expected ~2h ago)", createdAt)
	}

	authBefore := obj.Authority
	asOf := time.Now().UTC()
	n, err := memSvc.ExpireMemories(ctx, asOf)
	if err != nil {
		t.Fatalf("ExpireMemories: %v", err)
	}
	if n != 1 {
		t.Fatalf("ExpireMemories archived=%d want 1", n)
	}
	got, err := memRepo.GetByID(ctx, obj.ID)
	if err != nil || got == nil {
		t.Fatalf("GetByID after archive: %v", err)
	}
	if got.Status != api.StatusArchived {
		t.Fatalf("status=%q want archived", got.Status)
	}
	if got.Authority != authBefore {
		t.Fatalf("authority mutated: %d -> %d", authBefore, got.Authority)
	}
	if got.Statement != statement {
		t.Fatalf("statement changed")
	}

	currentBody, _ := json.Marshal(map[string]any{
		"retrieval_query": statement,
		"tags":            []string{tag},
		"recall_mode":     "current",
		"max_per_kind":    10,
	})
	currentBundle := postRecallCompile(t, srv.URL, currentBody)
	if recallBundleHasMemoryID(currentBundle, obj.ID) {
		t.Fatal("TTL-archived memory must be excluded from current recall")
	}

	histBody, _ := json.Marshal(map[string]any{
		"retrieval_query": statement,
		"tags":            []string{tag},
		"recall_mode":     "historical",
		"max_per_kind":    10,
	})
	histBundle := postRecallCompile(t, srv.URL, histBody)
	if !recallBundleHasMemoryID(histBundle, obj.ID) {
		t.Fatal("TTL-archived memory must appear in historical recall")
	}
	if role := findLifecycleRole(histBundle, obj.ID); role != "archived_context" {
		t.Fatalf("lifecycle_role=%q want archived_context", role)
	}
}

func TestTTLDoesNotArchiveHistoricallyValuableMemory_integration(t *testing.T) {
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

	ctx := context.Background()
	memRepo := &memory.Repo{DB: container.DB}
	memSvc := &memory.Service{Repo: memRepo, Lifecycle: &memory.LifecycleConfig{ExpirationAuthorityThreshold: 2}}
	tag := "phase9b-ttl-blocked-" + uuid.New().String()[:8]
	statement := "Phase9B TTL must not archive doctrine-tagged memory " + uuid.New().String()
	ttl := 60
	obj, err := memSvc.Create(ctx, memory.CreateRequest{
		Kind:          api.MemoryKindDecision,
		Authority:     1,
		Statement:     statement,
		Tags:          []string{tag, "doctrine"},
		TTLSeconds:    ttl,
		Applicability: api.ApplicabilityAdvisory,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if _, err := container.DB.ExecContext(ctx,
		`UPDATE memories SET created_at = $1 WHERE id = $2`, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), obj.ID); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	n, err := memSvc.ExpireMemories(ctx, time.Now().UTC())
	if err != nil {
		t.Fatalf("ExpireMemories: %v", err)
	}
	if n != 0 {
		t.Fatalf("ExpireMemories archived=%d want 0 for doctrine-tagged memory", n)
	}
	got, _ := memRepo.GetByID(ctx, obj.ID)
	if got != nil && got.Status != api.StatusActive {
		t.Fatalf("status=%q want active (historical-value tag blocked TTL archive)", got.Status)
	}
}

func TestAdvisoryPruneDoesNotDeleteCanonicalMemory_integration(t *testing.T) {
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

	ctx := context.Background()
	memRepo := &memory.Repo{DB: container.DB}
	memSvc := &memory.Service{Repo: memRepo}
	statement := "Phase9B canonical memory survives advisory prune " + uuid.New().String()
	obj, err := memSvc.Create(ctx, memory.CreateRequest{
		Kind:      api.MemoryKindPattern,
		Authority: 3,
		Statement: statement,
		Tags:      []string{"phase9b-advisory-boundary"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	srv := httptest.NewServer(rtr)
	defer srv.Close()

	pruneBody := bytes.NewBufferString(`{"older_than_hours":0,"limit":1000}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/advisory-episodes/prune-rejected", pruneBody)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("prune request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode >= 400 {
		t.Fatalf("prune HTTP %d", resp.StatusCode)
	}

	got, err := memRepo.GetByID(ctx, obj.ID)
	if err != nil || got == nil {
		t.Fatalf("canonical memory missing after advisory prune: %v", err)
	}
}

func TestRESTHistoricalRecallRejectsInvalidDate_integration(t *testing.T) {
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

	body := bytes.NewBufferString(`{"retrieval_query":"history","recall_mode":"historical","occurred_after":"not-a-date"}`)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/recall/compile", body)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("HTTP %d want 400 for invalid occurred_after", resp.StatusCode)
	}
}

func postRecallCompile(t *testing.T, base string, body []byte) *recall.RecallBundle {
	t.Helper()
	resp, err := http.Post(base+"/v1/recall/compile", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST compile: %v", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("compile HTTP %d: %s", resp.StatusCode, string(raw))
	}
	var bundle recall.RecallBundle
	if err := json.Unmarshal(raw, &bundle); err != nil {
		t.Fatalf("decode bundle: %v", err)
	}
	return &bundle
}

func recallBundleHasMemoryID(bundle *recall.RecallBundle, id uuid.UUID) bool {
	if bundle == nil {
		return false
	}
	idStr := id.String()
	for _, items := range [][]recall.MemoryItem{
		bundle.GoverningConstraints, bundle.Decisions, bundle.KnownFailures,
		bundle.ApplicablePatterns, bundle.Constraints, bundle.Continuity, bundle.Experience,
	} {
		for _, it := range items {
			if it.ID == idStr {
				return true
			}
		}
	}
	return false
}

func findLifecycleRole(bundle *recall.RecallBundle, id uuid.UUID) string {
	idStr := id.String()
	for _, items := range [][]recall.MemoryItem{
		bundle.GoverningConstraints, bundle.Decisions, bundle.KnownFailures,
		bundle.ApplicablePatterns, bundle.Constraints, bundle.Continuity, bundle.Experience,
	} {
		for _, it := range items {
			if it.ID == idStr {
				return it.LifecycleRole
			}
		}
	}
	return ""
}
