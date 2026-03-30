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

	"github.com/google/uuid"
)

// TestREST_candidateReview_fields verifies GET /v1/curation/candidates/{id}/review returns explanation, signal, capped episodes, preview.
func TestREST_candidateReview_fields(t *testing.T) {
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

	tag := "rest:review-" + uuid.New().String()[:8]
	body := fmt.Sprintf(`{"summary":"failure error timeout when connecting to redis; retry caused duplicate charge","source":"manual","tags":[%q,"entity:redis"]}`, tag)
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
			CandidateID string `json:"candidate_id"`
			Kind        string `json:"kind"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(db, &dist); err != nil {
		t.Fatal(err)
	}
	var failID string
	for _, c := range dist.Candidates {
		if c.Kind == "failure" {
			failID = c.CandidateID
			break
		}
	}
	if failID == "" {
		t.Fatal("want a failure candidate from distill")
	}

	rev, err := http.Get(srv.URL + "/v1/curation/candidates/" + failID + "/review")
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(rev.Body)
	_ = rev.Body.Close()
	if rev.StatusCode != http.StatusOK {
		t.Fatalf("review: %d %s", rev.StatusCode, string(rb))
	}
	var review struct {
		Explanation        string `json:"explanation"`
		SignalStrength     string `json:"signal_strength"`
		SignalDetail       string `json:"signal_detail"`
		SupportingEpisodes []struct {
			EpisodeID string `json:"episode_id"`
			Summary   string `json:"summary"`
		} `json:"supporting_episodes"`
		PromotionPreview *struct {
			Kind          string `json:"kind"`
			Statement     string `json:"statement"`
			Applicability string `json:"applicability"`
		} `json:"promotion_preview"`
		TagsGrouped struct {
			Entities []string `json:"entities"`
		} `json:"tags_grouped"`
	}
	if err := json.Unmarshal(rb, &review); err != nil {
		t.Fatal(err)
	}
	if review.Explanation == "" || !strings.Contains(review.Explanation, "failure") {
		t.Fatalf("explanation: %q", review.Explanation)
	}
	if review.SignalStrength == "" || review.SignalDetail == "" {
		t.Fatalf("signal: %s %s", review.SignalStrength, review.SignalDetail)
	}
	if len(review.SupportingEpisodes) < 1 {
		t.Fatal("want at least one supporting episode summary")
	}
	if len(review.SupportingEpisodes) > 3 {
		t.Fatalf("supporting episodes capped at 3, got %d", len(review.SupportingEpisodes))
	}
	if review.PromotionPreview == nil || review.PromotionPreview.Kind != "failure" {
		t.Fatalf("promotion preview: %+v", review.PromotionPreview)
	}
	if len(review.TagsGrouped.Entities) == 0 {
		t.Fatal("want entity tag surfaced in tags_grouped.entities")
	}
}

// TestREST_candidateReview_isolation_noMemoryWrite ensures GET review does not insert memories and does not affect recall/enforcement.
func TestREST_candidateReview_isolation_noMemoryWrite(t *testing.T) {
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

	unique := "REVIEW UNIQUE " + uuid.New().String()
	tag := "rest:review-iso-" + uuid.New().String()[:8]
	body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, unique+" error rollback duplicate", tag)
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
	var dist struct {
		Candidates []struct {
			CandidateID string `json:"candidate_id"`
		} `json:"candidates"`
	}
	_ = json.Unmarshal(db, &dist)
	if len(dist.Candidates) == 0 {
		t.Fatal("want candidates")
	}
	cid := dist.Candidates[0].CandidateID

	rev1, err := http.Get(srv.URL + "/v1/curation/candidates/" + cid + "/review")
	if err != nil {
		t.Fatal(err)
	}
	r1, _ := io.ReadAll(rev1.Body)
	_ = rev1.Body.Close()
	rev2, err := http.Get(srv.URL + "/v1/curation/candidates/" + cid + "/review")
	if err != nil {
		t.Fatal(err)
	}
	r2, _ := io.ReadAll(rev2.Body)
	_ = rev2.Body.Close()
	if rev1.StatusCode != http.StatusOK || rev2.StatusCode != http.StatusOK {
		t.Fatalf("review: %d %d", rev1.StatusCode, rev2.StatusCode)
	}
	if string(r1) != string(r2) {
		t.Fatal("review should be idempotent for same candidate")
	}

	var memAfter int
	if err := container.DB.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&memAfter); err != nil {
		t.Fatalf("count memories: %v", err)
	}
	if memBefore != memAfter {
		t.Fatalf("GET review must not write memories: before %d after %d", memBefore, memAfter)
	}

	compileBody := fmt.Sprintf(`{"retrieval_query":"rollback error distilled review","tags":[%q],"max_per_kind":8,"max_total":40}`, tag)
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
		t.Fatal("recall bundle must not contain distilled-only advisory text")
	}

	enfBody := fmt.Sprintf(`{"proposal_text":%q,"tags":[%q]}`, unique+" rollback violation", tag)
	ereq, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/enforcement/evaluate", strings.NewReader(enfBody))
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
		t.Fatalf("candidates must not surface as enforcement bindings: %+v", eval.TriggeredMemories)
	}
}
