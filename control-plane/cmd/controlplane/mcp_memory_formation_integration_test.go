//go:build integration
// +build integration

package main

import (
	"bufio"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"control-plane/internal/apiserver"
	"control-plane/internal/app"
	"control-plane/internal/migrate"

	_ "github.com/lib/pq"
)

func bootMCPProof(t *testing.T, dsn string, autoDistill bool) (*httptest.Server, *sql.DB, func()) {
	t.Helper()
	// go test runs TestREST_* before TestIntegration_*; REST tests (e.g. controlled promotion) may
	// materialize memories. MCP proof tests expect a clean memories table — reset when the regression
	// harness allows it (same as internal/eval proof_clean).
	if strings.TrimSpace(os.Getenv("TEST_PG_RESET_SCHEMA")) == "1" {
		pgdb, err := sql.Open("postgres", dsn)
		if err != nil {
			t.Fatalf("open postgres for integration reset: %v", err)
		}
		if err := migrate.MaybeResetPublicSchemaForIntegrationTests(context.Background(), pgdb); err != nil {
			_ = pgdb.Close()
			t.Fatalf("reset schema: %v", err)
		}
		_ = pgdb.Close()
	}
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	cfg.Similarity.Enabled = true
	cfg.Similarity.MinResemblance = 0.05
	cfg.Distillation.Enabled = true
	cfg.Distillation.AutoFromAdvisoryEpisodes = autoDistill
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
	return srv, container.DB, func() {
		srv.Close()
		container.DB.Close()
	}
}

func mcpCall(t *testing.T, base string, method string, params any) map[string]any {
	t.Helper()
	body := map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
	raw, _ := json.Marshal(body)
	resp, err := http.Post(strings.TrimRight(base, "/")+"/v1/mcp", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("mcp post: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("mcp http %d: %s", resp.StatusCode, string(b))
	}
	var out map[string]any
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatalf("mcp json: %v body=%s", err, string(b))
	}
	if errObj, ok := out["error"].(map[string]any); ok && errObj != nil {
		t.Fatalf("mcp error: %v", out)
	}
	return out
}

func mcpToolText(t *testing.T, base string, tool string, args map[string]any) (text string, isErr bool) {
	t.Helper()
	res := mcpCall(t, base, "tools/call", map[string]any{"name": tool, "arguments": args})
	rmap, _ := res["result"].(map[string]any)
	if rmap == nil {
		t.Fatalf("no result: %v", res)
	}
	if ie, ok := rmap["isError"].(bool); ok && ie {
		isErr = true
	}
	content, _ := rmap["content"].([]any)
	if len(content) == 0 {
		return "", isErr
	}
	c0, _ := content[0].(map[string]any)
	text, _ = c0["text"].(string)
	return text, isErr
}

func countMemories(t *testing.T, db *sql.DB) int {
	t.Helper()
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM memories`).Scan(&n); err != nil {
		t.Fatalf("count memories: %v", err)
	}
	return n
}

func countAdvisoryBySummary(t *testing.T, db *sql.DB, summary string) int {
	t.Helper()
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM advisory_episodes WHERE summary_text = $1 AND source = 'mcp'`, summary).Scan(&n)
	if err != nil {
		t.Fatalf("count advisory: %v", err)
	}
	return n
}

// TestIntegration_HTTP_MCP_initializeAndToolsList exercises JSON-RPC discovery on POST /v1/mcp.
func TestIntegration_HTTP_MCP_initializeAndToolsList(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, _, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL

	init := mcpCall(t, base, "initialize", map[string]any{})
	res, _ := init["result"].(map[string]any)
	if res["serverInfo"] == nil {
		t.Fatalf("initialize: %v", init)
	}

	list := mcpCall(t, base, "tools/list", map[string]any{})
	toolsRaw, _ := list["result"].(map[string]any)["tools"]
	tools, _ := toolsRaw.([]any)
	found := map[string]bool{}
	for _, row := range tools {
		m, _ := row.(map[string]any)
		n, _ := m["name"].(string)
		found[n] = true
	}
	for _, name := range []string{"recall_context", "record_experience", "mcp_episode_ingest", "memory_context_resolve", "curation_pending", "curation_promotion_suggestions", "curation_strengthened"} {
		if !found[name] {
			t.Fatalf("tools/list missing %q", name)
		}
	}
}

// TestIntegration_HTTP_MCP_parityToolsRegistered ensures agent-parity tools appear in tools/list (MCP parity sprint).
func TestIntegration_HTTP_MCP_parityToolsRegistered(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, _, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL
	mcpCall(t, base, "initialize", map[string]any{})
	list := mcpCall(t, base, "tools/list", map[string]any{})
	toolsRaw, _ := list["result"].(map[string]any)["tools"]
	tools, _ := toolsRaw.([]any)
	found := map[string]bool{}
	for _, row := range tools {
		m, _ := row.(map[string]any)
		n, _ := m["name"].(string)
		found[n] = true
	}
	for _, name := range []string{
		"memory_context_resolve", "memory_log_if_relevant", "auto_log_episode_if_relevant",
		"episode_search_similar", "episode_distill_explicit",
		"memory_recall_advanced", "memory_preflight_check",
		"curation_review_candidate", "curation_reject_candidate", "curation_auto_promote", "curation_promote_candidate",
		"memory_detect_contradictions", "memory_list_contradictions",
		"evidence_attach", "evidence_list",
		"memory_relationships_get", "memory_relationships_create",
	} {
		if !found[name] {
			t.Fatalf("tools/list missing %q", name)
		}
	}
}

// TestIntegration_HTTP_MCP_episodeAutoDistillAndCuration proves MCP-origin → candidate → visibility tools; no canon without materialize.
func TestIntegration_HTTP_MCP_episodeAutoDistillAndCuration(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, db, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL

	tag := "mcp-proof:" + fmt.Sprintf("%d", os.Getpid())
	summary := fmt.Sprintf(`MCP proof %s deployment failure error required rollback decision after timeout incident during peak traffic`, tag)

	txt, errCall := mcpToolText(t, base, "mcp_episode_ingest", map[string]any{
		"summary":        summary,
		"correlation_id": "sess-proof-1",
		"event_kind":     "failure",
	})
	if errCall {
		t.Fatalf("mcp_episode_ingest isError: %s", txt)
	}
	var ep map[string]any
	if err := json.Unmarshal([]byte(txt), &ep); err != nil {
		t.Fatalf("parse episode json: %v text=%s", err, txt)
	}
	if ep["source"] != "mcp" {
		t.Fatalf("want source mcp got %v", ep["source"])
	}
	if ep["deduplicated"] == true {
		t.Fatalf("unexpected deduplicated on first ingest")
	}

	memBefore := countMemories(t, db)
	if memBefore != 0 {
		t.Fatalf("expected zero canonical memories in fresh proof DB, got %d", memBefore)
	}

	pend, _ := mcpToolText(t, base, "curation_pending", map[string]any{})
	if !strings.Contains(pend, tag) {
		t.Fatalf("expected pending to reflect distill; snippet=%s", pend[:min(800, len(pend))])
	}
	if !strings.Contains(pend, "auto:mcp") {
		t.Fatalf("expected auto:mcp origin in pending: %s", pend[:min(1200, len(pend))])
	}
	if !strings.Contains(pend, "origin:mcp") {
		t.Fatalf("expected origin:mcp tag in candidate payload: %s", pend[:min(1200, len(pend))])
	}

	_, _ = mcpToolText(t, base, "curation_promotion_suggestions", map[string]any{})
	_, _ = mcpToolText(t, base, "curation_strengthened", map[string]any{"min_support": 2})

	if countMemories(t, db) != 0 {
		t.Fatalf("MCP path must not create canonical memory without materialize")
	}
}

// TestIntegration_HTTP_MCP_noAutoDistillNoCandidate scenario C: episode exists, no implicit candidate when auto off.
func TestIntegration_HTTP_MCP_noAutoDistillNoCandidate(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, db, cleanup := bootMCPProof(t, dsn, false)
	defer cleanup()
	base := srv.URL

	tag := "mcp-proof-nodist:" + fmt.Sprintf("%d", os.Getpid())
	summary := fmt.Sprintf(`MCP proof no auto distill %s failure error timeout regression during checkout flow learning constraint`, tag)

	txt, errCall := mcpToolText(t, base, "mcp_episode_ingest", map[string]any{"summary": summary})
	if errCall {
		t.Fatalf("ingest: %s", txt)
	}
	var ep map[string]any
	_ = json.Unmarshal([]byte(txt), &ep)
	if ep["id"] == nil {
		t.Fatalf("expected episode id: %s", txt)
	}

	pend, _ := mcpToolText(t, base, "curation_pending", map[string]any{})
	if strings.Contains(pend, tag) {
		t.Fatalf("did not expect pending candidate containing proof tag when auto-distill off: %s", pend[:min(600, len(pend))])
	}
	_ = db
}

// TestIntegration_HTTP_MCP_dedupRepeatedIngest scenario D: duplicate MCP ingest returns deduplicated; single advisory row.
func TestIntegration_HTTP_MCP_dedupRepeatedIngest(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, db, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL

	tag := "mcp-proof-dedup:" + fmt.Sprintf("%d", os.Getpid())
	summary := fmt.Sprintf(`MCP dedup proof %s failure error duplicate retry loop violation during deploy decision fix`, tag)
	args := map[string]any{"summary": summary, "correlation_id": "dedup-sess-1"}

	t1, err1 := mcpToolText(t, base, "mcp_episode_ingest", args)
	if err1 {
		t.Fatalf("first ingest: %s", t1)
	}
	t2, err2 := mcpToolText(t, base, "mcp_episode_ingest", args)
	if err2 {
		t.Fatalf("second ingest: %s", t2)
	}
	var ep1, ep2 map[string]any
	_ = json.Unmarshal([]byte(t1), &ep1)
	_ = json.Unmarshal([]byte(t2), &ep2)
	if ep1["id"] == nil || ep2["id"] == nil {
		t.Fatalf("expected ids: %v / %v", ep1, ep2)
	}
	if ep1["id"] != ep2["id"] {
		t.Fatalf("dedup should return same id: %v vs %v", ep1["id"], ep2["id"])
	}
	if ep2["deduplicated"] != true {
		t.Fatalf("second response should be deduplicated: %v", ep2)
	}
	if n := countAdvisoryBySummary(t, db, summary); n != 1 {
		t.Fatalf("want 1 advisory row, got %d", n)
	}
}

// TestIntegration_stdio_pluribusMcp_smoke runs cmd/pluribus-mcp against a live control-plane (secondary proof surface).
func TestIntegration_stdio_pluribusMcp_smoke(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stdio smoke in -short")
	}
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, _, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()

	root := controlPlaneModuleRoot(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/pluribus-mcp")
	cmd.Dir = root
	cmd.Env = append(os.Environ(), "CONTROL_PLANE_URL="+strings.TrimRight(srv.URL, "/"))
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("start stdio mcp: %v", err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()

	line3, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0", "id": float64(3), "method": "tools/call",
		"params": map[string]any{
			"name": "mcp_episode_ingest",
			"arguments": map[string]any{
				"summary":        "stdio smoke test failure error decision rollback incident during deploy learning constraint fix",
				"correlation_id": "stdio-smoke",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
		string(line3),
	}
	for _, line := range lines {
		if _, err := io.WriteString(stdin, line+"\n"); err != nil {
			t.Fatalf("write stdin: %v", err)
		}
	}
	_ = stdin.Close()

	sc := bufio.NewScanner(stdout)
	var decoded []map[string]any
	for sc.Scan() && len(decoded) < 3 {
		var m map[string]any
		if err := json.Unmarshal(sc.Bytes(), &m); err != nil {
			t.Fatalf("stdout json: %v line=%q", err, sc.Text())
		}
		decoded = append(decoded, m)
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan stdout: %v", err)
	}
	if len(decoded) < 3 {
		t.Fatalf("expected 3 json lines, got %d", len(decoded))
	}
	if decoded[0]["result"] == nil {
		t.Fatalf("initialize: %v", decoded[0])
	}
	if decoded[1]["result"] == nil {
		t.Fatalf("tools/list: %v", decoded[1])
	}
	res3, _ := decoded[2]["result"].(map[string]any)
	if res3 == nil {
		t.Fatalf("tools/call: %v", decoded[2])
	}
	if res3["isError"] == true {
		t.Fatalf("tools/call error: %v", decoded[2])
	}
}

// TestIntegration_HTTP_MCP_memoryContextResolve scenario B (partial): primary recall tool returns mcp_context + recall_bundle JSON.
func TestIntegration_HTTP_MCP_memoryContextResolve(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, _, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL

	txt, errCall := mcpToolText(t, base, "memory_context_resolve", map[string]any{
		"task_description": "We must not skip policy review before deploy due to prior timeout incident",
	})
	if errCall {
		t.Fatalf("memory_context_resolve isError: %s", txt)
	}
	var wrap map[string]any
	if err := json.Unmarshal([]byte(txt), &wrap); err != nil {
		t.Fatalf("parse json: %v text=%s", err, txt[:min(400, len(txt))])
	}
	if wrap["mcp_context"] == nil || wrap["recall_bundle"] == nil {
		t.Fatalf("expected mcp_context and recall_bundle: %v", wrap)
	}
	mc, _ := wrap["mcp_context"].(map[string]any)
	if mc == nil {
		t.Fatalf("mcp_context not object: %v", wrap["mcp_context"])
	}
	if mc["memory_signal_priority"] == nil || mc["why"] == nil {
		t.Fatalf("expected enriched mcp_context fields: %v", mc)
	}
	if mc["why_now"] == nil && mc["bundle_counts"] == nil {
		t.Fatalf("expected why_now or bundle_counts on mcp_context: %v", mc)
	}
}

// TestIntegration_HTTP_MCP_memoryLogRepeatedPhrase proves auto_log heuristics ingest without explicit failure keywords.
func TestIntegration_HTTP_MCP_memoryLogRepeatedPhrase(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, _, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL

	txt, errCall := mcpToolText(t, base, "memory_log_if_relevant", map[string]any{
		"text_block": strings.Repeat("pad ", 5) + " we hit the same regression again and again during the rollout " + strings.Repeat("z", 12),
	})
	if errCall {
		t.Fatalf("memory_log_if_relevant isError: %s", txt)
	}
	if !strings.Contains(txt, "ingested") && !strings.Contains(txt, `"source":"mcp"`) {
		t.Fatalf("expected ingest from repetition heuristic: %s", txt[:min(500, len(txt))])
	}
}

// TestIntegration_HTTP_MCP_memoryLogIfRelevant deterministic skip vs ingest (scenario C helper: no manual episode tool naming).
func TestIntegration_HTTP_MCP_memoryLogIfRelevant(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, _, cleanup := bootMCPProof(t, dsn, true)
	defer cleanup()
	base := srv.URL

	skipTxt, errCall := mcpToolText(t, base, "memory_log_if_relevant", map[string]any{
		"text_block": "hello world nothing special here " + strings.Repeat("x", 20),
	})
	if errCall {
		t.Fatalf("skip case isError: %s", skipTxt)
	}
	if !strings.Contains(skipTxt, "skipped") {
		t.Fatalf("expected skipped: %s", skipTxt)
	}

	ingestTxt, errIngest := mcpToolText(t, base, "memory_log_if_relevant", map[string]any{
		"text_block": "deployment failure error timeout incident during peak traffic decision rollback constraint fix learning",
	})
	if errIngest {
		t.Fatalf("ingest case isError: %s", ingestTxt)
	}
	if !strings.Contains(ingestTxt, "ingested") && !strings.Contains(ingestTxt, `"source":"mcp"`) {
		t.Fatalf("expected ingest markers: %s", ingestTxt[:min(600, len(ingestTxt))])
	}
}
