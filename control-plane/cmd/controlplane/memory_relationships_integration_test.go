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

func bootMemoryRelServer(t *testing.T, dsn string) (*httptest.Server, func()) {
	t.Helper()
	cfg, err := app.LoadConfig(integrationConfigPath(t))
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Postgres.DSN = dsn
	container, err := app.Boot(cfg)
	if err != nil {
		t.Fatalf("boot: %v", err)
	}
	rtr, err := apiserver.NewRouter(cfg, container)
	if err != nil {
		container.DB.Close()
		t.Fatalf("router: %v", err)
	}
	srv := httptest.NewServer(rtr)
	return srv, func() {
		srv.Close()
		container.DB.Close()
	}
}

func postMemory(t *testing.T, base, kind, stmt, tag string) string {
	t.Helper()
	body := fmt.Sprintf(`{"kind":%q,"statement":%q,"authority":6,"tags":[%q]}`,
		kind, stmt, tag)
	resp, err := http.Post(base+"/v1/memory", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create memory: %d %s", resp.StatusCode, string(b))
	}
	var out struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	return out.ID
}

// TestREST_memoryRelationships_createAndList proves POST/GET relationship endpoints.
func TestREST_memoryRelationships_createAndList(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootMemoryRelServer(t, dsn)
	defer cleanup()
	tag := "rest:rel:" + uuid.New().String()[:8]
	a := postMemory(t, srv.URL, "pattern", "REL A "+tag+" always validate config before deploy", tag)
	b := postMemory(t, srv.URL, "pattern", "REL B "+tag+" related rollout variant same family", tag)

	relBody := fmt.Sprintf(`{"from_memory_id":%q,"to_memory_id":%q,"relationship_type":"same_pattern_family","reason":"test","source":"integration"}`,
		b, a)
	resp, err := http.Post(srv.URL+"/v1/memory/relationships", "application/json", strings.NewReader(relBody))
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("relationship create: %d %s", resp.StatusCode, string(rb))
	}

	gr, err := http.Get(srv.URL + "/v1/memory/" + a + "/relationships")
	if err != nil {
		t.Fatal(err)
	}
	gb, _ := io.ReadAll(gr.Body)
	_ = gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get relationships: %d %s", gr.StatusCode, string(gb))
	}
	if !strings.Contains(string(gb), "same_pattern_family") {
		t.Fatalf("expected same_pattern_family in response: %s", string(gb))
	}
}

// TestREST_memoryRelationships_supersedesEdgeOnCreate proves supersedes_id creates a typed edge.
func TestREST_memoryRelationships_supersedesEdgeOnCreate(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootMemoryRelServer(t, dsn)
	defer cleanup()
	tag := "rest:sup:" + uuid.New().String()[:8]
	oldID := postMemory(t, srv.URL, "decision", "SUP OLD "+tag+" use feature flags for risky deploys", tag)
	body := fmt.Sprintf(`{"kind":"decision","statement":%q,"authority":7,"tags":[%q],"supersedes_id":%q}`,
		"SUP NEW "+tag+" use staged rollouts with flags", tag, oldID)
	resp, err := http.Post(srv.URL+"/v1/memory", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("create with supersedes: %d %s", resp.StatusCode, string(b))
	}
	var out struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(b, &out)

	gr, err := http.Get(srv.URL + "/v1/memory/" + oldID + "/relationships")
	if err != nil {
		t.Fatal(err)
	}
	gb, _ := io.ReadAll(gr.Body)
	_ = gr.Body.Close()
	if gr.StatusCode != http.StatusOK {
		t.Fatalf("get relationships: %d %s", gr.StatusCode, string(gb))
	}
	if !strings.Contains(string(gb), `"relationship_type":"supersedes"`) && !strings.Contains(string(gb), `"relationship_type": "supersedes"`) {
		t.Fatalf("expected supersedes inbound on old memory: %s", string(gb))
	}
	// Idempotent second POST same edge
	dupPayload := fmt.Sprintf(`{"from_memory_id":%q,"to_memory_id":%q,"relationship_type":"supersedes","source":"dup_test"}`, out.ID, oldID)
	resp2, err := http.Post(srv.URL+"/v1/memory/relationships", "application/json", strings.NewReader(dupPayload))
	if err != nil {
		t.Fatal(err)
	}
	b2, _ := io.ReadAll(resp2.Body)
	_ = resp2.Body.Close()
	if resp2.StatusCode != http.StatusCreated {
		t.Fatalf("duplicate relationship: %d %s", resp2.StatusCode, string(b2))
	}
}

// TestREST_memoryRelationships_contradicts verifies contradicts edge.
func TestREST_memoryRelationships_contradicts(t *testing.T) {
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		t.Skip("TEST_PG_DSN not set")
	}
	srv, cleanup := bootMemoryRelServer(t, dsn)
	defer cleanup()
	tag := "rest:con:" + uuid.New().String()[:8]
	x := postMemory(t, srv.URL, "constraint", "CON X "+tag+" never run migrations on friday", tag)
	y := postMemory(t, srv.URL, "constraint", "CON Y "+tag+" allow migrations on friday with approval", tag)
	body := fmt.Sprintf(`{"from_memory_id":%q,"to_memory_id":%q,"relationship_type":"contradicts","source":"integration"}`, x, y)
	resp, err := http.Post(srv.URL+"/v1/memory/relationships", "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	rb, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("contradicts: %d %s", resp.StatusCode, string(rb))
	}
}
