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
	"control-plane/internal/enforcement"

	"github.com/google/uuid"
)

// TestREST_episodeDistill_producesCandidates proves POST /v1/episodes/distill creates candidate rows, not memories.
func TestREST_episodeDistill_producesCandidates(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = true
	cfg.Distillation.Enabled = true

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

	tag := "rest:distill-" + uuid.New().String()[:8]
	body := fmt.Sprintf(`{"summary":"rollback after timeout error; we must not skip health checks","source":"manual","tags":[%q]}`, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("advisory create: %d %s", resp.StatusCode, string(b))
	}
	var ep struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &ep); err != nil {
		t.Fatal(err)
	}

	distBody := fmt.Sprintf(`{"episode_id":%q}`, ep.ID)
	dresp, err := http.Post(srv.URL+"/v1/episodes/distill", "application/json", strings.NewReader(distBody))
	if err != nil {
		t.Fatal(err)
	}
	db, _ := io.ReadAll(dresp.Body)
	_ = dresp.Body.Close()
	if dresp.StatusCode != http.StatusOK {
		t.Fatalf("distill: %d %s", dresp.StatusCode, string(db))
	}
	var dist struct {
		Candidates []struct {
			CandidateID             string `json:"candidate_id"`
			Kind                    string `json:"kind"`
			SourceAdvisoryEpisodeID string `json:"source_advisory_episode_id"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(db, &dist); err != nil {
		t.Fatal(err)
	}
	if len(dist.Candidates) < 2 {
		t.Fatalf("want at least failure+constraint candidates, got %d", len(dist.Candidates))
	}
	for _, c := range dist.Candidates {
		if c.SourceAdvisoryEpisodeID != ep.ID {
			t.Fatalf("traceability: want source %s got %s", ep.ID, c.SourceAdvisoryEpisodeID)
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
	if !strings.Contains(string(pb), dist.Candidates[0].CandidateID) {
		t.Fatalf("pending list should include distilled candidate id")
	}
}

// TestREST_episodeDistill_doesNotAffectRecallOrEnforcement ensures candidates stay out of recall bundles and enforcement.
func TestREST_episodeDistill_doesNotAffectRecallOrEnforcement(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = true
	cfg.Distillation.Enabled = true

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

	unique := "DISTILL UNIQUE PHRASE " + uuid.New().String()
	tag := "rest:distill-enf-" + uuid.New().String()[:8]
	body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, unique+" error rollback", tag)
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

	distBody := fmt.Sprintf(`{"episode_id":%q}`, ep.ID)
	dresp, err := http.Post(srv.URL+"/v1/episodes/distill", "application/json", strings.NewReader(distBody))
	if err != nil {
		t.Fatal(err)
	}
	db, _ := io.ReadAll(dresp.Body)
	_ = dresp.Body.Close()
	if dresp.StatusCode != http.StatusOK {
		t.Fatalf("distill: %d %s", dresp.StatusCode, string(db))
	}

	compileBody := fmt.Sprintf(`{"retrieval_query":"rollback error distilled","tags":[%q],"max_per_kind":8,"max_total":40}`, tag)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/recall/compile", strings.NewReader(compileBody))
	req.Header.Set("Content-Type", "application/json")
	cre, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := io.ReadAll(cre.Body)
	_ = cre.Body.Close()
	if cre.StatusCode != http.StatusOK {
		t.Fatalf("compile: %d %s", cre.StatusCode, string(cb))
	}
	if strings.Contains(string(cb), unique) {
		t.Fatal("recall bundle must not contain distilled-only text from advisory")
	}

	enfBody := fmt.Sprintf(`{"proposal_text":%q,"tags":[%q]}`, unique+" rollback violation", tag)
	ereq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/enforcement/evaluate", bytes.NewReader([]byte(enfBody)))
	ereq.Header.Set("Content-Type", "application/json")
	eres, err := http.DefaultClient.Do(ereq)
	if err != nil {
		t.Fatal(err)
	}
	eb, _ := io.ReadAll(eres.Body)
	_ = eres.Body.Close()
	if eres.StatusCode != http.StatusOK {
		t.Fatalf("enforcement: %d %s", eres.StatusCode, string(eb))
	}
	var eval enforcement.EvaluateResponse
	if err := json.Unmarshal(eb, &eval); err != nil {
		t.Fatal(err)
	}
	if len(eval.TriggeredMemories) > 0 {
		t.Fatalf("distilled candidates must not surface as binding memories: %+v", eval.TriggeredMemories)
	}
}

// TestREST_episodeDistill_consolidatesDuplicates merges two episodes with the same summary into one failure candidate (higher support, same id).
func TestREST_episodeDistill_consolidatesDuplicates(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = true
	cfg.Distillation.Enabled = true

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

	tag := "rest:distill-merge-" + uuid.New().String()[:8]
	shared := "failure error timeout when connecting to redis"
	body1 := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, shared, tag)
	resp1, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body1))
	if err != nil {
		t.Fatal(err)
	}
	b1, _ := io.ReadAll(resp1.Body)
	_ = resp1.Body.Close()
	if resp1.StatusCode != http.StatusCreated {
		t.Fatalf("advisory create: %d %s", resp1.StatusCode, string(b1))
	}
	var ep1 struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b1, &ep1)

	body2 := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, shared, tag)
	resp2, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body2))
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("advisory create: %d %s", resp2.StatusCode, string(b2))
	}
	var ep2 struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b2, &ep2)

	dist1 := fmt.Sprintf(`{"episode_id":%q}`, ep1.ID)
	r1, err := http.Post(srv.URL+"/v1/episodes/distill", "application/json", strings.NewReader(dist1))
	if err != nil {
		t.Fatal(err)
	}
	out1, _ := io.ReadAll(r1.Body)
	_ = r1.Body.Close()
	if r1.StatusCode != http.StatusOK {
		t.Fatalf("distill1: %d %s", r1.StatusCode, string(out1))
	}
	var d1 struct {
		Candidates []struct {
			CandidateID         string `json:"candidate_id"`
			Kind                string `json:"kind"`
			Merged              bool   `json:"merged"`
			DistillSupportCount int    `json:"distill_support_count"`
			SourceIDs           []string `json:"source_advisory_episode_ids"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(out1, &d1); err != nil {
		t.Fatal(err)
	}
	var failID string
	for _, c := range d1.Candidates {
		if c.Kind == "failure" {
			failID = c.CandidateID
			if c.Merged {
				t.Fatalf("first distill should not be merged")
			}
			if c.DistillSupportCount != 1 {
				t.Fatalf("want support 1, got %d", c.DistillSupportCount)
			}
			break
		}
	}
	if failID == "" {
		t.Fatalf("no failure candidate: %s", string(out1))
	}

	dist2 := fmt.Sprintf(`{"episode_id":%q}`, ep2.ID)
	r2, err := http.Post(srv.URL+"/v1/episodes/distill", "application/json", strings.NewReader(dist2))
	if err != nil {
		t.Fatal(err)
	}
	out2, _ := io.ReadAll(r2.Body)
	_ = r2.Body.Close()
	if r2.StatusCode != http.StatusOK {
		t.Fatalf("distill2: %d %s", r2.StatusCode, string(out2))
	}
	var d2 struct {
		Candidates []struct {
			CandidateID         string   `json:"candidate_id"`
			Kind                string   `json:"kind"`
			Merged              bool     `json:"merged"`
			DistillSupportCount int      `json:"distill_support_count"`
			SourceIDs           []string `json:"source_advisory_episode_ids"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(out2, &d2); err != nil {
		t.Fatal(err)
	}
	var fail2 struct {
		CandidateID         string
		Merged              bool
		DistillSupportCount int
		SourceIDs           []string
	}
	for _, c := range d2.Candidates {
		if c.Kind == "failure" {
			fail2.CandidateID = c.CandidateID
			fail2.Merged = c.Merged
			fail2.DistillSupportCount = c.DistillSupportCount
			fail2.SourceIDs = c.SourceIDs
			break
		}
	}
	if fail2.CandidateID != failID {
		t.Fatalf("expected same candidate id after merge, got %s vs %s", failID, fail2.CandidateID)
	}
	if !fail2.Merged {
		t.Fatal("second distill should set merged=true")
	}
	if fail2.DistillSupportCount != 2 {
		t.Fatalf("want support count 2, got %d", fail2.DistillSupportCount)
	}
	if len(fail2.SourceIDs) != 2 {
		t.Fatalf("want two source episode ids, got %v", fail2.SourceIDs)
	}
}
