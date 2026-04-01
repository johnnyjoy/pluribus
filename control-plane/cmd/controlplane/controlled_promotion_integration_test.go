//go:build integration
// +build integration

package main

import (
	"bytes"
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

	"github.com/google/uuid"
)

// TestREST_controlledPromotion_autoPromote creates a high-support distilled candidate and runs auto-promote when enabled.
func TestREST_controlledPromotion_autoPromote(t *testing.T) {
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
	cfg.Distillation.Enabled = true
	cfg.Promotion.AutoPromote = true
	cfg.Promotion.AutoMinSupportCount = 2
	cfg.Promotion.AutoMinSalience = 0.5
	cfg.Promotion.AutoAllowedKinds = []string{"failure", "pattern"}

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

	tag := "rest:ctrl-promo-" + uuid.New().String()[:8]
	shared := "failure error timeout when connecting to redis duplicate charge"
	for i := 0; i < 2; i++ {
		body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, shared, tag)
		resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("advisory: %d %s", resp.StatusCode, string(b))
		}
		var ep struct {
			ID string `json:"id"`
		}
		_ = json.Unmarshal(b, &ep)
		dresp, err := http.Post(srv.URL+"/v1/episodes/distill", "application/json",
			strings.NewReader(fmt.Sprintf(`{"episode_id":%q}`, ep.ID)))
		if err != nil {
			t.Fatal(err)
		}
		db, _ := io.ReadAll(dresp.Body)
		_ = dresp.Body.Close()
		if dresp.StatusCode != http.StatusOK {
			t.Fatalf("distill: %d %s", dresp.StatusCode, string(db))
		}
	}

	pend, err := http.Get(srv.URL + "/v1/curation/pending")
	if err != nil {
		t.Fatal(err)
	}
	pb, _ := io.ReadAll(pend.Body)
	_ = pend.Body.Close()
	if pend.StatusCode != http.StatusOK {
		t.Fatalf("pending: %d %s", pend.StatusCode, string(pb))
	}
	var plist []struct {
		ID                 string `json:"id"`
		PromotionReadiness string `json:"promotion_readiness"`
	}
	if err := json.Unmarshal(pb, &plist); err != nil {
		t.Fatal(err)
	}
	if len(plist) == 0 {
		t.Fatal("want pending candidates")
	}
	found := false
	for _, c := range plist {
		if strings.Contains(c.PromotionReadiness, "high") || strings.Contains(c.PromotionReadiness, "review") || c.PromotionReadiness == "not_ready" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected promotion_readiness on list: %s", string(pb))
	}

	areq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/curation/auto-promote", bytes.NewReader([]byte("{}")))
	areq.Header.Set("Content-Type", "application/json")
	ares, err := http.DefaultClient.Do(areq)
	if err != nil {
		t.Fatal(err)
	}
	ab, _ := io.ReadAll(ares.Body)
	_ = ares.Body.Close()
	if ares.StatusCode != http.StatusOK {
		t.Fatalf("auto-promote: %d %s", ares.StatusCode, string(ab))
	}
	var auto struct {
		Results []struct {
			Status      string `json:"status"`
			MemoryID    string `json:"memory_id"`
			CandidateID string `json:"candidate_id"`
			Detail      string `json:"detail"`
		} `json:"results"`
	}
	if err := json.Unmarshal(ab, &auto); err != nil {
		t.Fatal(err)
	}
	promoted := 0
	for _, r := range auto.Results {
		if r.Status == "promoted" {
			promoted++
			if r.MemoryID == "" {
				t.Fatal("promoted row missing memory_id")
			}
		}
	}
	if promoted == 0 {
		t.Fatalf("expected at least one promoted result, got %s", string(ab))
	}
}

// TestREST_controlledPromotion_autoPromoteDisabled returns 403 when auto_promote is false.
func TestREST_controlledPromotion_autoPromoteDisabled(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Promotion.AutoPromote = false

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

	areq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/curation/auto-promote", bytes.NewReader([]byte("{}")))
	areq.Header.Set("Content-Type", "application/json")
	ares, err := http.DefaultClient.Do(areq)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(ares.Body)
	_ = ares.Body.Close()
	if ares.StatusCode != http.StatusForbidden {
		t.Fatalf("want 403 when disabled, got %d %s", ares.StatusCode, string(b))
	}
}
