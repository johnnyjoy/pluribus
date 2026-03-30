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
	"time"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/enforcement"

	"github.com/google/uuid"
)

// TestREST_advisory_episodes_ingestOccurredAt proves POST /v1/advisory-episodes accepts occurred_at and entities (advisory only).
func TestREST_advisory_episodes_ingestOccurredAt(t *testing.T) {
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
	cfg.Similarity.MinResemblance = 0.01
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

	occ := time.Date(2020, 3, 10, 8, 30, 0, 0, time.UTC)
	body := fmt.Sprintf(`{"summary":"oatmeal breakfast at cafe","source":"manual","tags":["rest:episodic-it"],"occurred_at":%q,"entities":["self","breakfast","cafe"]}`,
		occ.Format(time.RFC3339Nano))
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create episode: want 201 got %d body=%s", resp.StatusCode, string(b))
	}
	var out map[string]interface{}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if out["occurred_at"] == nil {
		t.Fatalf("expected occurred_at in response: %#v", out)
	}
}

// TestREST_advisory_episodes_similarByDateRange filters by occurred_after / occurred_before on effective time.
func TestREST_advisory_episodes_similarByDateRange(t *testing.T) {
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
	cfg.Similarity.MinResemblance = 0.01
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

	tag := "rest:episodic-range-" + uuid.New().String()[:8]
	in2020 := time.Date(2020, 7, 1, 12, 0, 0, 0, time.UTC)
	in2024 := time.Date(2024, 1, 15, 9, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		summary string
		occ     time.Time
	}{
		{"year 2020 team lunch notes", in2020},
		{"year 2024 standup notes", in2024},
	} {
		body := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q],"occurred_at":%q}`,
			tc.summary, tag, tc.occ.Format(time.RFC3339Nano))
		resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create: %d %s", resp.StatusCode, string(b))
		}
	}

	simBody := fmt.Sprintf(`{
		"query": "standup lunch team notes",
		"tags": [%q],
		"occurred_after": %q,
		"occurred_before": %q,
		"max_results": 10
	}`, tag, time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339Nano),
		time.Date(2020, 12, 31, 23, 59, 59, 0, time.UTC).Format(time.RFC3339Nano))
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes/similar", "application/json", strings.NewReader(simBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("similar: %d %s", resp.StatusCode, string(b))
	}
	var sim struct {
		Cases []struct {
			Summary string `json:"summary"`
		} `json:"advisory_similar_cases"`
	}
	if err := json.Unmarshal(b, &sim); err != nil {
		t.Fatal(err)
	}
	if len(sim.Cases) != 1 {
		t.Fatalf("want 1 case in 2020 window, got %d", len(sim.Cases))
	}
	if !strings.Contains(sim.Cases[0].Summary, "2020") {
		t.Fatalf("expected 2020 episode, got %q", sim.Cases[0].Summary)
	}
}

// TestREST_advisory_episodes_similarByEntity filters by entity overlap (not a partition).
func TestREST_advisory_episodes_similarByEntity(t *testing.T) {
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
	cfg.Similarity.MinResemblance = 0.01
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

	tag := "rest:episodic-ent-" + uuid.New().String()[:8]
	bodyA := fmt.Sprintf(`{"summary":"deployment rollback discussed with platform team","source":"manual","tags":[%q],"entities":["platform-team","deploy"]}`, tag)
	bodyB := fmt.Sprintf(`{"summary":"unrelated snack break","source":"manual","tags":[%q],"entities":["kitchen"]}`, tag)
	for _, body := range []string{bodyA, bodyB} {
		resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create: %d %s", resp.StatusCode, string(b))
		}
	}

	simBody := fmt.Sprintf(`{"query":"deployment platform discussion","tags":[%q],"entity":"platform-team","max_results":10}`, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes/similar", "application/json", strings.NewReader(simBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("similar: %d %s", resp.StatusCode, string(b))
	}
	var sim struct {
		Cases []struct {
			Summary string `json:"summary"`
		} `json:"advisory_similar_cases"`
	}
	if err := json.Unmarshal(b, &sim); err != nil {
		t.Fatal(err)
	}
	if len(sim.Cases) < 1 || !strings.Contains(strings.ToLower(sim.Cases[0].Summary), "deployment") {
		t.Fatalf("expected platform episode first, got %#v", sim.Cases)
	}
}

// TestREST_enforcement_ignoresAdvisoryEpisodes proves binding evaluation uses durable memory rows, not advisory_episodes.
func TestREST_enforcement_ignoresAdvisoryEpisodes(t *testing.T) {
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

	// Advisory episode stating a constraint — not in memories table.
	epBody := `{"summary":"NEVER use deprecated API X under any circumstances","source":"manual","tags":["rest:episodic-enf"],"entities":["api-x"]}`
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(epBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("advisory create: %d %s", resp.StatusCode, string(b))
	}

	// Enforcement evaluates proposal against binding memory only; no memory row for that statement.
	reqBody := `{"proposal_text":"We will use deprecated API X for the migration.","tags":["rest:episodic-enf"]}`
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/enforcement/evaluate", bytes.NewReader([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	hc := &http.Client{}
	eres, err := hc.Do(req)
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
	// No governing constraint memory was created; advisory episode must not appear as triggered binding memory.
	if len(eval.TriggeredMemories) > 0 {
		t.Fatalf("expected no triggered memories from advisory-only episode, got %+v", eval.TriggeredMemories)
	}
}

func TestREST_advisory_episodes_similarRejectsInvertedWindow(t *testing.T) {
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

	simBody := `{"query":"anything","occurred_after":"2025-01-02T00:00:00Z","occurred_before":"2025-01-01T00:00:00Z"}`
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes/similar", "application/json", strings.NewReader(simBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("want 400 got %d body=%s", resp.StatusCode, string(b))
	}
}

// TestREST_advisory_episodes_similarByEntitiesArray uses entities[] (any overlap) like entity string.
func TestREST_advisory_episodes_similarByEntitiesArray(t *testing.T) {
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
	cfg.Similarity.MinResemblance = 0.01
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

	tag := "rest:episodic-ents-" + uuid.New().String()[:8]
	bodyA := fmt.Sprintf(`{"summary":"redis cache incident triage with vendor","source":"manual","tags":[%q],"entities":["redis","vendor-acme"]}`, tag)
	bodyB := fmt.Sprintf(`{"summary":"lunch","source":"manual","tags":[%q],"entities":["cafeteria"]}`, tag)
	for _, body := range []string{bodyA, bodyB} {
		resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		b, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("create: %d %s", resp.StatusCode, string(b))
		}
	}

	simBody := fmt.Sprintf(`{"query":"redis cache vendor incident","tags":[%q],"entities":["redis","other"],"max_results":10}`, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes/similar", "application/json", strings.NewReader(simBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("similar: %d %s", resp.StatusCode, string(b))
	}
	if !strings.Contains(string(b), "redis cache incident") {
		t.Fatalf("expected redis episode in body: %s", string(b))
	}
}

// TestREST_advisory_episodes_minimalBackwardCompat creates an episode without occurred_at or entities; similar still works.
func TestREST_advisory_episodes_minimalBackwardCompat(t *testing.T) {
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
	cfg.Similarity.MinResemblance = 0.01
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

	tag := "rest:episodic-min-" + uuid.New().String()[:8]
	body := fmt.Sprintf(`{"summary":"MINIMAL EPISODE WORDS UNIQUE XYZ","source":"manual","tags":[%q]}`, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", resp.StatusCode, string(b))
	}
	if !strings.Contains(string(b), `"entities":[]`) {
		t.Fatalf("expected empty entities array in response: %s", string(b))
	}

	simBody := fmt.Sprintf(`{"query":"MINIMAL EPISODE WORDS UNIQUE","tags":[%q],"max_results":5}`, tag)
	resp, err = http.Post(srv.URL+"/v1/advisory-episodes/similar", "application/json", strings.NewReader(simBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("similar: %d %s", resp.StatusCode, string(b))
	}
	if !strings.Contains(string(b), "MINIMAL EPISODE WORDS UNIQUE XYZ") {
		t.Fatalf("expected minimal episode in results: %s", string(b))
	}
}

// TestREST_recallCompile_excludesAdvisoryEpisodeText ensures canonical recall does not embed advisory_episodes rows.
func TestREST_recallCompile_excludesAdvisoryEpisodeText(t *testing.T) {
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

	tag := "rest:episodic-recall-" + uuid.New().String()[:8]
	unique := "ADVISORY ONLY NOT IN CANON " + uuid.New().String()
	epBody := fmt.Sprintf(`{"summary":%q,"source":"manual","tags":[%q]}`, unique, tag)
	resp, err := http.Post(srv.URL+"/v1/advisory-episodes", "application/json", strings.NewReader(epBody))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: %d %s", resp.StatusCode, string(b))
	}

	// Retrieval query must not echo the advisory-only phrase (bundle may echo retrieval_query).
	compileBody := fmt.Sprintf(`{"retrieval_query":"generic compile probe","tags":[%q],"max_per_kind":8,"max_total":40}`, tag)
	req, _ := http.NewRequest(http.MethodPost, srv.URL+"/v1/recall/compile", strings.NewReader(compileBody))
	req.Header.Set("Content-Type", "application/json")
	hc := &http.Client{}
	cre, err := hc.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	cb, _ := io.ReadAll(cre.Body)
	_ = cre.Body.Close()
	if cre.StatusCode != http.StatusOK {
		t.Fatalf("compile: %d %s", cre.StatusCode, string(cb))
	}
	if strings.Contains(string(cb), unique) {
		t.Fatalf("recall bundle must not contain advisory episode statement")
	}
}
