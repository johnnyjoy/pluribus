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

	"github.com/google/uuid"
)

func bootAdvisoryAutoDistill(t *testing.T, dsn string, auto bool) (*httptest.Server, func()) {
	t.Helper()
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = true
	cfg.Similarity.MinResemblance = 0.05
	cfg.Distillation.Enabled = true
	cfg.Distillation.AutoFromAdvisoryEpisodes = auto
	cfg.Enforcement.MinBindingAuthority = 3

	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		_ = container.DB.Close()
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(rtr)
	return srv, func() {
		srv.Close()
		container.DB.Close()
	}
}

func pendingBodies(t *testing.T, base string) string {
	t.Helper()
	resp, err := http.Get(base + "/v1/curation/pending")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("pending: %d %s", resp.StatusCode, string(b))
	}
	return string(b)
}

// TestREST_advisory_autoDistill_createsCandidate proves POST advisory with auto_from_advisory_episodes creates candidates without explicit distill.
func TestREST_advisory_autoDistill_createsCandidate(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootAdvisoryAutoDistill(t, dsn, true)
	defer cleanup()

	tag := "rest:auto-dist-on:" + uuid.New().String()[:8]
	summary := fmt.Sprintf(`AUTO DIST ON %s payment gateway timeout error caused duplicate charge incident during checkout peak load`, tag)
	body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, summary, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("advisory create: %d %s", resp.StatusCode, string(b))
	}

	pend := pendingBodies(t, srv.URL)
	if !strings.Contains(pend, tag) {
		t.Fatalf("expected pending candidate after auto distill; body=%s", pend)
	}
	if !strings.Contains(pend, `"pluribus_distill_origin":"auto"`) {
		t.Fatalf("expected proposal_json to mark auto origin; snippet=%s", pend[:min(1200, len(pend))])
	}
}

// TestREST_advisory_autoDistill_disabled_noImplicitCandidate explicit distill still required when auto is off.
func TestREST_advisory_autoDistill_disabled_noImplicitCandidate(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootAdvisoryAutoDistill(t, dsn, false)
	defer cleanup()

	tag := "rest:auto-dist-off:" + uuid.New().String()[:8]
	summary := fmt.Sprintf(`AUTO DIST OFF %s inventory sync failure error lost warehouse stock counts during overnight batch`, tag)
	body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, summary, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("advisory: %d %s", resp.StatusCode, string(rb))
	}
	var ep struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(rb, &ep)

	pend := pendingBodies(t, srv.URL)
	if strings.Contains(pend, tag) {
		t.Fatalf("did not expect pending row for tag before explicit distill; body=%s", pend)
	}

	dr, err := http.Post(srv.URL+"/v1/episodes/distill", "application/json",
		strings.NewReader(fmt.Sprintf(`{"episode_id":%q}`, ep.ID)))
	if err != nil {
		t.Fatal(err)
	}
	db, _ := io.ReadAll(dr.Body)
	_ = dr.Body.Close()
	if dr.StatusCode != http.StatusOK {
		t.Fatalf("distill: %d %s", dr.StatusCode, string(db))
	}
	pend2 := pendingBodies(t, srv.URL)
	if !strings.Contains(pend2, tag) {
		t.Fatalf("expected candidate after explicit distill; body=%s", pend2)
	}
}

// TestREST_advisory_autoDistill_noSignal short summary produces no candidate.
func TestREST_advisory_autoDistill_noSignal(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootAdvisoryAutoDistill(t, dsn, true)
	defer cleanup()

	tag := "rest:auto-no-sig:" + uuid.New().String()[:8]
	body := fmt.Sprintf(`{"summary":"short","source":"manual","tags":[%q]}`, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("advisory: %d %s", resp.StatusCode, string(rb))
	}
	pend := pendingBodies(t, srv.URL)
	if strings.Contains(pend, tag) {
		t.Fatalf("weak episode should not create distilled candidate; body=%s", pend)
	}
}

// TestREST_advisory_autoDistill_mergeRepeatedEpisodes two similar ingests strengthen one pending row.
func TestREST_advisory_autoDistill_mergeRepeatedEpisodes(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootAdvisoryAutoDistill(t, dsn, true)
	defer cleanup()

	tag := "rest:auto-merge:" + uuid.New().String()[:8]
	shared := fmt.Sprintf(`AUTO MERGE %s database transaction rollback error after duplicate key rejection on orders table during flash sale`, tag)
	for i := 0; i < 2; i++ {
		body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q,"iter-%d"]}`, shared, tag, i)
		resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		rb, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("iter %d: %d %s", i, resp.StatusCode, string(rb))
		}
	}
	pend := pendingBodies(t, srv.URL)
	if !strings.Contains(pend, `"distill_support_count":2`) {
		t.Fatalf("expected merged support count 2; body=%s", pend)
	}
}

// TestREST_advisory_autoDistill_recallIgnoresUntilMaterialize advisory + auto candidate does not appear in recall until materialized.
func TestREST_advisory_autoDistill_recallIgnoresUntilMaterialize(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootAdvisoryAutoDistill(t, dsn, true)
	defer cleanup()

	rid := uuid.New().String()[:8]
	tag := "rest:auto-rec:" + rid
	marker := fmt.Sprintf(`AUTO REC ISOLATION %s never deploy schema migrations without backup error caused outage pattern`, tag)
	body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, marker, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("advisory: %d %s", resp.StatusCode, string(rb))
	}

	rc := postJSONString(t, srv.URL+"/v1/recall/compile",
		fmt.Sprintf(`{"retrieval_query":%q,"tags":[%q],"max_per_kind":8,"max_total":40}`, marker, tag))
	if strings.Contains(rc, "AUTO REC ISOLATION") {
		t.Fatalf("recall must not surface advisory/candidate text before materialize; got %s", rc[:800])
	}

	pend := pendingBodies(t, srv.URL)
	if !strings.Contains(pend, "AUTO REC ISOLATION") {
		t.Fatalf("pending should list auto-distilled candidate: %s", pend)
	}
}

func postJSONString(t *testing.T, url, jsonBody string) string {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(jsonBody))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("POST %s: %d %s", url, resp.StatusCode, string(b))
	}
	return string(b)
}
